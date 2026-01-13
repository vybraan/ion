package transport

import (
	"bufio"
	"io"
	"net"

	"ion/protocol"
)

func HandleConn(c net.Conn) {
	defer c.Close()

	in := bufio.NewReader(c)
	out := bufio.NewWriter(c)

	for {
		f, err := protocol.ReadFrame(in)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			return
		}

		if f.Type != protocol.FrameTypeJSON {
			continue
		}

		var base protocol.BaseEvent
		if err := protocol.Decode(f.Payload, &base); err != nil {
			continue
		}

		if base.Type == protocol.EventDescribe {
			sendReady(out)
		}
	}
}

func sendReady(out *bufio.Writer) {
	ev := protocol.ReadyEvent{
		Type:       protocol.EventReady,
		Protocol:   "ion",
		SampleRate: 16000,
		Channels:   1,
		Format:     "s16le",
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
