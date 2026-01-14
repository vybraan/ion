package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"ion/audio"
	"ion/protocol"
)

const (
	defaultSampleRate = 16000
	defaultChannels   = 1
	defaultFormat     = "s16le"
)

type serverConfig struct {
	sampleRate             int
	channels               int
	format                 string
	asrBackend             string
	whisperCLI             string
	whisperModel           string
	whisperPartialInterval time.Duration
	whisperPartialWindow   time.Duration
}

var cfg serverConfig

type connState struct {
	out   *bufio.Writer
	outMu sync.Mutex

	streamMu   sync.Mutex
	streamOn   bool
	streamStop chan struct{}
	capture    *audio.PipewireCapture

	ttsMu   sync.Mutex
	ttsStop chan struct{}

	asrMu     sync.Mutex
	asrOn     bool
	asrBuffer []byte
	asrLang   string

	asrPartialStop chan struct{}
	asrRunning     bool
	asrLastPartial string
}

func main() {
	transport := flag.String("transport", "tcp", "tcp or stdio")
	addr := flag.String("addr", ":10300", "tcp listen address")
	sampleRate := flag.Int("sample-rate", defaultSampleRate, "sample rate for ready/capture")
	channels := flag.Int("channels", defaultChannels, "channel count for ready/capture")
	asrBackend := flag.String("asr", "mock", "asr backend: mock or whisper")
	whisperCLI := flag.String("whisper-cli", "", "path to whisper-cli")
	whisperModel := flag.String("whisper-model", "", "path to whisper model")
	whisperPartials := flag.Duration("whisper-partial-interval", 1*time.Second, "interval for whisper partials")
	whisperWindow := flag.Duration("whisper-partial-window", 6*time.Second, "audio window for whisper partials")
	flag.Parse()

	cfg = serverConfig{
		sampleRate:             *sampleRate,
		channels:               *channels,
		format:                 defaultFormat,
		asrBackend:             *asrBackend,
		whisperCLI:             *whisperCLI,
		whisperModel:           *whisperModel,
		whisperPartialInterval: *whisperPartials,
		whisperPartialWindow:   *whisperWindow,
	}

	if cfg.asrBackend == "whisper" {
		if cfg.whisperCLI == "" || cfg.whisperModel == "" {
			log.Fatal("whisper backend requires --whisper-cli and --whisper-model")
		}
		if cfg.whisperPartialInterval <= 0 {
			cfg.whisperPartialInterval = 2 * time.Second
		}
		if cfg.whisperPartialWindow <= 0 {
			cfg.whisperPartialWindow = 6 * time.Second
		}
	}

	switch *transport {
	case "tcp":
		ln, err := net.Listen("tcp", *addr)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("demo server listening on", *addr)
		for {
			c, err := ln.Accept()
			if err != nil {
				continue
			}
			go handleConn(bufio.NewReader(c), bufio.NewWriter(c), c.Close)
		}
	case "stdio":
		handleConn(bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout), func() error { return nil })
	default:
		log.Fatalf("unknown transport: %s", *transport)
	}
}

func handleConn(in *bufio.Reader, out *bufio.Writer, closer func() error) {
	defer closer()

	state := &connState{out: out}

	for {
		f, err := protocol.ReadFrame(in)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			log.Println("read frame:", err)
			return
		}

		switch f.Type {
		case protocol.FrameTypeJSON:
			if err := handleJSON(state, f.Payload); err != nil {
				log.Println("handleJSON:", err)
				return
			}
		case protocol.FrameTypeAudio:
			handleAudio(state, f.Payload)
		default:
			// ignore unknown
		}
	}
}

func handleJSON(state *connState, payload []byte) error {
	var base protocol.BaseEvent
	if err := protocol.Decode(payload, &base); err != nil {
		return err
	}

	switch base.Type {
	case protocol.EventDescribe:
		return sendReady(state)
	case protocol.EventStart:
		startStream(state)
	case protocol.EventStop:
		stopStream(state)
	case protocol.EventASRStart:
		var ev protocol.ASRStartEvent
		if err := protocol.Decode(payload, &ev); err != nil {
			return err
		}
		startASR(state, ev.Language)
	case protocol.EventASRStop:
		stopASR(state)
	case protocol.EventTTSStart:
		var ev protocol.TTSStartEvent
		if err := protocol.Decode(payload, &ev); err != nil {
			return err
		}
		startTTS(state, ev.Text)
	case protocol.EventTTSStop:
		stopTTS(state)
	default:
		return nil
	}
	return nil
}

