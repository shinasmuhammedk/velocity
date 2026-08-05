package positionservice

import (
    "context"


    "velocity/internal/persistence/postgres/generated"
    "velocity/internal/persistence/postgres/repository"
)

type Service struct {
    repo repository.PositionRepository
}

func New(
    repo repository.PositionRepository,
) *Service {

    return &Service{
        repo: repo,
    }
}

func (s *Service) List(
    ctx context.Context,
    userID int64,
) ([]generated.Position, error) {

    return s.repo.ListByUser(
        ctx,
        userID,
    )
}

func (s *Service) GetPosition(
	ctx context.Context,
	userID int64,
	symbol string,
) (generated.Position, error) {

	return s.repo.Get(
		ctx,
		userID,
		symbol,
	)
}