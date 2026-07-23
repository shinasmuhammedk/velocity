package orderservice

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"velocity/internal/domain/order"
	"velocity/internal/engine/registry"
	"velocity/internal/infrastructure/metrics"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/userstream"
	"velocity/pkg/constants"
	"velocity/pkg/errors"
	"velocity/pkg/timeutil"
)

type Service struct {
	orderRepo  repository.OrderRepository
	symbolRepo repository.SymbolRepository
	userRepo   repository.UserRepository

	registry *registry.Registry
	logger   *zap.Logger

	UserDispatcher *userstream.Dispatcher
}

func New(
	orderRepo repository.OrderRepository,
	symbolRepo repository.SymbolRepository,
	userRepo repository.UserRepository,

	registry *registry.Registry,
	logger *zap.Logger,

	userDispatcher *userstream.Dispatcher,
) *Service {
	return &Service{
		orderRepo:      orderRepo,
		symbolRepo:     symbolRepo,
		userRepo:       userRepo,
		registry:       registry,
		logger:         logger,
		UserDispatcher: userDispatcher,
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

type ModifyOrderRequest struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"quantity"`
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
		return nil, errors.ErrUserNotFound
	}

	symbol, err := s.symbolRepo.Get(
		ctx,
		req.Symbol,
	)
	if err != nil {
		return nil, errors.ErrSymbolNotFound
	}

	if !symbol.IsActive {
		return nil, errors.ErrSymbolInactive
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

		CreatedAt: timeutil.UTCNow(),
		UpdatedAt: timeutil.UTCNow(),
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

	eng := s.registry.Get(req.Symbol)

	err = eng.SubmitOrder(o)
	if err != nil {
		return nil, err
	}
	metrics.OrdersSubmitted.Inc()
	s.UserDispatcher.DispatchOrderAccepted(o)

	return o, nil
}

func (s *Service) Cancel(
	ctx context.Context,
	// userID string,
	orderID string,
) error {

	dbOrder, err := s.orderRepo.GetByID(
		ctx,
		uuid.MustParse(orderID),
	)

	if err != nil {
		return err
	}

	// eng := s.registry.Get(dbOrder.Symbol)

	eng, ok := s.registry.Find(dbOrder.Symbol)
	if !ok {
		return errors.ErrEngineUnavailable
	}

	err = eng.CancelOrder(orderID)
	if err != nil {
		return err

	}

	switch dbOrder.Status {
	case string(constants.OrderStatusFilled),
		string(constants.OrderStatusCancelled),
		string(constants.OrderStatusRejected):

		return errors.ErrOrderNotCancelable
	}

	metrics.OrdersCancelled.Inc()

	err = s.orderRepo.UpdateStatus(
		ctx,
		generated.UpdateOrderStatusParams{
			ID:     dbOrder.ID,
			Status: string(constants.OrderStatusCancelled),
		},
	)

	if err != nil {
		return err
	}

	o := &order.Order{
		ID:        dbOrder.ID.String(),
		UserID:    dbOrder.UserID.String(),
		Symbol:    dbOrder.Symbol,
		Status:    constants.OrderStatusCancelled,
		Price:     dbOrder.Price.Int64,
		Quantity:  dbOrder.Quantity,
		Filled:    dbOrder.Filled,
		Remaining: dbOrder.Remaining,
	}

	s.UserDispatcher.DispatchOrderCancelled(o)

	return nil
}

func (s *Service) Modify(
	ctx context.Context,
	orderID string,
	req ModifyOrderRequest,
) error {

	dbOrder, err := s.orderRepo.GetByID(
		ctx,
		uuid.MustParse(orderID),
	)
	if err != nil {
		return errors.ErrOrderNotFound
	}

	if dbOrder.Status != string(constants.OrderStatusOpen) {
		return errors.ErrOrderModificationNotAllowed
	}

	if req.Quantity < dbOrder.Filled {
		return errors.ErrQuantityTooLow
	}

	eng := s.registry.Get(dbOrder.Symbol)

	err = eng.ModifyOrder(
		orderID,
		req.Price,
		req.Quantity,
	)

	if err != nil {
		return err
	}

	remaining := req.Quantity - dbOrder.Filled

	metrics.OrdersModified.Inc()

	err = s.orderRepo.UpdateOrderForModify(
		ctx,
		generated.UpdateOrderForModifyParams{
			ID: dbOrder.ID,
			Price: pgtype.Int8{
				Int64: req.Price,
				Valid: true,
			},
			Quantity:  req.Quantity,
			Remaining: remaining,
		},
	)

	if err != nil {
		return err
	}

	o := &order.Order{
		ID:        dbOrder.ID.String(),
		UserID:    dbOrder.UserID.String(),
		Symbol:    dbOrder.Symbol,
		Status:    constants.OrderStatusOpen,
		Price:     req.Price,
		Quantity:  req.Quantity,
		Filled:    dbOrder.Filled,
		Remaining: remaining,
	}

	s.UserDispatcher.DispatchOrderModified(o)

	return nil
}
