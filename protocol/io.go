package protocol

import (
	"bufio"
	"io"
)

type Writer struct {
	w *bufio.Writer
}

func NewWriter(out io.Writer) *Writer {
	return &Writer{w: bufio.NewWriterSize(out, 4096)}
}

func (wr *Writer) WriteFrame(f *Frame) error {
	if err := WriteFrame(wr.w, f); err != nil {
		return err
	}
	return wr.w.Flush()
}
