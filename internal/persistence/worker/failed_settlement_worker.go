package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/service/settlementservice"
)

const maxSettlementRetries int32 = 10

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
			zap.Int32("max_retries", maxSettlementRetries),
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
		// Defensive check.
		//
		// ListUnresolved already excludes dead records, but keeping
		// this check here prevents accidental retries if the query
		// changes in the future.
		if failed.IsDead {
			continue
		}

		if failed.RetryCount >= maxSettlementRetries {
			w.markDead(ctx, failed)
			continue
		}

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

				continue
			}

			// failed.RetryCount represents the value before the
			// increment, so calculate the new value explicitly.
			newRetryCount := failed.RetryCount + 1

			if newRetryCount >= maxSettlementRetries {
				// The settlement has now exhausted its retry budget.
				w.markDeadAfterRetry(ctx, failed, newRetryCount)
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

func (w *FailedSettlementWorker) markDead(
	ctx context.Context,
	failed generated.FailedSettlement,
) {
	if err := w.failedSettlementRepo.MarkDead(
		ctx,
		failed.ID,
	); err != nil {
		w.logger.Error(
			"failed settlement worker: unable to mark settlement dead",
			zap.String("failure_id", failed.ID.String()),
			zap.Int64("trade_id", failed.TradeID),
			zap.Int32("retry_count", failed.RetryCount),
			zap.Error(err),
		)

		return
	}

	w.logger.Error(
		"FAILED SETTLEMENT MOVED TO DEAD LETTER STATE",
		zap.String("failure_id", failed.ID.String()),
		zap.Int64("trade_id", failed.TradeID),
		zap.String("symbol", failed.Symbol),
		zap.Int32("retry_count", failed.RetryCount),
		zap.String("error_message", failed.ErrorMessage),
	)
}

func (w *FailedSettlementWorker) markDeadAfterRetry(
	ctx context.Context,
	failed generated.FailedSettlement,
	retryCount int32,
) {
	if err := w.failedSettlementRepo.MarkDead(
		ctx,
		failed.ID,
	); err != nil {
		w.logger.Error(
			"failed settlement worker: unable to mark exhausted settlement dead",
			zap.String("failure_id", failed.ID.String()),
			zap.Int64("trade_id", failed.TradeID),
			zap.Int32("retry_count", retryCount),
			zap.Error(err),
		)

		return
	}

	w.logger.Error(
		"FAILED SETTLEMENT RETRY LIMIT EXCEEDED - MOVED TO DEAD LETTER STATE",
		zap.String("failure_id", failed.ID.String()),
		zap.Int64("trade_id", failed.TradeID),
		zap.String("symbol", failed.Symbol),
		zap.Int32("retry_count", retryCount),
		zap.String("error_message", failed.ErrorMessage),
	)
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