func sendReady(state *connState) error {
	ev := protocol.ReadyEvent{
		Type:       protocol.EventReady,
		Protocol:   "ion",
		SampleRate: cfg.sampleRate,
		Channels:   cfg.channels,
		Format:     cfg.format,
	}
	return writeJSON(state, ev)
}

func writeJSON(state *connState, ev any) error {
	data, err := protocol.Encode(ev)
	if err != nil {
		return err
	}
	frame := &protocol.Frame{
		Version: protocol.VersionByte,
		Type:    protocol.FrameTypeJSON,
		Length:  uint32(len(data)),
		Payload: data,
	}
	return writeFrame(state, frame)
}

func writeFrame(state *connState, frame *protocol.Frame) error {
	state.outMu.Lock()
	defer state.outMu.Unlock()

	if err := protocol.WriteFrame(state.out, frame); err != nil {
		return err
	}
	return state.out.Flush()
}

func startStream(state *connState) {
	state.streamMu.Lock()
	defer state.streamMu.Unlock()
	if state.streamOn {
		return
	}
	state.streamOn = true
	state.streamStop = make(chan struct{})
	cap, err := audio.NewPipewireCapture(cfg.sampleRate, cfg.channels)
	if err != nil {
		log.Println("pipewire capture:", err)
		state.streamOn = false
		state.streamStop = nil
		return
	}
	state.capture = cap
	go captureLoop(state, state.streamStop)
}

func stopStream(state *connState) {
	state.streamMu.Lock()
	defer state.streamMu.Unlock()
	if !state.streamOn {
		return
	}
	close(state.streamStop)
	state.streamOn = false
	state.streamStop = nil
	if state.capture != nil {
		_ = state.capture.Close()
		state.capture = nil
	}
}

func startTTS(state *connState, text string) {
	state.ttsMu.Lock()
	defer state.ttsMu.Unlock()
	if state.ttsStop != nil {
		close(state.ttsStop)
	}
	state.ttsStop = make(chan struct{})
	go ttsLoop(state, state.ttsStop, text)
}

func stopTTS(state *connState) {
	state.ttsMu.Lock()
	defer state.ttsMu.Unlock()
	if state.ttsStop != nil {
		close(state.ttsStop)
		state.ttsStop = nil
	}
}

func startASR(state *connState, language string) {
	state.asrMu.Lock()
	state.asrOn = true
	state.asrBuffer = nil
	state.asrLang = strings.TrimSpace(language)
	state.asrLastPartial = ""
	if state.asrPartialStop != nil {
		close(state.asrPartialStop)
	}
	state.asrPartialStop = make(chan struct{})
	state.asrMu.Unlock()

	_ = writeJSON(state, protocol.ASRPartialEvent{
		Type: protocol.EventASRPartial,
		Text: "listening...",
	})

	if cfg.asrBackend == "whisper" {
		go asrPartialLoop(state, state.asrPartialStop)
	}
}

func stopASR(state *connState) {
	state.asrMu.Lock()
	if state.asrPartialStop != nil {
		close(state.asrPartialStop)
		state.asrPartialStop = nil
	}
	state.asrOn = false
	buffer := append([]byte(nil), state.asrBuffer...)
	lang := state.asrLang
	state.asrMu.Unlock()

	if cfg.asrBackend == "whisper" {
		go func() {
			text, err := runWhisper(buffer, lang)
			if err != nil {
				_ = writeJSON(state, protocol.ASRErrorEvent{
					Type:    protocol.EventASRError,
					Message: err.Error(),
				})
				return
			}
			if text == "" {
				return
			}
			_ = writeJSON(state, protocol.ASRResultEvent{
				Type: protocol.EventASRResult,
				Text: text,
			})
		}()
	} else {
		_ = writeJSON(state, protocol.ASRResultEvent{
			Type: protocol.EventASRResult,
			Text: "demo transcript (replace with Whisper)",
		})
	}
}

