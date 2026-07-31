package settlementservice

import (
	"context"

	"velocity/internal/domain/order"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/userstream"
	"velocity/pkg/constants"

	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"
	"velocity/internal/service/walletservice"

	"github.com/jackc/pgx/v5"
)

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
		buyerBalance  userstream.BalanceUpdate
		sellerBalance userstream.BalanceUpdate

		buyerPositionEvent  userstream.PositionUpdate
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

			exists, err := tradeRepo.TradeExists(
				ctx,
				req.TradeID,
			)

			if err != nil {
				return err
			}

			if exists {
				tradeAlreadySettled = true

				return nil
			}

			err = walletService.ConsumeLockedFunds(
				ctx,
				req.BuyerID,
				req.QuoteAsset,
				req.Price*req.Quantity,
			)

			if err != nil {
				return err
			}

			err = walletService.ConsumeLockedFunds(
				ctx,
				req.SellerID,
				req.BaseAsset,
				req.Quantity,
			)

			if err != nil {
				return err
			}

			err = walletService.Deposit(
				ctx,
				req.BuyerID,
				req.BaseAsset,
				req.Quantity,
			)

			if err != nil {
				return err
			}

			err = walletService.Deposit(
				ctx,
				req.SellerID,
				req.QuoteAsset,
				req.Price*req.Quantity,
			)

			if err != nil {
				return err
			}

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

			_, err = tradeRepo.Create(
				ctx,
				generated.CreateTradeParams{
					ID:          req.TradeID,
					BuyOrderID:  req.BuyOrderID,
					SellOrderID: req.SellOrderID,
					BuyerID:     req.BuyerID,
					SellerID:    req.SellerID,
					Symbol:      req.Symbol,
					Price:       req.Price,
					Quantity:    req.Quantity,
				},
			)
            
            if err != nil {
				return err
			}

			tradeEvent = userstream.TradeExecution{
				TradeID: req.TradeID.String(),

				BuyOrderID:  req.BuyOrderID.String(),
				SellOrderID: req.SellOrderID.String(),

				BuyerID:  req.BuyerID.String(),
				SellerID: req.SellerID.String(),

				Symbol: req.Symbol,

				Price:    req.Price,
				Quantity: req.Quantity,
			}

			

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
				ID:        buyOrder.ID.String(),
				UserID:    buyOrder.UserID.String(),
				Symbol:    buyOrder.Symbol,
				Status:    constants.OrderStatus(buyStatus),
				Price:     buyOrder.Price.Int64,
				Quantity:  buyOrder.Quantity,
				Filled:    buyFilledQty,
				Remaining: buyRemaining,
			}

			
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

			sellOrderEvent = &order.Order{
				ID:        sellOrder.ID.String(),
				UserID:    sellOrder.UserID.String(),
				Symbol:    sellOrder.Symbol,
				Status:    constants.OrderStatus(sellStatus),
				Price:     sellOrder.Price.Int64,
				Quantity:  sellOrder.Quantity,
				Filled:    sellFilledQty,
				Remaining: sellRemaining,
			}

			if err != nil {
				return err
			}

			buyerBaseWallet, err := walletRepo.Get(
				ctx,
				req.BuyerID,
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

			buyerBalance = userstream.BalanceUpdate{
				Asset:     req.BaseAsset,
				Available: buyerBaseWallet.Available,
				Locked:    buyerBaseWallet.Locked,
			}

			sellerBalance = userstream.BalanceUpdate{
				Asset:     req.QuoteAsset,
				Available: sellerQuoteWallet.Available,
				Locked:    sellerQuoteWallet.Locked,
			}

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

	if err != nil {
		return err
	}
	if tradeAlreadySettled {
		return nil
	}

	s.UserDispatcher.DispatchBalanceUpdated(
		req.BuyerID.String(),
		buyerBalance,
	)

	s.UserDispatcher.DispatchBalanceUpdated(
		req.SellerID.String(),
		sellerBalance,
	)

	s.UserDispatcher.DispatchPositionUpdated(
		req.BuyerID.String(),
		buyerPositionEvent,
	)

	s.UserDispatcher.DispatchPositionUpdated(
		req.SellerID.String(),
		sellerPositionEvent,
	)

	s.UserDispatcher.DispatchTradeExecuted(
		tradeEvent,
	)

	if buyFilled {
		s.UserDispatcher.DispatchOrderFilled(buyOrderEvent)
	} else {
		s.UserDispatcher.DispatchOrderPartiallyFilled(buyOrderEvent)
	}

	if sellFilled {
		s.UserDispatcher.DispatchOrderFilled(sellOrderEvent)
	} else {
		s.UserDispatcher.DispatchOrderPartiallyFilled(sellOrderEvent)
	}

	return nil
}
