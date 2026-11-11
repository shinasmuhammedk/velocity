# Sequence Diagram: Partial Fill

## Purpose

Traces what happens inside the matching engine when an incoming order
matches against a resting order for less than its full quantity — the
incoming order ends up `PARTIALLY_FILLED` (and rests in the book) or, if
it fully consumes the resting order while some of the incoming quantity
remains, the *resting* order ends up `FILLED` while the incoming order
continues matching against the next price level. This diagram covers the
single-trade case: one incoming order, one resting order, quantities
don't match exactly.

Continues from [`order-placement.md`](order-placement.md), picking up
inside `Engine.SubmitOrder`.

## Participants and source references

| Participant | Source |
|---|---|
| Per-symbol engine | `internal/engine/engine.go` (`SubmitOrder`, `processBatch`, `applyCommand`) |
| WAL writer | `internal/engine/wal/writer.go` (`WriteBatch`) |
| Matcher | `internal/engine/matcher/matcher.go` (`Match`) |
| Order book | `internal/engine/orderbook/order_book.go` |
| Event publisher | `internal/engine/events` + `internal/infrastructure/kafka/event_publisher.go` |
| Trade consumer (settlement) | `internal/persistence/worker/trade_consumer.go` |

## Diagram

```mermaid
sequenceDiagram
    autonumber
    participant Caller as SubmitOrder (caller)
    participant Engine as Engine.commandQueue
    participant WAL as wal.Writer
    participant Matcher as matcher.Matcher
    participant Book as OrderBook
    participant Events as events.Dispatcher
    participant Kafka as Kafka (audit/analytics only)
    participant TC as TradeConsumer (in-process)

    Caller->>Engine: commandQueue <- Submit(order)
    Engine->>Engine: processBatch: assign sequence number

    Engine->>WAL: WriteBatch([SubmitEvent])
    Note right of WAL: One fsync per batch, not per order<br/>(see docs/adr, group-commit note in<br/>docs/performance/benchmark-results.md).
    WAL-->>Engine: durable

    Engine->>Matcher: Match(incoming order)
    Matcher->>Book: find best opposing price level

    Book-->>Matcher: resting order (quantity < incoming.Remaining,<br/>or incoming.Remaining < resting.Remaining)

    alt resting order fully consumed, incoming has quantity left
        Matcher->>Matcher: tradeQty = resting.Remaining
        Matcher->>Book: remove resting order from price level
        Matcher->>Matcher: resting.Status = FILLED
        Matcher->>Matcher: incoming.Remaining -= tradeQty; incoming.Filled += tradeQty
        Matcher->>Matcher: incoming.Status = PARTIALLY_FILLED (Remaining > 0, < Quantity)
        Note over Matcher: Loop continues: incoming.Remaining > 0,<br/>so the matcher tries the next best price level.
    else incoming fully consumed against a larger resting order
        Matcher->>Matcher: tradeQty = incoming.Remaining
        Matcher->>Book: decrement resting order's Remaining in place
        Matcher->>Matcher: resting.Status = PARTIALLY_FILLED
        Matcher->>Matcher: incoming.Remaining = 0; incoming.Status = FILLED
        Note over Matcher: Loop ends: incoming.Remaining == 0.
    end

    Matcher-->>Engine: []trade.Trade{ one trade }

    Engine->>TC: tradeQueue <- trade (direct in-process channel)
    Engine->>Events: publish(TradeExecutedEvent)
    Events->>Kafka: EventPublisher.Handle (audit/analytics topic)

    Note over Engine: If incoming.Remaining > 0 and TimeInForce == GTC,<br/>it now rests in the book at its own price level.<br/>If IOC/FOK, remaining quantity is not booked - see<br/>matcher.go for the exact TIF handling.

    Engine-->>Caller: nil (SubmitOrder returns; command.Result channel)
```

## Notes

- **A "partial fill" is a property of one side of one trade, not of the
  trade as a whole.** In a single trade, the resting order can end
  `FILLED` while the incoming order ends `PARTIALLY_FILLED`, or vice
  versa — see `matcher.go` lines around `resting.Remaining == 0` /
  `incoming.Remaining == 0` for the exact per-side status assignment.
- **Settlement is synchronous and in-process, not Kafka-driven.** The
  trade is pushed directly to `TradeConsumer` via the engine's own
  `tradeQueue` Go channel (wired in `internal/app/bootstrap.go`, inside
  the `api` binary). Kafka only carries a copy of the event for the
  audit/analytics pipeline (`EventAuditSubscriber`, candles, stats),
  consumed by the separate `cmd/worker` process. **This differs from the
  architecture described in the top-level README**, which describes
  settlement as being consumed asynchronously from Kafka by a worker
  process — see the note in `full-fill.md` for the full discrepancy and
  why it matters.
- A partially-filled order that rests in the book keeps its original
  order ID and price-time priority position for its *remaining*
  quantity — it does not go to the back of the queue at its price level.