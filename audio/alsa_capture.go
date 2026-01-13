// // audio/alsa_capture.go
package audio

//
// import (
// 	"fmt"
//
// 	alsa "github.com/Narsil/alsa-go"
// )
//
// type ALSACapture struct {
// 	h *alsa.Handle
// }
//
// func NewALSACapture(device string, sampleRate, channels int) (*ALSACapture, error) {
// 	h := alsa.New()
//
// 	// Open capture stream on ALSA "device" (e.g. "default", "hw:0")
// 	if err := h.Open(device, alsa.StreamTypeCapture, alsa.ModeBlock); err != nil {
// 		return nil, fmt.Errorf("alsa open: %w", err)
// 	}
//
// 	// Configure PCM parameters
// 	h.SampleFormat = alsa.SampleFormatS16LE
// 	h.SampleRate = sampleRate
// 	h.Channels = channels
//
// 	if err := h.ApplyHwParams(); err != nil {
// 		h.Close()
// 		return nil, fmt.Errorf("alsa apply hw params: %w", err)
// 	}
//
// 	return &ALSACapture{h: h}, nil
// }
//
// func (c *ALSACapture) FrameSize() int {
// 	return c.h.FrameSize()
// }
//
// func (c *ALSACapture) Read(buf []byte) (int, error) {
// 	return c.h.Read(buf)
// }
//
// func (c *ALSACapture) Close() error {
// 	if c.h != nil {
// 		c.h.Close()
// 	}
// 	return nil
// }
