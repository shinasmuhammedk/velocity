package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    OrdersSubmitted = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "velocity_orders_submitted_total",
            Help: "Total number of submitted orders",
        },
    )

    OrdersCancelled = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "velocity_orders_cancelled_total",
            Help: "Total number of cancelled orders",
        },
    )

    OrdersModified = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "velocity_orders_modified_total",
            Help: "Total number of modified orders",
        },
    )

    TradesExecuted = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "velocity_trades_executed_total",
            Help: "Total number of executed trades",
        },
    )
)

func Register() {
    prometheus.MustRegister(
        OrdersSubmitted,
        OrdersCancelled,
        OrdersModified,
        TradesExecuted,
    )
}