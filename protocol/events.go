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

const (
	EventSatelliteHello EventType = "satellite.hello"
	EventSatelliteState EventType = "satellite.state"
)

const (
	EventWakeDetected EventType = "wake.detected"
	EventWakeReset    EventType = "wake.reset"
	EventVADStart     EventType = "vad.start"
	EventVADStop      EventType = "vad.stop"
)

const (
	EventASRStart   EventType = "asr.start"
	EventASRStop    EventType = "asr.stop"
	EventASRPartial EventType = "asr.partial"
	EventASRResult  EventType = "asr.result"
	EventASRError   EventType = "asr.error"
)

const (
	EventTTSStart EventType = "tts.start"
	EventTTSReady EventType = "tts.ready"
	EventTTSDone  EventType = "tts.done"
	EventTTSStop  EventType = "tts.stop"
	EventTTSError EventType = "tts.error"
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

type ErrorEvent struct {
	Type    EventType `json:"type"`
	Message string    `json:"message"`
}

type SatelliteHelloEvent struct {
	Type       EventType `json:"type"`
	Name       string    `json:"name"`
	SampleRate int       `json:"sample_rate"`
	Channels   int       `json:"channels"`
	Format     string    `json:"format"`
	Wake       bool      `json:"wake,omitempty"`
	VAD        bool      `json:"vad,omitempty"`
	ASR        bool      `json:"asr,omitempty"`
	TTS        bool      `json:"tts,omitempty"`
}

type SatelliteStateEvent struct {
	Type  EventType `json:"type"`
	State string    `json:"state"`
}

type WakeDetectedEvent struct {
	Type EventType `json:"type"`
	Name string    `json:"name,omitempty"`
}

type WakeResetEvent struct {
	Type EventType `json:"type"`
}

type VADStartEvent struct {
	Type EventType `json:"type"`
}

type VADStopEvent struct {
	Type EventType `json:"type"`
}

type ASRStartEvent struct {
	Type     EventType `json:"type"`
	Language string    `json:"language,omitempty"`
}

type ASRStopEvent struct {
	Type EventType `json:"type"`
}

type ASRPartialEvent struct {
	Type EventType `json:"type"`
	Text string    `json:"text"`
}

type ASRResultEvent struct {
	Type EventType `json:"type"`
	Text string    `json:"text"`
}

type ASRErrorEvent struct {
	Type    EventType `json:"type"`
	Message string    `json:"message"`
}

type TTSStartEvent struct {
	Type     EventType `json:"type"`
	Text     string    `json:"text"`
	Voice    string    `json:"voice,omitempty"`
	Language string    `json:"language,omitempty"`
}

type TTSReadyEvent struct {
	Type EventType `json:"type"`
}

type TTSDoneEvent struct {
	Type EventType `json:"type"`
}

type TTSStopEvent struct {
	Type EventType `json:"type"`
}

type TTSErrorEvent struct {
	Type    EventType `json:"type"`
	Message string    `json:"message"`
}

func Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