func handleAudio(state *connState, payload []byte) {
	state.asrMu.Lock()
	on := state.asrOn
	if on {
		state.asrBuffer = append(state.asrBuffer, payload...)
		if cfg.whisperPartialWindow > 0 {
			maxBytes := int(cfg.whisperPartialWindow.Seconds()) * cfg.sampleRate * cfg.channels * 2 * 2
			if maxBytes > 0 && len(state.asrBuffer) > maxBytes {
				state.asrBuffer = state.asrBuffer[len(state.asrBuffer)-maxBytes:]
			}
		}
	}
	state.asrMu.Unlock()
}

func asrPartialLoop(state *connState, stop <-chan struct{}) {
	ticker := time.NewTicker(cfg.whisperPartialInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		}

		state.asrMu.Lock()
		if !state.asrOn || state.asrRunning {
			state.asrMu.Unlock()
			continue
		}
		buffer := append([]byte(nil), state.asrBuffer...)
		lang := state.asrLang
		state.asrRunning = true
		state.asrMu.Unlock()

		minBytes := cfg.sampleRate * cfg.channels * 2 / 2
		if len(buffer) < minBytes {
			state.asrMu.Lock()
			state.asrRunning = false
			state.asrMu.Unlock()
			continue
		}

		window := buffer
		if cfg.whisperPartialWindow > 0 {
			windowBytes := int(cfg.whisperPartialWindow.Seconds()) * cfg.sampleRate * cfg.channels * 2
			if windowBytes > 0 && len(window) > windowBytes {
				window = window[len(window)-windowBytes:]
			}
		}

		text, err := runWhisper(window, lang)

		state.asrMu.Lock()
		state.asrRunning = false
		if !state.asrOn || err != nil || text == "" || text == state.asrLastPartial {
			state.asrMu.Unlock()
			if err != nil {
				_ = writeJSON(state, protocol.ASRErrorEvent{
					Type:    protocol.EventASRError,
					Message: err.Error(),
				})
			}
			continue
		}
		state.asrLastPartial = text
		state.asrMu.Unlock()

		_ = writeJSON(state, protocol.ASRPartialEvent{
			Type: protocol.EventASRPartial,
			Text: text,
		})
	}
}

func runWhisper(pcm []byte, lang string) (string, error) {
	if cfg.whisperCLI == "" || cfg.whisperModel == "" {
		return "", nil
	}
	if len(pcm) == 0 {
		return "", nil
	}
	samples := bytesToInt16(pcm)
	if cfg.channels > 1 {
		samples = downmixMono(samples, cfg.channels)
	}
	if cfg.sampleRate != defaultSampleRate {
		samples = resampleLinear(samples, cfg.sampleRate, defaultSampleRate)
	}

	tmp, err := os.CreateTemp("", "ion-whisper-*.wav")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if err := writeWAV(tmp, samples, defaultSampleRate, 1); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	args := []string{"-m", cfg.whisperModel, "-f", tmpPath, "--no-timestamps"}
	if strings.TrimSpace(lang) != "" {
		args = append(args, "-l", lang)
	}
	out, err := exec.Command(cfg.whisperCLI, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", fmt.Errorf("whisper exec: %w", err)
		}
		return "", fmt.Errorf("whisper exec: %w: %s", err, msg)
	}
	return extractTranscript(string(out)), nil
}

func extractTranscript(output string) string {
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "whisper_") || strings.HasPrefix(line, "main:") || strings.HasPrefix(line, "system_info:") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.Contains(line, "-->") {
			continue
		}
		return line
	}
	return ""
}

