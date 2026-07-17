package wal

type Serializer interface {
	Serialize(*Event) ([]byte, error)
	Deserialize([]byte) (*Event, error)
}