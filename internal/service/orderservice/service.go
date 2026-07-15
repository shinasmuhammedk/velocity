package orderservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"velocity/internal/domain/order"
	"velocity/internal/engine/registry"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/pkg/constants"
)

type Service struct {
	orderRepo  repository.OrderRepository
	symbolRepo repository.SymbolRepository
	userRepo   repository.UserRepository

	registry *registry.Registry
	logger   *zap.Logger
}

func New(
	orderRepo repository.OrderRepository,
	symbolRepo repository.SymbolRepository,
	userRepo repository.UserRepository,

	registry *registry.Registry,
	logger *zap.Logger,
) *Service {
	return &Service{
		orderRepo:  orderRepo,
		symbolRepo: symbolRepo,
		userRepo:   userRepo,
		registry:   registry,
		logger:     logger,
	}
}

type SubmitOrderRequest struct {
	UserID string

	Symbol string

	Side        constants.OrderSide
	Type        constants.OrderType
	TimeInForce constants.TimeInForce

	Price     int64
	StopPrice int64
	Quantity  int64
}

func (s *Service) Submit(
	ctx context.Context,
	req SubmitOrderRequest,
) (*order.Order, error) {

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, err
	}

	_, err = s.userRepo.GetByID(
		ctx,
		userID,
	)
	if err != nil {
		return nil, errors.New("user not found")
	}

	symbol, err := s.symbolRepo.Get(
		ctx,
		req.Symbol,
	)
	if err != nil {
		return nil, errors.New("symbol not found")
	}

	if !symbol.IsActive {
		return nil, errors.New("symbol inactive")
	}

	o := &order.Order{
		ID:     uuid.NewString(),
		UserID: req.UserID,
		Symbol: req.Symbol,

		Side:        req.Side,
		Type:        req.Type,
		TimeInForce: req.TimeInForce,

		Status: constants.OrderStatusOpen,

		Price:     req.Price,
		StopPrice: req.StopPrice,

		Quantity:  req.Quantity,
		Remaining: req.Quantity,
		Filled:    0,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.logger.Info(
		"creating order",
		zap.String("tif", string(o.TimeInForce)),
		zap.String("type", string(o.Type)),
		zap.String("side", string(o.Side)),
	)

	_, err = s.orderRepo.Create(
		ctx,
		generated.CreateOrderParams{
			ID:          uuid.MustParse(o.ID),
			UserID:      uuid.MustParse(o.UserID),
			Symbol:      o.Symbol,
			Side:        string(o.Side),
			OrderType:   string(o.Type),
			TimeInForce: string(o.TimeInForce),
			Status:      string(o.Status),

			Price: pgtype.Int8{
				Int64: o.Price,
				Valid: true,
			},

			StopPrice: o.StopPrice,

			Quantity:  o.Quantity,
			Remaining: o.Remaining,
			Filled:    o.Filled,

			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
		},
	)

	if err != nil {
		return nil, err
	}

	fmt.Printf("ORDER BEFORE ENGINE %+v\n", o)

	eng := s.registry.Get(req.Symbol)

	err = eng.SubmitOrder(o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (s *Service) Cancel(
	ctx context.Context,
	orderID string,
) error {

	dbOrder, err := s.orderRepo.GetByID(
		ctx,
		uuid.MustParse(orderID),
	)

	if err != nil {
		return err
	}

	eng := s.registry.Get(dbOrder.Symbol)

	err = eng.CancelOrder(orderID)
	if err != nil {
		return err
	}

	return s.orderRepo.UpdateStatus(
		ctx,
		generated.UpdateOrderStatusParams{
			ID:     dbOrder.ID,
			Status: string(constants.OrderStatusCancelled),
		},
	)
}
