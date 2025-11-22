package protocol

import (
	"bytes"
	"testing"
)

func TestFrameRoundtrip(t *testing.T) {
	orig := &Frame{
		Version: VersionByte,
		Type:    FrameTypeJSON,
		Payload: []byte(`{"type":"test"}`),
		Length:  15,
	}

	buf := new(bytes.Buffer)
	if err := WriteFrame(buf, orig); err != nil {
		t.Fatal(err)
	}

	out, err := ReadFrame(buf)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(orig.Payload, out.Payload) {
		t.Fatal("payload mismatch")
	}
}
