package marketdataservice

import (
	"context"
	"fmt"
	"velocity/internal/domain/depth"
	"velocity/internal/engine/registry"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/pkg/errors"

	"github.com/google/uuid"
)

type Service struct {
	registry  *registry.Registry
    symbolRepo repository.SymbolRepository
	tradeRepo repository.TradeRepository
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
) *Service {
	return &Service{
		registry:  reg,
        symbolRepo: symbolRepo,
		tradeRepo: tradeRepo,
	}
}

type OrderBook struct {
	Symbol string
	Bids   []depth.Level
	Asks   []depth.Level
}

func (s *Service) GetOrderBook(
	symbol string,
	limit int,
) (*OrderBook, error) {

	engine := s.registry.Get(symbol)
	if engine == nil {
		return nil, errors.ErrSymbolNotFound
	}

	book := engine.OrderBook()

	// DEBUG
	fmt.Println("========== ORDERBOOK QUERY ==========")
	fmt.Println("Symbol:", symbol)
	fmt.Println("Ask Levels:", len(book.AskLevels(limit)))
	fmt.Println("Bid Levels:", len(book.BidLevels(limit)))
	fmt.Println("=====================================")

	return &OrderBook{
		Symbol: symbol,
		Bids:   book.BidLevels(limit),
		Asks:   book.AskLevels(limit),
	}, nil
}

func (s *Service) GetTicker(
	symbol string,
) (*Ticker, error) {

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

	engine := s.registry.Get(symbol)
	if engine == nil {
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
	userID uuid.UUID,
) ([]generated.Trade, error) {

	return s.tradeRepo.ListByUser(
		ctx,
		userID,
	)
}