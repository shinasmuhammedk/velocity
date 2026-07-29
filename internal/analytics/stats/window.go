package stats

import "time"

const WindowDuration = 24 * time.Hour

func (s *MarketStats) WindowExpired(now time.Time) bool {
	return now.Sub(s.WindowStart) >= WindowDuration
}