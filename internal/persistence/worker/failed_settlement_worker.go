package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/service/settlementservice"
)

type FailedSettlementWorker struct {
	settlement           *settlementservice.Service
	failedSettlementRepo repository.FailedSettlementRepository
	symbolRepo           repository.SymbolRepository

	logger *zap.Logger

	interval time.Duration
}

func NewFailedSettlementWorker(
	settlement *settlementservice.Service,
	failedSettlementRepo repository.FailedSettlementRepository,
	symbolRepo repository.SymbolRepository,
	logger *zap.Logger,
	interval time.Duration,
) *FailedSettlementWorker {

	return &FailedSettlementWorker{
		settlement:           settlement,
		failedSettlementRepo: failedSettlementRepo,
		symbolRepo:           symbolRepo,
		logger:               logger,
		interval:             interval,
	}
}

func (w *FailedSettlementWorker) Start(ctx context.Context) {

	go func() {

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		w.logger.Info(
			"failed settlement worker started",
			zap.Duration("interval", w.interval),
		)

		// Process existing failures immediately on startup.
		w.process(ctx)

		for {
			select {

			case <-ctx.Done():
				w.logger.Info("failed settlement worker stopped")
				return

			case <-ticker.C:
				w.process(ctx)
			}
		}

	}()

}

func (w *FailedSettlementWorker) process(ctx context.Context) {

	failures, err := w.failedSettlementRepo.ListUnresolved(ctx)

	if err != nil {
		w.logger.Error(
			"failed settlement worker: unable to load unresolved settlements",
			zap.Error(err),
		)
		return
	}

	if len(failures) == 0 {
		return
	}

	w.logger.Warn(
		"failed settlement worker: unresolved settlements found",
		zap.Int("count", len(failures)),
	)

	for _, failed := range failures {

		if err := w.retry(ctx, failed); err != nil {
			w.logger.Error(
				"failed settlement retry unsuccessful",
				zap.String("failure_id", failed.ID.String()),
				zap.Int64("trade_id", failed.TradeID),
				zap.Int32("retry_count", failed.RetryCount),
				zap.Error(err),
			)

			if incrementErr := w.failedSettlementRepo.IncrementRetryCount(
				ctx,
				failed.ID,
			); incrementErr != nil {

				w.logger.Error(
					"failed settlement worker: unable to increment retry count",
					zap.String("failure_id", failed.ID.String()),
					zap.Error(incrementErr),
				)
			}

			continue
		}

		if err := w.failedSettlementRepo.Resolve(
			ctx,
			failed.ID,
		); err != nil {

			w.logger.Error(
				"failed settlement worker: settlement succeeded but failed to mark record resolved",
				zap.String("failure_id", failed.ID.String()),
				zap.Int64("trade_id", failed.TradeID),
				zap.Error(err),
			)

			continue
		}

		w.logger.Info(
			"failed settlement successfully recovered",
			zap.String("failure_id", failed.ID.String()),
			zap.Int64("trade_id", failed.TradeID),
			zap.Int32("retry_count", failed.RetryCount),
		)
	}
}

func (w *FailedSettlementWorker) retry(
	ctx context.Context,
	failed generated.FailedSettlement,
) error {

	symbol, err := w.symbolRepo.GetBySymbol(
		ctx,
		failed.Symbol,
	)
	if err != nil {
		return err
	}

	return w.settlement.Settle(
		ctx,
		settlementservice.SettlementRequest{
			TradeID: failed.TradeID,

			BuyOrderID:  failed.BuyOrderID,
			SellOrderID: failed.SellOrderID,

			BuyerID:  failed.BuyerID,
			SellerID: failed.SellerID,

			Symbol: failed.Symbol,

			BaseAsset:  symbol.BaseAsset,
			QuoteAsset: symbol.QuoteAsset,

			Price:    failed.Price,
			Quantity: failed.Quantity,
		},
	)
}
