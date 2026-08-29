package candles

import "time"

type Service struct {
	manager *Manager
}

func NewService(manager *Manager) *Service {
	return &Service{
		manager: manager,
	}
}

func (s *Service) Get(
	symbol string,
	interval Interval,
) ([]*Candle, bool) {

	return s.manager.Get(
		symbol,
		interval,
	)
}

func (s *Service) Query(
	symbol string,
	interval Interval,
	limit int,
	startTime *time.Time,
	endTime *time.Time,
) ([]*Candle, bool) {

	return s.manager.Query(
		symbol,
		interval,
		limit,
		startTime,
		endTime,
	)
}

func (s *Service) Latest(
	symbol string,
	interval Interval,
) (*Candle, bool) {

	return s.manager.Latest(
		symbol,
		interval,
	)
}
