package candles

import "time"

type Interval string

const (
	Interval1m  Interval = "1m"
	Interval5m  Interval = "5m"
	Interval15m Interval = "15m"
	Interval1h  Interval = "1h"
	Interval4h  Interval = "4h"
	Interval1d  Interval = "1d"
)

func (i Interval) Duration() time.Duration {
	switch i {
	case Interval1m:
		return time.Minute

	case Interval5m:
		return 5 * time.Minute

	case Interval15m:
		return 15 * time.Minute

	case Interval1h:
		return time.Hour

	case Interval4h:
		return 4 * time.Hour

	case Interval1d:
		return 24 * time.Hour
	}

	return 0
}

func (i Interval) IsValid() bool {
	switch i {
	case Interval1m,
		Interval5m,
		Interval15m,
		Interval1h,
		Interval4h,
		Interval1d:
		return true
	default:
		return false
	}
}
