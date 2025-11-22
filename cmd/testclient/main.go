package main

import (
	"os"
	"vystream/protocol"
)

func main() {
	ev := protocol.Event{
		Type: "describe",
		Data: map[string]interface{}{},
	}

	b, _ := protocol.EncodeEvent(ev)

	f := &protocol.Frame{
		Version: protocol.VersionByte,
		Type:    protocol.FrameTypeJSON,
		Length:  uint32(len(b)),
		Payload: b,
	}

	_ = protocol.WriteFrame(os.Stdout, f)
}
