package userservice

import (
	"context"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/service/walletservice"
	"velocity/pkg/logger"
	"velocity/pkg/timeutil"
)

type Service struct {
	userRepo  repository.UserRepository
	walletSvc *walletservice.Service
}

func New(
	userRepo repository.UserRepository,
	walletSvc *walletservice.Service,
) *Service {
	return &Service{
		userRepo:  userRepo,
		walletSvc: walletSvc,
	}
}

func (s *Service) CreateUser(
	ctx context.Context,
	req CreateUserRequest,
) (*generated.User, error) {

	exists, err := s.userRepo.Exists(ctx, req.ID)
	if err != nil {
		logger.Error(
			"failed checking user existence",
			logger.Int64("user_id", req.ID),
			logger.ErrorField(err),
		)
		return nil, err
	}

	if exists {
		logger.Info(
			"user already synchronized",
			logger.Int64("user_id", req.ID),
		)

		user, err := s.userRepo.GetByID(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return &user, nil
	}

	user, err := s.userRepo.Create(
		ctx,
		generated.CreateUserParams{
			ID:        req.ID,
			Email:     req.Email,
			CreatedAt: timeutil.UTCNow(),
			UpdatedAt: timeutil.UTCNow(),
		},
	)

	if err := s.walletSvc.CreateDefaultWallets(
		ctx,
		user.ID,
	); err != nil {

		logger.Error(
			"failed creating default wallets",
			logger.ErrorField(err),
		)

		return nil, err
	}

	if err != nil {
		logger.Error(
			"failed creating user",
			logger.Int64("user_id", req.ID),
			logger.ErrorField(err),
		)
		return nil, err
	}

	logger.Info(
		"user synchronized successfully",
		logger.Int64("user_id", user.ID),
		logger.String("email", user.Email),
	)

	return &user, nil
}
