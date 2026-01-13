package audio

import (
	"io"
	"os/exec"
	"strconv"
)

type PipewireCapture struct {
	cmd       *exec.Cmd
	stdout    io.ReadCloser
	frameSize int
}

func NewPipewireCapture(sampleRate, channels int) (*PipewireCapture, error) {
	if channels <= 0 {
		channels = 1
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	args := []string{
		"--raw",
		"--channels", strconv.Itoa(channels),
		"--rate", strconv.Itoa(sampleRate),
		"--format", "s16le",
	}
	cmd := exec.Command("parec", args...)

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &PipewireCapture{
		cmd:       cmd,
		stdout:    out,
		frameSize: 2 * channels,
	}, nil
}

func (p *PipewireCapture) Read(buf []byte) (int, error) {
	return p.stdout.Read(buf)
}

func (p *PipewireCapture) FrameSize() int {
	return p.frameSize
}

func (p *PipewireCapture) Close() error {
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	return nil
}
