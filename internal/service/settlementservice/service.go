package settlementservice

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

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
			walletService.Deposit(
				ctx,
				req.SellerID,
				req.QuoteAsset,
				req.Price*req.Quantity,
			)

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

			// 8. Update order status
			if err := orderRepo.UpdateStatus(
				ctx,
				generated.UpdateOrderStatusParams{
					ID:     req.BuyOrderID,
					Status: "FILLED",
				},
			); err != nil {
				return err
			}

			if err := orderRepo.UpdateStatus(
				ctx,
				generated.UpdateOrderStatusParams{
					ID:     req.SellOrderID,
					Status: "FILLED",
				},
			); err != nil {
				return err
			}

			return nil
		},
	)
}
