package settlementservice

import (
	"context"
	stderrors "errors"

	"velocity/internal/domain/order"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/userstream"
	"velocity/pkg/constants"
	"velocity/pkg/errors"

	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"
	"velocity/internal/service/walletservice"

	"github.com/jackc/pgx/v5"
)

// isSettleable reports whether an order is still eligible to be settled
// against a trade. Only orders that are open or partially filled can
// accept further fills; orders that are already fully filled, cancelled,
// or rejected must never be settled again.
func isSettleable(status string) bool {
	switch constants.OrderStatus(status) {
	case constants.OrderStatusOpen, constants.OrderStatusPartiallyFilled:
		return true
	default:
		return false
	}
}

type Service struct {
	txManager tx.Manager

	UserDispatcher *userstream.Dispatcher
}

func New(
	txManager tx.Manager,
	userDispatcher *userstream.Dispatcher,
) *Service {

	return &Service{
		txManager:      txManager,
		UserDispatcher: userDispatcher,
	}
}

func (s *Service) Settle(
	ctx context.Context,
	req SettlementRequest,
) error {

	var (
		buyerBaseBalance   userstream.BalanceUpdate
		buyerQuoteBalance  userstream.BalanceUpdate
		sellerBaseBalance  userstream.BalanceUpdate
		sellerQuoteBalance userstream.BalanceUpdate

		buyerPositionEvent userstream.PositionUpdate
		sellerPositionEvent userstream.PositionUpdate

		tradeEvent userstream.TradeExecution

		buyOrderEvent  *order.Order
		sellOrderEvent *order.Order

		buyFilled  bool
		sellFilled bool

		tradeAlreadySettled bool
	)

	err := s.txManager.WithTransaction(
		ctx,
		func(tx pgx.Tx) error {

			orderRepo := repository.NewOrderRepository(tx)
			tradeRepo := repository.NewTradeRepository(tx)
			positionRepo := repository.NewPositionRepository(tx)
			walletRepo := repository.NewWalletRepository(tx)

			walletService := walletservice.New(walletRepo)

			// ---------------------------------------------------------
			// 1. Atomically claim the trade ID.
			//
			// The trade ID is the idempotency key for settlement.
			//
			// If the trade already exists:
			//   ON CONFLICT DO NOTHING
			//   RETURNING *
			//
			// produces pgx.ErrNoRows.
			//
			// We return immediately so duplicate settlement performs
			// absolutely no state mutation and emits no events.
			// ---------------------------------------------------------

			_, err := tradeRepo.CreateIfNotExists(
				ctx,
				generated.CreateTradeIfNotExistsParams{
					ID:         req.TradeID,
					BuyOrderID: req.BuyOrderID,
					SellOrderID: req.SellOrderID,
					BuyerID:    req.BuyerID,
					SellerID:   req.SellerID,
					Symbol:     req.Symbol,
					Price:      req.Price,
					Quantity:   req.Quantity,
				},
			)

			if err != nil {
				if stderrors.Is(err, pgx.ErrNoRows) {
					tradeAlreadySettled = true
					return nil
				}

				return err
			}

			// ---------------------------------------------------------
			// 2. Load the orders.
			// ---------------------------------------------------------

			buyOrder, err := orderRepo.GetByID(
				ctx,
				req.BuyOrderID,
			)
			if err != nil {
				return err
			}

			sellOrder, err := orderRepo.GetByID(
				ctx,
				req.SellOrderID,
			)
			if err != nil {
				return err
			}

			// ---------------------------------------------------------
			// 3. Validate order state.
			//
			// Only OPEN and PARTIALLY_FILLED orders can receive
			// another trade.
			// ---------------------------------------------------------

			if !isSettleable(buyOrder.Status) {
				return errors.ErrOrderNotSettleable
			}

			if !isSettleable(sellOrder.Status) {
				return errors.ErrOrderNotSettleable
			}

			// ---------------------------------------------------------
			// 4. Validate the settlement quantity.
			// ---------------------------------------------------------

			if req.Quantity <= 0 {
				return errors.ErrInvalidQuantity
			}

			if req.Quantity > buyOrder.Remaining {
				return errors.ErrInvalidQuantity
			}

			if req.Quantity > sellOrder.Remaining {
				return errors.ErrInvalidQuantity
			}

			// ---------------------------------------------------------
			// 5. Create trade event.
			// ---------------------------------------------------------

			tradeEvent = userstream.TradeExecution{
				TradeID: req.TradeID,

				BuyOrderID:  req.BuyOrderID,
				SellOrderID: req.SellOrderID,

				BuyerID:  req.BuyerID,
				SellerID: req.SellerID,

				Symbol: req.Symbol,

				Price:    req.Price,
				Quantity: req.Quantity,
			}

			// ---------------------------------------------------------
			// 6. Consume buyer's locked quote funds.
			// ---------------------------------------------------------

			err = walletService.ConsumeLockedFunds(
				ctx,
				req.BuyerID,
				req.QuoteAsset,
				req.Price*req.Quantity,
			)
			if err != nil {
				return err
			}

			// ---------------------------------------------------------
			// 7. Consume seller's locked base funds.
			// ---------------------------------------------------------

			err = walletService.ConsumeLockedFunds(
				ctx,
				req.SellerID,
				req.BaseAsset,
				req.Quantity,
			)
			if err != nil {
				return err
			}

			// ---------------------------------------------------------
			// 8. Deposit purchased base asset to buyer.
			// ---------------------------------------------------------

			err = walletService.Deposit(
				ctx,
				req.BuyerID,
				req.BaseAsset,
				req.Quantity,
			)
			if err != nil {
				return err
			}

			// ---------------------------------------------------------
			// 9. Deposit quote asset to seller.
			// ---------------------------------------------------------

			err = walletService.Deposit(
				ctx,
				req.SellerID,
				req.QuoteAsset,
				req.Price*req.Quantity,
			)
			if err != nil {
				return err
			}

			// ---------------------------------------------------------
			// 10. Update buyer position.
			// ---------------------------------------------------------

			err = positionRepo.Upsert(
				ctx,
				generated.UpsertPositionParams{
					UserID:   req.BuyerID,
					Symbol:   req.Symbol,
					Quantity: req.Quantity,
				},
			)
			if err != nil {
				return err
			}

			// ---------------------------------------------------------
			// 11. Update seller position.
			// ---------------------------------------------------------

			err = positionRepo.Upsert(
				ctx,
				generated.UpsertPositionParams{
					UserID:   req.SellerID,
					Symbol:   req.Symbol,
					Quantity: -req.Quantity,
				},
			)
			if err != nil {
				return err
			}

			// ---------------------------------------------------------
			// 12. Update BUY order.
			// ---------------------------------------------------------

			buyRemaining := buyOrder.Remaining - req.Quantity
			buyFilledQty := buyOrder.Filled + req.Quantity

			buyFilled = buyRemaining == 0

			buyStatus := string(constants.OrderStatusPartiallyFilled)

			if buyRemaining == 0 {
				buyStatus = string(constants.OrderStatusFilled)
			}

			err = orderRepo.UpdateOrderAfterTrade(
				ctx,
				generated.UpdateOrderAfterTradeParams{
					ID:        buyOrder.ID,
					Remaining: buyRemaining,
					Filled:    buyFilledQty,
					Status:    buyStatus,
				},
			)
			if err != nil {
				return err
			}

			buyOrderEvent = &order.Order{
				ID:        buyOrder.ID,
				UserID:    buyOrder.UserID,
				Symbol:    buyOrder.Symbol,
				Status:    constants.OrderStatus(buyStatus),
				Price:     buyOrder.Price.Int64,
				Quantity:  buyOrder.Quantity,
				Filled:    buyFilledQty,
				Remaining: buyRemaining,
			}

			// ---------------------------------------------------------
			// 13. Update SELL order.
			// ---------------------------------------------------------

			sellRemaining := sellOrder.Remaining - req.Quantity
			sellFilledQty := sellOrder.Filled + req.Quantity

			sellFilled = sellRemaining == 0

			sellStatus := string(constants.OrderStatusPartiallyFilled)

			if sellRemaining == 0 {
				sellStatus = string(constants.OrderStatusFilled)
			}

			err = orderRepo.UpdateOrderAfterTrade(
				ctx,
				generated.UpdateOrderAfterTradeParams{
					ID:        sellOrder.ID,
					Remaining: sellRemaining,
					Filled:    sellFilledQty,
					Status:    sellStatus,
				},
			)
			if err != nil {
				return err
			}

			sellOrderEvent = &order.Order{
				ID:        sellOrder.ID,
				UserID:    sellOrder.UserID,
				Symbol:    sellOrder.Symbol,
				Status:    constants.OrderStatus(sellStatus),
				Price:     sellOrder.Price.Int64,
				Quantity:  sellOrder.Quantity,
				Filled:    sellFilledQty,
				Remaining: sellRemaining,
			}

			// ---------------------------------------------------------
			// 14. Read final wallet balances for user-stream events.
			// ---------------------------------------------------------

			buyerBaseWallet, err := walletRepo.Get(
				ctx,
				req.BuyerID,
				req.BaseAsset,
			)
			if err != nil {
				return err
			}

			buyerQuoteWallet, err := walletRepo.Get(
				ctx,
				req.BuyerID,
				req.QuoteAsset,
			)
			if err != nil {
				return err
			}

			sellerBaseWallet, err := walletRepo.Get(
				ctx,
				req.SellerID,
				req.BaseAsset,
			)
			if err != nil {
				return err
			}

			sellerQuoteWallet, err := walletRepo.Get(
				ctx,
				req.SellerID,
				req.QuoteAsset,
			)
			if err != nil {
				return err
			}

			// ---------------------------------------------------------
			// 15. Read final positions for user-stream events.
			// ---------------------------------------------------------

			buyerPosition, err := positionRepo.Get(
				ctx,
				req.BuyerID,
				req.Symbol,
			)
			if err != nil {
				return err
			}

			sellerPosition, err := positionRepo.Get(
				ctx,
				req.SellerID,
				req.Symbol,
			)
			if err != nil {
				return err
			}

			// ---------------------------------------------------------
			// 16. Build balance events.
			// ---------------------------------------------------------

			buyerBaseBalance = userstream.BalanceUpdate{
				Asset:     req.BaseAsset,
				Available: buyerBaseWallet.Available,
				Locked:    buyerBaseWallet.Locked,
			}

			buyerQuoteBalance = userstream.BalanceUpdate{
				Asset:     req.QuoteAsset,
				Available: buyerQuoteWallet.Available,
				Locked:    buyerQuoteWallet.Locked,
			}

			sellerBaseBalance = userstream.BalanceUpdate{
				Asset:     req.BaseAsset,
				Available: sellerBaseWallet.Available,
				Locked:    sellerBaseWallet.Locked,
			}

			sellerQuoteBalance = userstream.BalanceUpdate{
				Asset:     req.QuoteAsset,
				Available: sellerQuoteWallet.Available,
				Locked:    sellerQuoteWallet.Locked,
			}

			// ---------------------------------------------------------
			// 17. Build position events.
			// ---------------------------------------------------------

			buyerPositionEvent = userstream.PositionUpdate{
				Symbol:   req.Symbol,
				Quantity: buyerPosition.Quantity,
			}

			sellerPositionEvent = userstream.PositionUpdate{
				Symbol:   req.Symbol,
				Quantity: sellerPosition.Quantity,
			}

			return nil
		},
	)

	// -------------------------------------------------------------
	// Transaction failed.
	// -------------------------------------------------------------

	if err != nil {
		return err
	}

	// -------------------------------------------------------------
	// Duplicate settlement.
	//
	// No state was changed and no events should be emitted.
	// -------------------------------------------------------------

	if tradeAlreadySettled {
		return nil
	}

	// -------------------------------------------------------------
	// Publish events ONLY after the database transaction commits.
	// -------------------------------------------------------------

	s.UserDispatcher.DispatchBalanceUpdated(
		req.BuyerID,
		buyerBaseBalance,
	)

	s.UserDispatcher.DispatchBalanceUpdated(
		req.BuyerID,
		buyerQuoteBalance,
	)

	s.UserDispatcher.DispatchBalanceUpdated(
		req.SellerID,
		sellerBaseBalance,
	)

	s.UserDispatcher.DispatchBalanceUpdated(
		req.SellerID,
		sellerQuoteBalance,
	)

	s.UserDispatcher.DispatchPositionUpdated(
		req.BuyerID,
		buyerPositionEvent,
	)

	s.UserDispatcher.DispatchPositionUpdated(
		req.SellerID,
		sellerPositionEvent,
	)

	s.UserDispatcher.DispatchTradeExecuted(
		tradeEvent,
	)

	if buyFilled {
		s.UserDispatcher.DispatchOrderFilled(
			buyOrderEvent,
		)
	} else {
		s.UserDispatcher.DispatchOrderPartiallyFilled(
			buyOrderEvent,
		)
	}

	if sellFilled {
		s.UserDispatcher.DispatchOrderFilled(
			sellOrderEvent,
		)
	} else {
		s.UserDispatcher.DispatchOrderPartiallyFilled(
			sellOrderEvent,
		)
	}

	return nil
}