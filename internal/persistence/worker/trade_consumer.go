package worker

import (
	"context"
	"fmt"

	"velocity/internal/domain/trade"
	"velocity/internal/marketdata"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/service/settlementservice"

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

				fmt.Println("TRADE RECEIVED BY CONSUMER")

				if t == nil {
					fmt.Println("TRADE IS NIL")
					continue
				}

				fmt.Println("TRADE:", t.Symbol)
				// if t == nil {
				// 	continue
				// }

				symbol, err := c.symbolRepo.GetBySymbol(
					ctx,
					t.Symbol,
				)
				if err != nil {
					// TODO: log
					continue
				}

				fmt.Println("STARTING SETTLEMENT")
				// Persist trade
				if err := c.settlement.Settle(
					ctx,
					settlementservice.SettlementRequest{
						TradeID:     t.ID,
						BuyOrderID:  t.BuyOrderID,
						SellOrderID: t.SellOrderID,
						BuyerID:     t.BuyerID,
						SellerID:    t.SellerID,
						Symbol:      t.Symbol,
						Price:       t.Price,
						Quantity:    t.Quantity,

						BaseAsset:  symbol.BaseAsset,
						QuoteAsset: symbol.QuoteAsset,
					},
				); err != nil {
					fmt.Println("SETTLEMENT ERROR:", err)
					// TODO:
					// retry
					// dead letter queue
					// logging
					continue
				}

				fmt.Println("SETTLEMENT SUCCESS")

				fmt.Println("TRADE CONSUMER: Dispatching trade")
				fmt.Println("Symbol:", t.Symbol)

				// Publish trade event
				book := c.orderBookFor(t.Symbol)

				if book == nil {
					fmt.Println("BOOK IS NIL")
				} else {
					fmt.Println("BOOK FOUND")
				}

				fmt.Println("ABOUT TO DISPATCH")

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
