package snapshot

type Serializer interface {
	Serialize(*Snapshot) ([]byte, error)
	Deserialize([]byte) (*Snapshot, error)
}