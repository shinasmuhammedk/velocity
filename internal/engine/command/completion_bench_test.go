package command

import (
	"testing"
)

func BenchmarkResultChannel(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ch := make(chan error, 1)

		ch <- nil

		_ = <-ch
	}
}