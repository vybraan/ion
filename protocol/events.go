package protocol

import "encoding/json"

type EventType string

const (
	EventDescribe EventType = "describe"
	EventReady    EventType = "ready"
	EventStart    EventType = "start"
	EventStop     EventType = "stop"
	EventError    EventType = "error"
)

const (
	EventAudio EventType = "audio"
)

type BaseEvent struct {
	Type EventType `json:"type"`
}

type ReadyEvent struct {
	Type       EventType `json:"type"`
	Protocol   string    `json:"protocol"`
	SampleRate int       `json:"sample_rate"`
	Channels   int       `json:"channels"`
	Format     string    `json:"format"`
}

type StartEvent struct {
	Type EventType `json:"type"`
}

type StopEvent struct {
	Type EventType `json:"type"`
}

func Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
