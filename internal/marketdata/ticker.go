package marketdata

import "velocity/pkg/constants"

func NewTickerMessage(
	symbol string,
	lastPrice int64,
	bestBid int64,
	bestAsk int64,
) Message {

	spread := int64(0)
	mid := int64(0)

	if bestBid > 0 && bestAsk > 0 {
		spread = bestAsk - bestBid
		mid = (bestBid + bestAsk) / 2
	}

	return Message{
		Type:   constants.MessageTicker,
		Symbol: symbol,
		Data: TickerMessage{
			LastPrice: lastPrice,
			BestBid:   bestBid,
			BestAsk:   bestAsk,
			Spread:    spread,
			MidPrice:  mid,
		},
	}
}
