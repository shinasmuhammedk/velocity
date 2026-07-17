package wal

import "encoding/json"

type JSONSerializer struct{}

func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

func (s *JSONSerializer) Serialize(
	event *Event,
) ([]byte, error) {
	return json.Marshal(event)
}

func (s *JSONSerializer) Deserialize(
	data []byte,
) (*Event, error) {
	var event Event

	err := json.Unmarshal(data, &event)
	if err != nil {
		return nil, err
	}

	return &event, nil
}