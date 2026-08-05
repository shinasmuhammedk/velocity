package orderservice

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"velocity/internal/domain/order"
	"velocity/internal/engine/registry"
	"velocity/internal/infrastructure/metrics"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/mapper"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/service/riskservice"
	"velocity/internal/service/walletservice"
	"velocity/internal/userstream"
	"velocity/pkg/constants"
	"velocity/pkg/errors"
	"velocity/pkg/idgen"
	"velocity/pkg/timeutil"
)

type Service struct {
	orderRepo  repository.OrderRepository
	symbolRepo repository.SymbolRepository
	userRepo   repository.UserRepository

	risk   *riskservice.Service
	wallet *walletservice.Service

	registry *registry.Registry
	logger   *zap.Logger

	UserDispatcher *userstream.Dispatcher
}

func New(
	orderRepo repository.OrderRepository,
	symbolRepo repository.SymbolRepository,
	userRepo repository.UserRepository,

	risk *riskservice.Service,
	wallet *walletservice.Service,

	registry *registry.Registry,
	logger *zap.Logger,

	userDispatcher *userstream.Dispatcher,
) *Service {
	return &Service{
		orderRepo:      orderRepo,
		symbolRepo:     symbolRepo,
		userRepo:       userRepo,
		risk:           risk,
		wallet:         wallet,
		registry:       registry,
		logger:         logger,
		UserDispatcher: userDispatcher,
	}
}

type SubmitOrderRequest struct {
	UserID int64

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

	userID := req.UserID
	var err error
	if err != nil {
		return nil, err
	}

	fmt.Println("STEP 1 - user")
	fmt.Println("UserID:", userID)
	user, errr := s.userRepo.GetByID(
		ctx,
		userID,
	)
	fmt.Println(user)
    fmt.Println(errr)
	// fmt.Println(err)
	if err != nil {
		return nil, errors.ErrUserNotFound
	}

	fmt.Println("STEP 2 - symbol")
	symbol, err := s.symbolRepo.Get(
		ctx,
		req.Symbol,
	)
	fmt.Println(err)
	if err != nil {
		return nil, errors.ErrSymbolNotFound
	}

	if !symbol.IsActive {
		return nil, errors.ErrSymbolInactive
	}

	o := &order.Order{
		ID:     idgen.Next(),
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

	fmt.Println("STEP 3 - risk")
	_, err = s.risk.Validate(
		ctx,
		riskservice.ValidateOrderRequest{
			Order: o,
		},
	)
	fmt.Println(err)

	if err != nil {
		s.UserDispatcher.DispatchOrderRejected(o)
		return nil, err
	}

	userID = o.UserID

	switch o.Side {

	case constants.OrderSideBuy:

		amount := o.Price * o.Quantity

		err = s.wallet.LockFunds(
			ctx,
			userID,
			symbol.QuoteAsset,
			amount,
		)

	case constants.OrderSideSell:

		fmt.Println("STEP 4 - lock")
		err = s.wallet.LockFunds(
			ctx,
			userID,
			symbol.BaseAsset,
			o.Quantity,
		)
		fmt.Println(err)
	}

	if err != nil {
		s.UserDispatcher.DispatchOrderRejected(o)
		return nil, err
	}

	fmt.Println("STEP 5 - create")
	_, err = s.orderRepo.Create(
		ctx,
		generated.CreateOrderParams{
			ID:          o.ID,
			UserID:      o.UserID,
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
	fmt.Println(err)

	if err != nil {
		return nil, err
	}

	fmt.Println("STEP 6 - registry")
	eng := s.registry.Get(req.Symbol)
	fmt.Println(eng)

	fmt.Println("STEP 7 - submit")
	err = eng.SubmitOrder(o)
	fmt.Println(err)
	if err != nil {
		return nil, err
	}
	metrics.OrdersSubmitted.Inc()
	s.UserDispatcher.DispatchOrderAccepted(o)

	fmt.Println("STEP 8 - done")

	return o, nil
}

func (s *Service) Cancel(
	ctx context.Context,
	orderID int64,
) error {

	dbOrder, err := s.orderRepo.GetByID(
		ctx,
		orderID,
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
		ID:        dbOrder.ID,
		UserID:    dbOrder.UserID,
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
	orderID int64,
	req ModifyOrderRequest,
) error {

	dbOrder, err := s.orderRepo.GetByID(
		ctx,
		orderID,
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
		ID:        dbOrder.ID,
		UserID:    dbOrder.UserID,
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

func (s *Service) GetOpenOrders(
	ctx context.Context,
	userID int64,
) ([]*order.Order, error) {

	_, err := s.userRepo.GetByID(
		ctx,
		userID,
	)
	if err != nil {
		return nil, errors.ErrUserNotFound
	}

	rows, err := s.orderRepo.ListOpenOrdersByUser(
		ctx,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return mapper.ToDomainOrders(rows), nil
}

func (s *Service) ListOrderHistory(
	ctx context.Context,
	userID int64,
) ([]*order.Order, error) {

	_, err := s.userRepo.GetByID(
		ctx,
		userID,
	)
	if err != nil {
		return nil, errors.ErrUserNotFound
	}

	rows, err := s.orderRepo.ListOrdersByUser(
		ctx,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return mapper.ToDomainOrders(rows), nil
}

func (s *Service) GetOrderByID(
	ctx context.Context,
	orderID int64,
) (*order.Order, error) {

	dbOrder, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, errors.ErrOrderNotFound
	}

	return mapper.ToDomainOrder(dbOrder), nil
}
