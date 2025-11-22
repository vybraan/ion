package protocol

import (
	"encoding/binary"
	"io"
)

const (
	VersionByte = 0x01

	FrameTypeJSON  = 0x01
	FrameTypeAudio = 0x02
)

type Frame struct {
	Version byte
	Type    byte
	Length  uint32
	Payload []byte
}

func ReadFrame(r io.Reader) (*Frame, error) {
	header := make([]byte, 6)

	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	f := &Frame{
		Version: header[0],
		Type:    header[1],
		Length:  binary.LittleEndian.Uint32(header[2:6]),
	}

	f.Payload = make([]byte, f.Length)
	if _, err := io.ReadFull(r, f.Payload); err != nil {
		return nil, err
	}

	return f, nil
}

func WriteFrame(w io.Writer, f *Frame) error {
	header := make([]byte, 6)
	header[0] = f.Version
	header[1] = f.Type
	binary.LittleEndian.PutUint32(header[2:6], f.Length)

	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(f.Payload)
	return err
}
