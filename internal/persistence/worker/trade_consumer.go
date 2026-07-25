package worker

import (
	"context"

	"velocity/internal/domain/trade"
	"velocity/internal/marketdata"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/service/settlementservice"

	"github.com/google/uuid"
)

type TradeConsumer struct {
	settlement *settlementservice.Service
	symbolRepo repository.SymbolRepository

	dispatcher   *marketdata.Broadcaster
	orderBookFor OrderBookProvider
}

func NewTradeConsumer(
	settlement *settlementservice.Service,
	symbolRepo repository.SymbolRepository,
	dispatcher *marketdata.Broadcaster,
	provider OrderBookProvider,
) *TradeConsumer {

	return &TradeConsumer{
		settlement:   settlement,
		symbolRepo:   symbolRepo,
		dispatcher:   dispatcher,
		orderBookFor: provider,
	}
}

func (c *TradeConsumer) Start(
	ctx context.Context,
	trades <-chan *trade.Trade,
) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case t := <-trades:
				if t == nil {
					continue
				}

				symbol, err := c.symbolRepo.GetBySymbol(
					ctx,
					t.Symbol,
				)
				if err != nil {
					// TODO: log
					continue
				}
				// Persist trade
				if err := c.settlement.Settle(
					ctx,
					settlementservice.SettlementRequest{
						TradeID:     t.ID,
						BuyOrderID:  uuid.MustParse(t.BuyOrderID),
						SellOrderID: uuid.MustParse(t.SellOrderID),
						BuyerID:     uuid.MustParse(t.BuyerID),
						SellerID:    uuid.MustParse(t.SellerID),
						Symbol:      t.Symbol,
						Price:       t.Price,
						Quantity:    t.Quantity,
                        
						BaseAsset:  symbol.BaseAsset,
						QuoteAsset: symbol.QuoteAsset,
					},
				); err != nil {
					// TODO:
					// retry
					// dead letter queue
					// logging
					continue
				}

				// Publish trade event
				book := c.orderBookFor(t.Symbol)

				if book != nil {
					c.dispatcher.DispatchTrade(
						t,
						book,
					)
				}
			}
		}
	}()
}
