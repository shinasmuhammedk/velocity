package candles

import (
	"context"
	"fmt"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
)

type BackfillService struct {
	tradeRepo repository.TradeRepository
	manager   *Manager
}

func NewBackfillService(
	tradeRepo repository.TradeRepository,
	manager *Manager,
) *BackfillService {
	return &BackfillService{
		tradeRepo: tradeRepo,
		manager:   manager,
	}
}

// BackfillSymbol rebuilds all supported candle intervals for a symbol
// from persisted trades.
//
// Trades must be processed chronologically so Manager.Update()
// reconstructs OHLCV correctly.
func (s *BackfillService) BackfillSymbol(
	ctx context.Context,
	symbol string,
) error {

	now := time.Now().UTC()
	todayStart := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		time.UTC,
	)

	trades, err := s.tradeRepo.ListBySymbolSinceAsc(
		ctx,
		symbol,
		todayStart,
	)
	if err != nil {
		return fmt.Errorf(
			"load trades for candle backfill %s: %w",
			symbol,
			err,
		)
	}

	for _, trade := range trades {
		s.applyTrade(trade)
	}

	return nil
}

func (s *BackfillService) applyTrade(
	trade generated.Trade,
) {
	s.manager.Update(
		trade.Symbol,
		trade.Price,
		trade.Quantity,
		trade.ExecutedAt,
	)
}

// BackfillSymbols rebuilds candle history for multiple symbols.
func (s *BackfillService) BackfillSymbols(
	ctx context.Context,
	symbols []generated.Symbol,
) error {

	for _, symbol := range symbols {
		if err := s.BackfillSymbol(
			ctx,
			symbol.Symbol,
		); err != nil {
			return err
		}
	}

	return nil
}
