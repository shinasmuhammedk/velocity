package worker

import (
	"context"

	"velocity/internal/domain/trade"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"
	"velocity/pkg/constants"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type tradePersistenceWorker struct {
	txManager tx.Manager

	orderRepo    repository.OrderRepository
	tradeRepo    repository.TradeRepository
	positionRepo repository.PositionRepository
}

func NewTradePersistenceWorker(
	txManager tx.Manager,
	orderRepo repository.OrderRepository,
	tradeRepo repository.TradeRepository,
	positionRepo repository.PositionRepository,
) TradePersistenceWorker {

	return &tradePersistenceWorker{
		txManager: txManager,

		orderRepo:    orderRepo,
		tradeRepo:    tradeRepo,
		positionRepo: positionRepo,
	}
}

func (w *tradePersistenceWorker) ProcessTrade(
	ctx context.Context,
	t *trade.Trade,
) error {

	return w.txManager.WithTransaction(
		ctx,
		func(tx pgx.Tx) error {

			queries := generated.New(tx)

			_, err := queries.CreateTrade(
				ctx,
				generated.CreateTradeParams{
					ID:          t.ID,
					BuyOrderID:  uuid.MustParse(t.BuyOrderID),
					SellOrderID: uuid.MustParse(t.SellOrderID),
					BuyerID:     uuid.MustParse(t.BuyerID),
					SellerID:    uuid.MustParse(t.SellerID),
					Symbol:      t.Symbol,
					Price:       t.Price,
					Quantity:    t.Quantity,
					ExecutedAt:  t.ExecutedAt,
				},
			)
			if err != nil {
				return err
			}

			buyOrder, err := queries.GetOrderByID(
				ctx,
				uuid.MustParse(t.BuyOrderID),
			)
			if err != nil {
				return err
			}

			sellOrder, err := queries.GetOrderByID(
				ctx,
				uuid.MustParse(t.SellOrderID),
			)
			if err != nil {
				return err
			}

			buyRemaining := buyOrder.Remaining - t.Quantity
			buyFilled := buyOrder.Filled + t.Quantity

			buyStatus := string(constants.OrderStatusPartiallyFilled)
			if buyRemaining == 0 {
				buyStatus = "FILLED"
			}

			err = queries.UpdateOrderAfterTrade(
				ctx,
				generated.UpdateOrderAfterTradeParams{
					ID:        buyOrder.ID,
					Remaining: buyRemaining,
					Filled:    buyFilled,
					Status:    buyStatus,
				},
			)
			if err != nil {
				return err
			}

			sellRemaining := sellOrder.Remaining - t.Quantity
			sellFilled := sellOrder.Filled + t.Quantity

			sellStatus := string(constants.OrderStatusPartiallyFilled)
			if sellRemaining == 0 {
				sellStatus = "FILLED"
			}

			err = queries.UpdateOrderAfterTrade(
				ctx,
				generated.UpdateOrderAfterTradeParams{
					ID:        sellOrder.ID,
					Remaining: sellRemaining,
					Filled:    sellFilled,
					Status:    sellStatus,
				},
			)
			if err != nil {
				return err
			}

			err = queries.UpsertPosition(
				ctx,
				generated.UpsertPositionParams{
					UserID:   uuid.MustParse(t.BuyerID),
					Symbol:   t.Symbol,
					Quantity: t.Quantity,
				},
			)
			if err != nil {
				return err
			}

			err = queries.UpsertPosition(
				ctx,
				generated.UpsertPositionParams{
					UserID:   uuid.MustParse(t.SellerID),
					Symbol:   t.Symbol,
					Quantity: -t.Quantity,
				},
			)
			if err != nil {
				return err
			}

			return nil
		},
	)
}
