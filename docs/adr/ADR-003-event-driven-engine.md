# ADR-003: Event Driven Matching Engine Architecture

## Status

Accepted

---

# Context

Traditional applications often process requests synchronously.

Example:

```text
API Request
    ↓
Matching
    ↓
Database
    ↓
Response
```

This approach works for low throughput systems but becomes a major bottleneck for exchange-grade workloads.

Velocity aims to eventually support:

- 100,000+ requests per second
- Low latency order processing
- High throughput trade generation
- Horizontal scalability

To achieve these goals, matching must be decoupled from external systems.

---

# Problem Statement

Consider the following synchronous flow:

```text
Receive Order
    ↓
Match Order
    ↓
Store Trade
    ↓
Publish WebSocket Event
    ↓
Update Analytics
    ↓
Return Response
```

Problems:

- Matching waits for database writes.
- Matching waits for websocket delivery.
- Matching waits for analytics processing.
- Slow downstream systems increase latency.

This directly impacts exchange performance.

---

# Alternatives Considered

---

## Option 1: Fully Synchronous Processing

Example:

```text
API
 ↓
Matcher
 ↓
Database
 ↓
WebSocket
 ↓
Analytics
```

### Advantages

- Simple implementation.
- Easy to understand.

### Disadvantages

- Very high latency.
- Slow components block matching.
- Poor scalability.
- Difficult to scale independently.

---

## Option 2: Worker Pools

Example:

```text
API
 ↓
Worker Pool
 ↓
Matcher
```

### Advantages

- Better parallelism.
- Better CPU utilization.

### Disadvantages

- Breaks deterministic ordering.
- Introduces race conditions.
- Complicates matching logic.

---

## Option 3: Event Driven Architecture

Example:

```text
API
 ↓
Order Queue
 ↓
Matcher
 ↓
Trade Queue
 ↓
Consumers
```

### Advantages

- Decoupled components.
- Low latency matching.
- Natural backpressure.
- High scalability.
- Fault isolation.

### Disadvantages

- Slightly more complex architecture.
- Requires queue management.

---

# Decision

Velocity adopts:

```text
Event Driven Architecture
```

Orders and trades move through the system as events.

---

# Core Principle

The matcher should perform only one task:

```text
Match orders.
```

Everything else becomes a downstream concern.

Examples:

- Persistence
- WebSocket broadcasts
- Analytics
- Risk checks
- Audit logs

---

# Architecture

```text
Client
  ↓
API
  ↓
Registry
  ↓
Engine
  ↓
Order Queue
  ↓
Matcher
  ↓
Trade Queue
  ↓
Consumers
```

---

# Order Queue

Current implementation:

```go
orderQueue chan *order.Order
```

Example:

```go
e.orderQueue <- order
```

---

## Purpose

Decouple:

```text
Order Submission
```

from

```text
Order Matching
```

---

## Benefits

### Burst Absorption

Incoming traffic:

```text
10000 orders arrive simultaneously
```

The queue absorbs the burst.

---

### Non-Blocking API

The API returns quickly:

```text
Order accepted.
```

instead of waiting for matching.

---

### Backpressure

When the queue fills:

```text
API can reject new orders.
```

instead of crashing.

---

# Matching Worker

Current implementation:

```go
for order := range e.orderQueue {

    trades, err := e.matcher.Match(order)

    if err != nil {
        continue
    }

    for _, trade := range trades {
        e.tradeQueue <- trade
    }
}
```

---

# Trade Queue

Current implementation:

```go
tradeQueue chan *trade.Trade
```

---

## Purpose

Decouple:

```text
Trade Generation
```

from

```text
Trade Processing
```

---

## Example

Instead of:

```text
Matcher
 ↓
Database Write
```

Velocity performs:

```text
Matcher
 ↓
Trade Queue
 ↓
Database Worker
```

---

# Trade Consumers

Future consumers include:

---

## Persistence Service

Responsible for:

- trade history
- audit logs
- recovery

---

## WebSocket Service

Responsible for:

- live trade feeds
- live ticker updates
- market depth updates

---

## Analytics Service

Responsible for:

- volume calculations
- VWAP
- statistics
- indicators

---

## Risk Engine

Responsible for:

- exposure checks
- margin checks
- limits

---

## Notification Service

Responsible for:

- user notifications
- fills
- order status changes

---

# Event Flow Example

Incoming order:

```text
BUY 100 BTC @ 1000
```

---

Step 1:

```text
API receives request
```

---

Step 2:

```text
Order enters queue
```

```text
orderQueue
```

---

Step 3:

```text
Matcher processes order
```

---

Step 4:

```text
Trade generated
```

```text
BUY 100
SELL 100
```

---

Step 5:

```text
Trade enters tradeQueue
```

---

Step 6:

Consumers process trade independently.

---

# Advantages

## Low Matching Latency

Matching only performs:

```text
Order Book Updates
```

and

```text
Trade Generation
```

Nothing else blocks execution.

---

## Horizontal Scaling

Consumers scale independently.

Example:

```text
1 Matcher
5 WebSocket Workers
10 Database Workers
3 Analytics Workers
```

---

## Failure Isolation

If:

```text
Analytics Service crashes
```

matching continues.

If:

```text
WebSocket Service crashes
```

matching continues.

---

## Natural Backpressure

Queues provide buffering.

---

## Decoupled Architecture

Each subsystem evolves independently.

---

# Queue Capacity

Current implementation:

```go
make(chan *order.Order, 100000)
make(chan *trade.Trade, 100000)
```

---

## Why Buffered Channels?

Unbuffered channels would block immediately.

Buffered channels allow temporary bursts.

Example:

```text
10000 orders arrive
```

The queue absorbs the spike.

---

# Future Improvements

---

## Ring Buffers

Possible replacement:

```text
LMAX Disruptor Pattern
```

Benefits:

- lower allocations
- lower latency
- better cache locality

---

## Kafka Integration

Trade queue may evolve into:

```text
Kafka
```

or

```text
NATS
```

for distributed systems.

---

## Event Sourcing

Future architecture may store:

```text
Order Events
Trade Events
Cancel Events
```

instead of snapshots.

---

## Replay Support

The engine could replay:

```text
Historical event streams
```

to rebuild state.

---

# Industry Examples

Most modern exchanges use event-driven systems.

Examples:

- NASDAQ
- CME
- Binance
- Coinbase
- Kraken

The terminology varies:

- Event Bus
- Message Queue
- Command Stream
- Event Stream

The principle remains identical.

---

# Consequences

Velocity accepts:

- additional architectural complexity

in exchange for:

- scalability
- throughput
- isolation
- low latency

---

# Final Decision

Velocity adopts:

```text
Event Driven Architecture
```

using:

- Order Queues
- Trade Queues
- Independent Consumers

because it provides:

- low latency matching
- high throughput
- horizontal scalability
- fault isolation
- exchange-grade architecture

This decision forms one of the fundamental pillars of the Velocity matching engine.