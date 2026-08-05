package marketservice

import (
	"context"
	"velocity/internal/analytics/candles"
	"velocity/internal/analytics/stats"
	"velocity/internal/domain/depth"
	"velocity/internal/engine/registry"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/pkg/errors"

)

type Service struct {
	registry  *registry.Registry
    symbolRepo repository.SymbolRepository
	tradeRepo repository.TradeRepository
    
    statsService *stats.Service
    candleService *candles.Service
}

type Ticker struct {
	Symbol    string `json:"symbol"`
	LastPrice int64  `json:"last_price"`
	BestBid   int64  `json:"best_bid"`
	BestAsk   int64  `json:"best_ask"`
	Spread    int64  `json:"spread"`
	BidSize   int64  `json:"bid_size"`
	AskSize   int64  `json:"ask_size"`
}

func New(
	reg *registry.Registry,
    symbolRepo repository.SymbolRepository,
	tradeRepo repository.TradeRepository,
    statsService *stats.Service,
    candleService *candles.Service,
) *Service {
	return &Service{
		registry:  reg,
        symbolRepo: symbolRepo,
		tradeRepo: tradeRepo,
        statsService: statsService,
        candleService: candleService,
	}
}

type OrderBook struct {
	Symbol string
	Bids   []depth.Level
	Asks   []depth.Level
}

func (s *Service) GetOrderBook(symbol string, limit int) (*OrderBook, error) {

	if _, err := s.symbolRepo.GetBySymbol(context.Background(), symbol); err != nil {
		return nil, errors.ErrSymbolNotFound
	}

	engine := s.registry.Get(symbol)
	book := engine.OrderBook()

	return &OrderBook{
		Symbol: symbol,
		Bids:   book.BidLevels(limit),
		Asks:   book.AskLevels(limit),
	}, nil
}


func (s *Service) GetTicker(
	symbol string,
) (*Ticker, error) {

	if _, err := s.symbolRepo.GetBySymbol(context.Background(), symbol); err != nil {
		return nil, errors.ErrSymbolNotFound
	}

	engine := s.registry.Get(symbol)

	book := engine.OrderBook()

	bestBid := book.BestBid()
	bestAsk := book.BestAsk()

	var bid int64
	var ask int64

	if bestBid != nil {
		bid = bestBid.Price
	}

	if bestAsk != nil {
		ask = bestAsk.Price
	}

	var spread int64
	if bid != 0 && ask != 0 {
		spread = ask - bid
	}

	return &Ticker{
		Symbol:    symbol,
		LastPrice: engine.LastTradePrice(),
		BestBid:   bid,
		BestAsk:   ask,
		Spread:    spread,
	}, nil
}

func (s *Service) GetRecentTrades(
	ctx context.Context,
	symbol string,
) ([]generated.Trade, error) {

	if _, err := s.symbolRepo.GetBySymbol(ctx, symbol); err != nil {
		return nil, errors.ErrSymbolNotFound
	}

	return s.tradeRepo.ListBySymbol(
		ctx,
		symbol,
	)
}


func (s *Service) GetSymbols(
    ctx context.Context,
) ([]generated.Symbol, error) {

    return s.symbolRepo.List(
        ctx,
    )
}


func (s *Service) GetUserTrades(
	ctx context.Context,
	userID int64,
) ([]generated.Trade, error) {

	return s.tradeRepo.ListByUser(
		ctx,
		userID,
	)
}

func (s *Service) GetMarketStats(symbol string) (*stats.MarketStats, error) {

	if _, err := s.symbolRepo.GetBySymbol(context.Background(), symbol); err != nil {
		return nil, errors.ErrSymbolNotFound
	}

	marketStats, ok := s.statsService.Get(symbol)
	if !ok {
		return &stats.MarketStats{Symbol: symbol}, nil // valid symbol, just no trades yet
	}

	return marketStats, nil
}


func (s *Service) GetCandles(
	symbol string,
	interval candles.Interval,
) ([]*candles.Candle, error) {

	if _, err := s.symbolRepo.GetBySymbol(context.Background(), symbol); err != nil {
		return nil, errors.ErrSymbolNotFound
	}

	candleList, ok := s.candleService.Get(
		symbol,
		interval,
	)

	if !ok {
		return []*candles.Candle{}, nil
	}

	return candleList, nil
}