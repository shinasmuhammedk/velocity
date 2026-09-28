package command

import (
	"testing"

	"velocity/internal/domain/order"
)

func BenchmarkCommandEnvelopeChannel(b *testing.B) {
	b.ReportAllocs()

	ch := make(chan Command, 1_000_000)
	result := make(chan error, 1)
	o := &order.Order{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var cmd Command

		switch i % 3 {
		case 0:
			cmd = Command{
				Kind:   Submit,
				Order:  o,
				Result: result,
			}
		case 1:
			cmd = Command{
				Kind:    Cancel,
				OrderID: int64(i),
				Result:  result,
			}
		case 2:
			cmd = Command{
				Kind:        Modify,
				OrderID:     int64(i),
				NewPrice:    1000,
				NewQuantity: 10,
				Result:      result,
			}
		}

		ch <- cmd
		_ = <-ch
	}
}