package main

import (
	"log"
	"os"

	"vystream/protocol"
)

func main() {
	for {
		frame, err := protocol.ReadFrame(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}

		switch frame.Type {
		case protocol.FrameTypeJSON:
			ev, err := protocol.DecodeEvent(frame.Payload)
			if err != nil {
				log.Println(err)
				continue
			}

			outPayload, _ := protocol.EncodeEvent(ev)

			out := &protocol.Frame{
				Version: protocol.VersionByte,
				Type:    protocol.FrameTypeJSON,
				Length:  uint32(len(outPayload)),
				Payload: outPayload,
			}

			_ = protocol.WriteFrame(os.Stdout, out)

		case protocol.FrameTypeAudio:
		}
	}
}
