package main

import (
	"bufio"
	"io"
	"log"
	"os"

	"vystream/protocol"
)

var (
	sampleRate = 16000
	channels   = 1
	format     = "s16le"
)

func main() {
	in := bufio.NewReaderSize(os.Stdin, 4096)
	out := bufio.NewWriterSize(os.Stdout, 4096)

	for {
		f, err := protocol.ReadFrame(in)
		if err != nil {
			if err == io.EOF {
				return
			}
			if err == io.ErrUnexpectedEOF {
				return
			}
			log.Fatal(err)
		}

		if f.Type == protocol.FrameTypeJSON {
			handleJSON(f, out)
		}

		if f.Type == protocol.FrameTypeAudio {
			handleAudio(f.Payload)
		}
	}
}

func handleJSON(f *protocol.Frame, out *bufio.Writer) {
	var base protocol.BaseEvent
	if err := protocol.Decode(f.Payload, &base); err != nil {
		return
	}

	switch base.Type {
	case protocol.EventDescribe:
		sendReady(out)

	case protocol.EventStart:
		// Start stream

	case protocol.EventStop:
		// Stop stream

	default:
		// Ignore unknown
	}
}

func sendReady(out *bufio.Writer) {
	ev := protocol.ReadyEvent{
		Type:       protocol.EventReady,
		Protocol:   "vystream",
		SampleRate: sampleRate,
		Channels:   channels,
		Format:     format,
	}

	b, _ := protocol.Encode(ev)

	frame := &protocol.Frame{
		Version: protocol.VersionByte,
		Type:    protocol.FrameTypeJSON,
		Length:  uint32(len(b)),
		Payload: b,
	}

	_ = protocol.WriteFrame(out, frame)
	_ = out.Flush()
}

func handleAudio(pcm []byte) {
}
