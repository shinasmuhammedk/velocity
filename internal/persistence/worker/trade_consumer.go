package worker

import (
	"context"

	"go.uber.org/zap"

	"velocity/internal/domain/trade"
	"velocity/internal/marketdata"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/service/settlementservice"
)

type TradeConsumer struct {
	settlement           *settlementservice.Service
	symbolRepo           repository.SymbolRepository
	failedSettlementRepo repository.FailedSettlementRepository

	dispatcher   *marketdata.Broadcaster
	orderBookFor OrderBookProvider

	logger *zap.Logger
}

func NewTradeConsumer(
	settlement *settlementservice.Service,
	symbolRepo repository.SymbolRepository,
	failedSettlementRepo repository.FailedSettlementRepository,
	dispatcher *marketdata.Broadcaster,
	provider OrderBookProvider,
	logger *zap.Logger,
) *TradeConsumer {

	return &TradeConsumer{
		settlement:           settlement,
		symbolRepo:           symbolRepo,
		failedSettlementRepo: failedSettlementRepo,
		dispatcher:           dispatcher,
		orderBookFor:         provider,
		logger:               logger,
	}
}

func (c *TradeConsumer) Start(
	ctx context.Context,
	trades <-chan trade.Trade,
) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case t := <-trades:

				// if t == nil {
				// 	c.logger.Warn("trade consumer received nil trade")
				// 	continue
				// }

				c.logger.Debug(
					"trade received by consumer",
					zap.String("symbol", t.Symbol),
					zap.Int64("trade_id", t.ID),
				)

				symbol, err := c.symbolRepo.GetBySymbol(
					ctx,
					t.Symbol,
				)
				if err != nil {
					c.logger.Error(
						"trade consumer: symbol lookup failed, trade dropped",
						zap.String("symbol", t.Symbol),
						zap.Int64("trade_id", t.ID),
						zap.Error(err),
					)
					continue
				}

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
					// The trade has already matched at the engine level
					// at this point. Rather than only logging and
					// dropping it, the failure is persisted so it can
					// be inspected and retried later - see
					// failed_settlements table / ListUnresolved.
					//
					// A background context is used for this insert
					// deliberately: if ctx is already done (e.g.
					// shutdown in progress), that must not prevent
					// the failure record itself from being saved.
					c.logger.Error(
						"trade consumer: settlement failed, recording for retry",
						zap.String("symbol", t.Symbol),
						zap.Int64("trade_id", t.ID),
						zap.Int64("buy_order_id", t.BuyOrderID),
						zap.Int64("sell_order_id", t.SellOrderID),
						zap.Error(err),
					)

					if _, recordErr := c.failedSettlementRepo.Create(
						context.Background(),
						generated.CreateFailedSettlementParams{
							TradeID:      t.ID,
							BuyOrderID:   t.BuyOrderID,
							SellOrderID:  t.SellOrderID,
							BuyerID:      t.BuyerID,
							SellerID:     t.SellerID,
							Symbol:       t.Symbol,
							Price:        t.Price,
							Quantity:     t.Quantity,
							ErrorMessage: err.Error(),
						},
					); recordErr != nil {
						// This is the worst case: settlement failed
						// AND we couldn't even record that it failed.
						// Nothing further to fall back to but the log -
						// this line is the one to alert on.
						c.logger.Error(
							"trade consumer: FAILED TO RECORD settlement failure, trade fully unrecoverable",
							zap.String("symbol", t.Symbol),
							zap.Int64("trade_id", t.ID),
							zap.Error(recordErr),
						)
					}

					continue
				}

				c.logger.Info(
					"trade settled",
					zap.String("symbol", t.Symbol),
					zap.Int64("trade_id", t.ID),
				)

				book := c.orderBookFor(t.Symbol)

				if book == nil {
					c.logger.Warn(
						"trade consumer: no order book found for symbol, skipping dispatch",
						zap.String("symbol", t.Symbol),
					)
					continue
				}

				c.dispatcher.DispatchTrade(
					t,
					book,
				)
			}
		}
	}()
}
