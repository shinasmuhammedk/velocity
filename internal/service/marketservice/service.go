package marketservice

import (
	"context"
	"velocity/internal/analytics/candles"
	"velocity/internal/analytics/stats"
	"velocity/internal/domain/depth"
	"velocity/internal/engine/registry"
	"velocity/internal/infrastructure/redis"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/pkg/errors"
)

type Service struct {
	registry   *registry.Registry
	symbolRepo repository.SymbolRepository
	tradeRepo  repository.TradeRepository

	statsService  *stats.Service
	candleService *candles.Service
	marketCache   *redis.MarketCache
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
	marketCache *redis.MarketCache,
) *Service {
	return &Service{
		registry:      reg,
		symbolRepo:    symbolRepo,
		tradeRepo:     tradeRepo,
		statsService:  statsService,
		candleService: candleService,
		marketCache:   marketCache,
	}
}

type OrderBook struct {
	Symbol string
	Bids   []depth.Level
	Asks   []depth.Level
}

func (s *Service) GetOrderBook(
	ctx context.Context,
	symbol string,
	limit int,
) (*OrderBook, error) {

	if _, err := s.symbolRepo.GetBySymbol(ctx, symbol); err != nil {
		return nil, errors.ErrSymbolNotFound
	}

	// Cache can satisfy requests up to the configured cached depth.
	if limit <= redis.MarketOrderBookDepth {

		var cached OrderBook

		if err := s.marketCache.GetOrderBook(
			ctx,
			symbol,
			&cached,
		); err == nil {

			if limit < len(cached.Bids) {
				cached.Bids = cached.Bids[:limit]
			}

			if limit < len(cached.Asks) {
				cached.Asks = cached.Asks[:limit]
			}

			return &cached, nil
		}
	}

	engine := s.registry.Get(symbol)

	if engine == nil {
		return nil, errors.ErrSymbolNotFound
	}

	book := engine.OrderBook()

	// Requests above the cached depth are served directly
	// from the live order book.
	if limit > redis.MarketOrderBookDepth {
		return &OrderBook{
			Symbol: symbol,
			Bids:   book.BidLevels(limit),
			Asks:   book.AskLevels(limit),
		}, nil
	}

	// Redis miss/error: fetch the configured cache depth
	// from the live order book.
	result := &OrderBook{
		Symbol: symbol,
		Bids:   book.BidLevels(redis.MarketOrderBookDepth),
		Asks:   book.AskLevels(redis.MarketOrderBookDepth),
	}

	// Best-effort cache write. Redis failure must not
	// make the market-data endpoint unavailable.
	_ = s.marketCache.SetOrderBook(
		ctx,
		symbol,
		result,
	)

	// Return exactly what the caller requested.
	if limit < len(result.Bids) {
		result.Bids = result.Bids[:limit]
	}

	if limit < len(result.Asks) {
		result.Asks = result.Asks[:limit]
	}

	return result, nil
}

func (s *Service) GetTicker(
	ctx context.Context,
	symbol string,
) (*Ticker, error) {

	if _, err := s.symbolRepo.GetBySymbol(context.Background(), symbol); err != nil {
		return nil, errors.ErrSymbolNotFound
	}

	var cached Ticker

	if err := s.marketCache.GetTicker(ctx, symbol, &cached); err == nil {
		return &cached, nil
	}

	engine := s.registry.Get(symbol)
	if engine == nil {
		return nil, errors.ErrSymbolNotFound
	}

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

	ticker := &Ticker{
		Symbol:    symbol,
		LastPrice: engine.LastTradePrice(),
		BestBid:   bid,
		BestAsk:   ask,
		Spread:    spread,
	}

	_ = s.marketCache.SetTicker(ctx, symbol, ticker)

	return ticker, nil

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


// -------------------------
// Admin
// -------------------------

type CreateSymbolRequest struct {
	Symbol      string
	DisplayName string
	BaseAsset   string
	QuoteAsset  string
	TickSize    int64
	LotSize     int64
	IsActive    bool
}

// CreateSymbol registers a new tradeable symbol.
//
// It does not start a matching engine for the symbol - the engine
// registry (Registry.Get) creates one lazily on first order/lookup,
// so nothing further is needed here for the symbol to become tradeable
// once IsActive is true.
func (s *Service) CreateSymbol(
	ctx context.Context,
	req CreateSymbolRequest,
) (generated.Symbol, error) {

	// Existence is checked before insert rather than relying on the
	// DB unique constraint's error code, to keep this consistent with
	// the rest of the service layer. This leaves a small race window
	// between two concurrent creates of the same symbol; acceptable
	// here since this is an admin-only, low-frequency operation, and
	// the DB unique constraint on `symbol` still prevents a duplicate
	// row from actually being persisted either way.
	if _, err := s.symbolRepo.GetBySymbol(ctx, req.Symbol); err == nil {
		return generated.Symbol{}, errors.ErrSymbolAlreadyExists
	}

	return s.symbolRepo.Create(
		ctx,
		generated.CreateSymbolParams{
			Symbol:      req.Symbol,
			DisplayName: req.DisplayName,
			BaseAsset:   req.BaseAsset,
			QuoteAsset:  req.QuoteAsset,
			TickSize:    req.TickSize,
			LotSize:     req.LotSize,
			IsActive:    req.IsActive,
		},
	)
}

// UpdateSymbolStatus enables or disables trading for an existing symbol.
func (s *Service) UpdateSymbolStatus(
	ctx context.Context,
	symbol string,
	isActive bool,
) error {

	if _, err := s.symbolRepo.GetBySymbol(ctx, symbol); err != nil {
		return errors.ErrSymbolNotFound
	}

	return s.symbolRepo.UpdateStatus(
		ctx,
		generated.UpdateSymbolStatusParams{
			Symbol:   symbol,
			IsActive: isActive,
		},
	)
}