func bytesToInt16(pcm []byte) []int16 {
	if len(pcm)%2 != 0 {
		pcm = pcm[:len(pcm)-1]
	}
	out := make([]int16, len(pcm)/2)
	for i := 0; i < len(out); i++ {
		out[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	return out
}

func downmixMono(samples []int16, channels int) []int16 {
	if channels <= 1 {
		return samples
	}
	frames := len(samples) / channels
	out := make([]int16, frames)
	for i := 0; i < frames; i++ {
		sum := 0
		for ch := 0; ch < channels; ch++ {
			sum += int(samples[i*channels+ch])
		}
		out[i] = int16(sum / channels)
	}
	return out
}

func resampleLinear(samples []int16, inRate, outRate int) []int16 {
	if inRate == outRate || len(samples) == 0 {
		return samples
	}
	ratio := float64(inRate) / float64(outRate)
	outLen := int(math.Round(float64(len(samples)) / ratio))
	if outLen <= 1 {
		return samples
	}
	out := make([]int16, outLen)
	for i := 0; i < outLen; i++ {
		pos := float64(i) * ratio
		idx := int(pos)
		if idx >= len(samples)-1 {
			out[i] = samples[len(samples)-1]
			continue
		}
		frac := pos - float64(idx)
		s0 := float64(samples[idx])
		s1 := float64(samples[idx+1])
		out[i] = int16(s0*(1-frac) + s1*frac)
	}
	return out
}

func writeWAV(w io.Writer, samples []int16, rate, channels int) error {
	dataSize := uint32(len(samples) * 2)
	if _, err := w.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(36)+dataSize); err != nil {
		return err
	}
	if _, err := w.Write([]byte("WAVE")); err != nil {
		return err
	}
	if _, err := w.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(channels)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(rate)); err != nil {
		return err
	}
	byteRate := uint32(rate * channels * 2)
	if err := binary.Write(w, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	blockAlign := uint16(channels * 2)
	if err := binary.Write(w, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(16)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, dataSize); err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, samples)
}

func captureLoop(state *connState, stop <-chan struct{}) {
	framesPerChunk := cfg.sampleRate / 50
	frameSize := 2 * cfg.channels
	buf := make([]byte, framesPerChunk*frameSize)

	for {
		select {
		case <-stop:
			return
		default:
		}

		state.streamMu.Lock()
		cap := state.capture
		state.streamMu.Unlock()
		if cap == nil {
			return
		}

		n, err := cap.Read(buf)
		if n > 0 {
			frame := &protocol.Frame{
				Version: protocol.VersionByte,
				Type:    protocol.FrameTypeAudio,
				Length:  uint32(n),
				Payload: buf[:n],
			}
			if werr := writeFrame(state, frame); werr != nil {
				log.Println("write audio frame:", werr)
				return
			}
		}
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			log.Println("capture read:", err)
			return
		}
	}
}

func ttsLoop(state *connState, stop <-chan struct{}, text string) {
	_ = writeJSON(state, protocol.TTSReadyEvent{Type: protocol.EventTTSReady})

	length := 0.6 + float64(len(text))*0.04
	if length < 0.6 {
		length = 0.6
	}
	if length > 4.0 {
		length = 4.0
	}

	framesPerChunk := cfg.sampleRate / 50
	frameSize := 2 * cfg.channels
	buf := make([]byte, framesPerChunk*frameSize)
	phase := 0.0
	step := (2 * math.Pi * 660.0) / float64(cfg.sampleRate)
	chunks := int(math.Ceil(length / 0.02))

	for i := 0; i < chunks; i++ {
		select {
		case <-stop:
			return
		default:
		}
		for j := 0; j < framesPerChunk; j++ {
			val := int16(math.Sin(phase) * 0.2 * 32767)
			binary.LittleEndian.PutUint16(buf[j*2:], uint16(val))
			phase += step
			if phase > 2*math.Pi {
				phase -= 2 * math.Pi
			}
		}
		frame := &protocol.Frame{
			Version: protocol.VersionByte,
			Type:    protocol.FrameTypeAudio,
			Length:  uint32(len(buf)),
			Payload: buf,
		}
		if err := writeFrame(state, frame); err != nil {
			log.Println("write tts audio:", err)
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	_ = writeJSON(state, protocol.TTSDoneEvent{Type: protocol.EventTTSDone})
}
