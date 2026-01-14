package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"ion/protocol"
)

type audioSink struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	mu    sync.Mutex
}

func main() {
	transport := flag.String("transport", "tcp", "tcp or stdio")
	addr := flag.String("addr", ":10300", "tcp address")
	name := flag.String("name", "ion-satellite", "satellite name")
	micCmd := flag.String("mic-command", "", "command that outputs raw PCM on stdout")
	sndCmd := flag.String("snd-command", "", "command that accepts raw PCM on stdin")
	autoASR := flag.Bool("auto-asr", true, "send asr.start and stream mic immediately")
	flag.Parse()

	var (
		reader *bufio.Reader
		writer *bufio.Writer
		closer io.Closer
	)

	switch *transport {
	case "tcp":
		c, err := net.Dial("tcp", *addr)
		if err != nil {
			log.Fatal(err)
		}
		reader = bufio.NewReader(c)
		writer = bufio.NewWriter(c)
		closer = c
	case "stdio":
		reader = bufio.NewReader(os.Stdin)
		writer = bufio.NewWriter(os.Stdout)
	default:
		log.Fatalf("unknown transport: %s", *transport)
	}
	if closer != nil {
		defer closer.Close()
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

	if err := sendEvent(writer, protocol.BaseEvent{Type: protocol.EventDescribe}); err != nil {
		log.Fatal(err)
	}

	readyFrame, err := protocol.ReadFrame(reader)
	if err != nil {
		log.Fatal(err)
	}
	if readyFrame.Type != protocol.FrameTypeJSON {
		log.Fatalf("expected ready JSON frame, got type: %d", readyFrame.Type)
	}
	var ready protocol.ReadyEvent
	if err := protocol.Decode(readyFrame.Payload, &ready); err != nil {
		log.Fatal(err)
	}
	if ready.Type != protocol.EventReady {
		log.Fatalf("expected ready event, got: %s", ready.Type)
	}

	if err := sendEvent(writer, protocol.SatelliteHelloEvent{
		Type:       protocol.EventSatelliteHello,
		Name:       *name,
		SampleRate: ready.SampleRate,
		Channels:   ready.Channels,
		Format:     ready.Format,
		Wake:       false,
		VAD:        false,
		ASR:        true,
		TTS:        true,
	}); err != nil {
		log.Fatal(err)
	}

	sink, err := startAudioSink(*sndCmd)
	if err != nil {
		log.Fatal(err)
	}
	defer sink.Close()

	if *autoASR && *micCmd != "" {
		if err := sendEvent(writer, protocol.ASRStartEvent{Type: protocol.EventASRStart}); err != nil {
			log.Fatal(err)
		}
	}

	micDone := make(chan struct{})
	if *micCmd != "" {
		go func() {
			if err := streamMic(*micCmd, writer); err != nil {
				log.Println("mic stream error:", err)
			}
			close(micDone)
		}()
	}

	go func() {
		<-interrupt
		if *autoASR && *micCmd != "" {
			_ = sendEvent(writer, protocol.ASRStopEvent{Type: protocol.EventASRStop})
		}
		if sink != nil {
			_ = sink.Close()
		}
		os.Exit(0)
	}()

	for {
		f, err := protocol.ReadFrame(reader)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			log.Fatal(err)
		}
		switch f.Type {
		case protocol.FrameTypeJSON:
			log.Printf("event: %s", string(f.Payload))
		case protocol.FrameTypeAudio:
			if sink != nil {
				_ = sink.Write(f.Payload)
			}
		default:
			// ignore
		}
	}
}

func sendEvent(w *bufio.Writer, ev any) error {
	payload, err := protocol.Encode(ev)
	if err != nil {
		return err
	}
	frame := &protocol.Frame{
		Version: protocol.VersionByte,
		Type:    protocol.FrameTypeJSON,
		Length:  uint32(len(payload)),
		Payload: payload,
	}
	if err := protocol.WriteFrame(w, frame); err != nil {
		return err
	}
	return w.Flush()
}

func streamMic(cmdLine string, w *bufio.Writer) error {
	cmd := exec.Command("sh", "-c", cmdLine)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		_ = cmd.Process.Kill()
	}()

	buf := make([]byte, 320*2)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			frame := &protocol.Frame{
				Version: protocol.VersionByte,
				Type:    protocol.FrameTypeAudio,
				Length:  uint32(n),
				Payload: buf[:n],
			}
			if err := protocol.WriteFrame(w, frame); err != nil {
				return err
			}
			if err := w.Flush(); err != nil {
				return err
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func startAudioSink(cmdLine string) (*audioSink, error) {
	if strings.TrimSpace(cmdLine) == "" {
		return nil, nil
	}
	cmd := exec.Command("sh", "-c", cmdLine)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &audioSink{cmd: cmd, stdin: stdin}, nil
}

func (s *audioSink) Write(p []byte) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.stdin.Write(p)
	return err
}

func (s *audioSink) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	return nil
}
