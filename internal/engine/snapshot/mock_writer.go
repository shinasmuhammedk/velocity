package snapshot

type MockWriter struct{}

func (m *MockWriter) Write(
    snapshot *Snapshot,
) error {
    return nil
}