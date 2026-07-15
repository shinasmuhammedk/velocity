package marketdata

import "velocity/internal/domain/depth"

type TopOfBookProvider interface {
	BestBidPrice() int64
	BestAskPrice() int64
}

type DepthProvider interface {
	BidLevels(limit int) []depth.Level
	AskLevels(limit int) []depth.Level
}

type MarketDataProvider interface {
	TopOfBookProvider
	DepthProvider
}