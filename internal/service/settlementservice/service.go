package settlementservice

import (
	"context"

	"velocity/internal/persistence/postgres/generated"
	"velocity/pkg/constants"

	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"
	"velocity/internal/service/walletservice"

	"github.com/jackc/pgx/v5"
)

type Service struct {
	txManager tx.Manager
}

func New(
	txManager tx.Manager,
) *Service {

	return &Service{
		txManager: txManager,
	}
}

func (s *Service) Settle(
	ctx context.Context,
	req SettlementRequest,
) error {

	return s.txManager.WithTransaction(
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
				// Trade already settled.
				// Safe to return because settlement has already completed.
				return nil
			}

			// 1. Consume buyer's locked quote asset
			if err := walletService.ConsumeLockedFunds(
				ctx,
				req.BuyerID,
				req.QuoteAsset,
				req.Price*req.Quantity,
			); err != nil {
				return err
			}

			// 2. Consume seller's locked base asset
			if err := walletService.ConsumeLockedFunds(
				ctx,
				req.SellerID,
				req.BaseAsset,
				req.Quantity,
			); err != nil {
				return err
			}

			// 3. Credit buyer with purchased asset
			if err := walletService.Deposit(
				ctx,
				req.BuyerID,
				req.BaseAsset,
				req.Quantity,
			); err != nil {
				return err
			}

			// 4. Credit seller with quote asset
			if err := walletService.Deposit(
				ctx,
				req.SellerID,
				req.QuoteAsset,
				req.Price*req.Quantity,
			); err != nil {
				return err
			}

			// 5. Update buyer position
			if err := positionRepo.Upsert(
				ctx,
				generated.UpsertPositionParams{
					UserID:   req.BuyerID,
					Symbol:   req.Symbol,
					Quantity: req.Quantity,
				},
			); err != nil {
				return err
			}

			// 6. Update seller position

			if err := positionRepo.Upsert(
				ctx,
				generated.UpsertPositionParams{
					UserID:   req.SellerID,
					Symbol:   req.Symbol,
					Quantity: -req.Quantity,
				},
			); err != nil {
				return err
			}

			// 7. Persist trade
			if _, err := tradeRepo.Create(
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
			); err != nil {
				return err
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

			// 8. Update order status
			buyRemaining := buyOrder.Remaining - req.Quantity
			buyFilled := buyOrder.Filled + req.Quantity

			buyStatus := string(constants.OrderStatusPartiallyFilled)
			if buyRemaining == 0 {
				buyStatus = string(constants.OrderStatusFilled)
			}

			if err := orderRepo.UpdateOrderAfterTrade(
				ctx,
				generated.UpdateOrderAfterTradeParams{
					ID:        buyOrder.ID,
					Remaining: buyRemaining,
					Filled:    buyFilled,
					Status:    buyStatus,
				},
			); err != nil {
				return err
			}

			sellRemaining := sellOrder.Remaining - req.Quantity
			sellFilled := sellOrder.Filled + req.Quantity

			sellStatus := string(constants.OrderStatusPartiallyFilled)
			if sellRemaining == 0 {
				sellStatus = string(constants.OrderStatusFilled)
			}

			if err := orderRepo.UpdateOrderAfterTrade(
				ctx,
				generated.UpdateOrderAfterTradeParams{
					ID:        sellOrder.ID,
					Remaining: sellRemaining,
					Filled:    sellFilled,
					Status:    sellStatus,
				},
			); err != nil {
				return err
			}

			return nil
		},
	)
}
