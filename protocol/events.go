package protocol

type EventType string

const (
	EventReady    EventType = "ready"
	EventStart    EventType = "start"
	EventStop     EventType = "stop"
	EventError    EventType = "error"
	EventAudio    EventType = "audio"
	EventDescribe EventType = "describe"
)

type ReadyEvent struct {
	Type         EventType `json:"type"`
	Protocol     string    `json:"protocol"`
	AudioFormats []string  `json:"audio_formats,omitempty"`
}
