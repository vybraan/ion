package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"ion/audio"
	"ion/protocol"
)

const (
	defaultSampleRate = 16000
	defaultChannels   = 1
	defaultFormat     = "s16le"
	defaultAddr       = ":10300"
)

type connState struct {
	out      *bufio.Writer
	outMu    sync.Mutex
	capture  *audio.PipewireCapture
	streamMu sync.Mutex
	streamOn bool
}

func main() {
	mode := flag.String("mode", "server", "server or client")
	transport := flag.String("transport", "tcp", "tcp or stdio")
	addr := flag.String("addr", defaultAddr, "tcp listen/connect address")
	flag.Parse()

	switch *mode {
	case "server":
		runServer(*transport, *addr)
	case "client":
		runClient(*transport, *addr)
	default:
		log.Fatalf("unknown mode: %s", *mode)
	}
}

func runServer(transport, addr string) {
	switch transport {
	case "stdio":
		handleConn(os.Stdin, os.Stdout)
	case "tcp":
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("listening on", addr)
		for {
			c, err := ln.Accept()
			if err != nil {
				continue
			}
			go handleTCPConn(c)
		}
	default:
		log.Fatalf("unknown transport: %s", transport)
	}
}

func handleTCPConn(c net.Conn) {
	defer c.Close()
	handleConn(c, c)
}

func handleConn(in io.Reader, out io.Writer) {
	reader := bufio.NewReader(in)
	writer := bufio.NewWriter(out)
	state := &connState{out: writer}

	for {
		f, err := protocol.ReadFrame(reader)
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
			// This node is a source; ignore incoming audio for now.
		default:
			// Ignore unknown frame types.
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
		return startStreaming(state)
	case protocol.EventStop:
		return stopStreaming(state)
	default:
		return nil
	}
}

func sendReady(state *connState) error {
	ev := protocol.ReadyEvent{
		Type:       protocol.EventReady,
		Protocol:   "ion",
		SampleRate: defaultSampleRate,
		Channels:   defaultChannels,
		Format:     defaultFormat,
	}
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
	return state.writeFrame(frame)
}

func (s *connState) writeFrame(f *protocol.Frame) error {
	s.outMu.Lock()
	defer s.outMu.Unlock()

	if err := protocol.WriteFrame(s.out, f); err != nil {
		return err
	}
	return s.out.Flush()
}

func startStreaming(s *connState) error {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()

	if s.streamOn {
		return nil
	}

	cap, err := audio.NewPipewireCapture(defaultSampleRate, defaultChannels)
	if err != nil {
		return err
	}
	s.capture = cap
	s.streamOn = true

	go streamLoop(s)
	return nil
}

func stopStreaming(s *connState) error {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()

	if !s.streamOn {
		return nil
	}
	s.streamOn = false
	if s.capture != nil {
		_ = s.capture.Close()
		s.capture = nil
	}
	return nil
}

func streamLoop(s *connState) {
	framesPerChunk := defaultSampleRate / 50

	s.streamMu.Lock()
	cap := s.capture
	s.streamMu.Unlock()
	if cap == nil {
		return
	}

	frameSize := cap.FrameSize()
	if frameSize <= 0 {
		frameSize = 2 * defaultChannels
	}
	buf := make([]byte, framesPerChunk*frameSize)

	for {
		s.streamMu.Lock()
		on := s.streamOn
		cap := s.capture
		s.streamMu.Unlock()

		if !on || cap == nil {
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
			if werr := s.writeFrame(frame); werr != nil {
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

func runClient(transport, addr string) {
	describe := protocol.BaseEvent{Type: protocol.EventDescribe}
	data, err := protocol.Encode(describe)
	if err != nil {
		log.Fatal(err)
	}
	frame := &protocol.Frame{
		Version: protocol.VersionByte,
		Type:    protocol.FrameTypeJSON,
		Length:  uint32(len(data)),
		Payload: data,
	}

	switch transport {
	case "stdio":
		if err := protocol.WriteFrame(os.Stdout, frame); err != nil {
			log.Fatal(err)
		}
	case "tcp":
		c, err := net.Dial("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}
		defer c.Close()

		if err := protocol.WriteFrame(c, frame); err != nil {
			log.Fatal(err)
		}

		reply, err := protocol.ReadFrame(bufio.NewReader(c))
		if err == nil && reply.Type == protocol.FrameTypeJSON {
			var base protocol.BaseEvent
			if derr := protocol.Decode(reply.Payload, &base); derr == nil {
				fmt.Fprintln(os.Stderr, "reply:", base.Type)
			}
		}
	default:
		log.Fatalf("unknown transport: %s", transport)
	}
}
