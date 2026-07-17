package snapshot

import "encoding/json"

type JSONSerailizer struct{}

func NewJSONSerializer() *JSONSerailizer {
	return &JSONSerailizer{}
}

func (s *JSONSerailizer) Serialize(snapshot *Snapshot) ([]byte, error) {
	return json.MarshalIndent(snapshot, "", " ")
}

func (s *JSONSerailizer) Deserialize(data []byte) (*Snapshot, error) {
	var snapshot Snapshot

	err := json.Unmarshal(data, &snapshot)

	if err != nil {
		return nil, err
	}

	return &snapshot, nil
}
