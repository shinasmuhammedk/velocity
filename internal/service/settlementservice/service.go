package settlementservice

import (
	"context"
	"fmt"

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

			fmt.Println("1. TradeExists")

			exists, err := tradeRepo.TradeExists(
				ctx,
				req.TradeID,
			)

			fmt.Println("1 DONE", err)

			if err != nil {
				return err
			}

			if exists {
				fmt.Println("Trade already settled")
				return nil
			}

			fmt.Println("2. Consume buyer")

			err = walletService.ConsumeLockedFunds(
				ctx,
				req.BuyerID,
				req.QuoteAsset,
				req.Price*req.Quantity,
			)

			fmt.Println("2 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("3. Consume seller")

			err = walletService.ConsumeLockedFunds(
				ctx,
				req.SellerID,
				req.BaseAsset,
				req.Quantity,
			)

			fmt.Println("3 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("4. Deposit buyer")

			err = walletService.Deposit(
				ctx,
				req.BuyerID,
				req.BaseAsset,
				req.Quantity,
			)

			fmt.Println("4 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("5. Deposit seller")

			err = walletService.Deposit(
				ctx,
				req.SellerID,
				req.QuoteAsset,
				req.Price*req.Quantity,
			)

			fmt.Println("5 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("6. Update buyer position")

			err = positionRepo.Upsert(
				ctx,
				generated.UpsertPositionParams{
					UserID:   req.BuyerID,
					Symbol:   req.Symbol,
					Quantity: req.Quantity,
				},
			)

			fmt.Println("6 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("7. Update seller position")

			err = positionRepo.Upsert(
				ctx,
				generated.UpsertPositionParams{
					UserID:   req.SellerID,
					Symbol:   req.Symbol,
					Quantity: -req.Quantity,
				},
			)

			fmt.Println("7 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("8. Persist trade")

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

			fmt.Println("8 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("9. Get buy order")

			buyOrder, err := orderRepo.GetByID(
				ctx,
				req.BuyOrderID,
			)

			fmt.Println("9 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("10. Get sell order")

			sellOrder, err := orderRepo.GetByID(
				ctx,
				req.SellOrderID,
			)

			fmt.Println("10 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("11. Update buy order")

			buyRemaining := buyOrder.Remaining - req.Quantity
			buyFilled := buyOrder.Filled + req.Quantity

			buyStatus := string(constants.OrderStatusPartiallyFilled)
			if buyRemaining == 0 {
				buyStatus = string(constants.OrderStatusFilled)
			}

			err = orderRepo.UpdateOrderAfterTrade(
				ctx,
				generated.UpdateOrderAfterTradeParams{
					ID:        buyOrder.ID,
					Remaining: buyRemaining,
					Filled:    buyFilled,
					Status:    buyStatus,
				},
			)

			fmt.Println("11 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("12. Update sell order")

			sellRemaining := sellOrder.Remaining - req.Quantity
			sellFilled := sellOrder.Filled + req.Quantity

			sellStatus := string(constants.OrderStatusPartiallyFilled)
			if sellRemaining == 0 {
				sellStatus = string(constants.OrderStatusFilled)
			}

			err = orderRepo.UpdateOrderAfterTrade(
				ctx,
				generated.UpdateOrderAfterTradeParams{
					ID:        sellOrder.ID,
					Remaining: sellRemaining,
					Filled:    sellFilled,
					Status:    sellStatus,
				},
			)

			fmt.Println("12 DONE", err)

			if err != nil {
				return err
			}

			fmt.Println("SETTLEMENT COMPLETE")

			return nil
		},
	)
}