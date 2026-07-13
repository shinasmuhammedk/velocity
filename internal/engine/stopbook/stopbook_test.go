package stopbook

import (
	"testing"
	"velocity/internal/domain/order"
	"velocity/pkg/constants"
    apperr "velocity/pkg/errors"

	"github.com/stretchr/testify/require"
)

func TestAddBuyStopOrder(t *testing.T) {
	sb := New()

	o := &order.Order{
		ID:        "stop-buy-1",
		UserID:    "user1",
		Symbol:    "BTCUSDT",
		Side:      constants.OrderSideBuy,
		Type:      constants.StopMarketOrder,
		StopPrice: 1050,
		Status:    constants.OrderStatusPending,
	}

	sb.Add(o)

	require.Len(t, sb.buyStops, 1)
	require.Len(t, sb.buyStops[1050], 1)

	_, exists := sb.orderIndex[o.ID]
	require.True(t, exists)
}

func TestAddSellStopOrder(t *testing.T) {
	sb := New()

	o := &order.Order{
		ID:        "stop-sell-1",
		UserID:    "user1",
		Symbol:    "BTCUSDT",
		Side:      constants.OrderSideSell,
		Type:      constants.StopMarketOrder,
		StopPrice: 950,
		Status:    constants.OrderStatusPending,
	}

	sb.Add(o)

	require.Len(t, sb.sellStops, 1)
	require.Len(t, sb.sellStops[950], 1)

	_, exists := sb.orderIndex[o.ID]
	require.True(t, exists)
}

func TestTriggerBuyStops(t *testing.T) {
	sb := New()

	o1 := &order.Order{
		ID:        "buy-stop-1",
		Side:      constants.OrderSideBuy,
		Type:      constants.StopMarketOrder,
		StopPrice: 1050,
	}

	o2 := &order.Order{
		ID:        "buy-stop-2",
		Side:      constants.OrderSideBuy,
		Type:      constants.StopMarketOrder,
		StopPrice: 1060,
	}

	sb.Add(o1)
	sb.Add(o2)

	triggered := sb.Trigger(1065)

	require.Len(t, triggered, 2)

	require.Empty(t, sb.buyStops)

	_, exists := sb.orderIndex[o1.ID]
	require.False(t, exists)

	_, exists = sb.orderIndex[o2.ID]
	require.False(t, exists)
}

func TestTriggerSellStops(t *testing.T) {
	sb := New()

	o1 := &order.Order{
		ID:        "sell-stop-1",
		Side:      constants.OrderSideSell,
		Type:      constants.StopMarketOrder,
		StopPrice: 950,
	}

	o2 := &order.Order{
		ID:        "sell-stop-2",
		Side:      constants.OrderSideSell,
		Type:      constants.StopMarketOrder,
		StopPrice: 940,
	}

	sb.Add(o1)
	sb.Add(o2)

	triggered := sb.Trigger(930)

	require.Len(t, triggered, 2)

	require.Empty(t, sb.sellStops)

	_, exists := sb.orderIndex[o1.ID]
	require.False(t, exists)

	_, exists = sb.orderIndex[o2.ID]
	require.False(t, exists)
}

func TestTriggerDoesNotActivateOrdersPrematurely(t *testing.T) {
	sb := New()

	o := &order.Order{
		ID:        "buy-stop-1",
		Side:      constants.OrderSideBuy,
		Type:      constants.StopMarketOrder,
		StopPrice: 1050,
	}

	sb.Add(o)

	triggered := sb.Trigger(1040)

	require.Len(t, triggered, 0)

	_, exists := sb.orderIndex[o.ID]
	require.True(t, exists)
}

func TestCancelStopOrder(t *testing.T) {
	sb := New()

	o := &order.Order{
		ID:        "stop-1",
		Side:      constants.OrderSideBuy,
		Type:      constants.StopMarketOrder,
		StopPrice: 1050,
		Status:    constants.OrderStatusPending,
	}

	sb.Add(o)

	err := sb.CancelOrder(o.ID)

	require.NoError(t, err)

	require.Equal(
		t,
		constants.OrderStatusCancelled,
		o.Status,
	)

	_, exists := sb.orderIndex[o.ID]
	require.False(t, exists)

	require.Empty(t, sb.buyStops)
}

func TestCancelUnknownStopOrder(t *testing.T) {
	sb := New()

	err := sb.CancelOrder("unknown")

	require.ErrorIs(
		t,
		err,
		apperr.ErrStopOrderNotFound,
	)
}

func TestCancelRemovesOnlyRequestedOrder(t *testing.T) {
	sb := New()

	o1 := &order.Order{
		ID:        "stop-1",
		Side:      constants.OrderSideBuy,
		Type:      constants.StopMarketOrder,
		StopPrice: 1050,
	}

	o2 := &order.Order{
		ID:        "stop-2",
		Side:      constants.OrderSideBuy,
		Type:      constants.StopMarketOrder,
		StopPrice: 1050,
	}

	sb.Add(o1)
	sb.Add(o2)

	err := sb.CancelOrder(o1.ID)

	require.NoError(t, err)

	require.Len(t, sb.buyStops[1050], 1)
	require.Equal(t, o2.ID, sb.buyStops[1050][0].ID)

	_, exists := sb.orderIndex[o2.ID]
	require.True(t, exists)
}
