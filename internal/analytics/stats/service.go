package stats

type Service struct {
	manager *Manager
}

func NewService(manager *Manager) *Service {
	return &Service{
		manager: manager,
	}
}

func (s *Service) Get(symbol string) (*MarketStats, bool) {
	return s.manager.Get(symbol)
}