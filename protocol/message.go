package protocol

import "encoding/json"

type Event struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data,omitempty"`
}

func EncodeEvent(ev Event) ([]byte, error) {
	return json.Marshal(ev)
}

func DecodeEvent(b []byte) (Event, error) {
	var ev Event
	err := json.Unmarshal(b, &ev)
	return ev, err
}
