# Matching Engine Architecture

## Purpose

The Matching Engine is the core component of Velocity. It is responsible for receiving, validating, sequencing, and executing trading commands while maintaining a deterministic order book for a single trading symbol.

Every trade executed on the exchange passes through a matching engine.

The engine is designed to maximize correctness, determinism, and performance while keeping the matching path lock-free.

---

# Responsibilities

The Matching Engine is responsible for:

* Receiving trading commands
* Maintaining the in-memory order book
* Matching buy and sell orders
* Enforcing Price-Time Priority
* Managing stop orders
* Generating trades
* Publishing trade events
* Recording commands in the Write-Ahead Log (WAL)
* Creating snapshots
* Maintaining deterministic execution order

The engine **does not** perform:

* Authentication
* Wallet management
* Risk validation
* Settlement
* Database writes
* HTTP handling

Those responsibilities belong to other subsystems.

---

# One Engine Per Symbol

Velocity creates one independent matching engine for each trading symbol.

Example:

```text
BTCUSDT  → Engine
ETHUSDT  → Engine
SOLUSDT  → Engine
```

Each engine owns all state related to its symbol.

This provides:

* Isolation
* Parallelism
* Deterministic execution
* Independent recovery

No engine shares mutable state with another.

---

# High-Level Architecture

```text
                    Engine
                       │
     ┌─────────────────┼─────────────────┐
     │                 │                 │
 OrderBook         StopBook         Sequence
     │                                   │
     ▼                                   ▼
 Matcher                      Command Processing
     │
     ▼
 Trade Queue
     │
     ▼
 Event Publisher
     │
     ├── Market Data
     ├── Statistics
     ├── Candles
     ├── Settlement
     └── Persistence

            WAL
             │
         Snapshot
```

---

# Core Components

## OrderBook

The OrderBook stores all active limit orders for the symbol.

Responsibilities:

* Insert orders
* Remove orders
* Modify orders
* Maintain bid/ask price levels
* Preserve FIFO ordering within each price level
* Provide best executable prices

See: `orderbook.md`

---

## Matcher

The Matcher performs trade execution.

Responsibilities:

* Match incoming orders
* Calculate fills
* Generate trades
* Update remaining quantities
* Remove completed orders
* Handle partial fills
* Handle multiple fills

The Matcher enforces Price-Time Priority at all times.

---

## StopBook

The StopBook manages stop orders that are waiting for a trigger price.

Supported order types:

* Stop Market
* Stop Limit

Once triggered, stop orders are submitted back into the normal matching workflow.

---

## Sequence Generator

Every command processed by the engine receives a monotonically increasing sequence number.

This guarantees:

* Deterministic replay
* Stable ordering
* Recovery consistency

Sequence numbers are never reused.

---

## Write-Ahead Log (WAL)

Before a command is executed, it is written to the WAL.

This ensures commands can be replayed after an unexpected shutdown.

Execution order:

```text
Receive Command
        │
        ▼
Write WAL
        │
        ▼
Execute Command
```

---

## Snapshot Writer

Snapshots periodically serialize the engine state.

Snapshots reduce recovery time by avoiding replaying the entire command history.

Recovery uses:

1. Snapshot
2. WAL replay (if necessary)

---

## Event Publisher

After successful execution, the engine publishes domain events.

Current events include:

* TradeExecuted

Subscribers include:

* Market Data
* Statistics
* Candles
* Settlement
* Persistence
* User Streams

The engine is unaware of its subscribers.

---

# Command Processing

The engine processes commands sequentially.

Supported commands:

* SubmitOrder
* CancelOrder
* ModifyOrder

Every command passes through the same execution pipeline.

```text
Receive Command
        │
        ▼
Assign Sequence
        │
        ▼
Write WAL
        │
        ▼
Execute
        │
        ▼
Generate Events
        │
        ▼
Queue Persistence
```

This guarantees deterministic execution.

---

# Engine Loop

Each engine owns exactly one goroutine.

Pseudo-code:

```go
for {
    select {
    case command := <-commandQueue:
        process(command)
    }
}
```

Because only one goroutine mutates engine state:

* No mutexes are required
* No race conditions exist inside the matching engine
* Execution order is deterministic

---

# Trade Lifecycle

A successful trade follows this flow:

```text
Order Received
        │
        ▼
Validate Command
        │
        ▼
Write WAL
        │
        ▼
Match Order
        │
        ▼
Generate Trade
        │
        ▼
Publish Event
        │
        ▼
Queue Persistence
        │
        ▼
Settlement
        │
        ▼
Market Data Broadcast
```

---

# Engine State

Each engine maintains its own in-memory state.

Typical state includes:

* Order Book
* Stop Book
* Sequence Number
* Last Trade Price
* Trade Queue
* WAL Writer
* Snapshot Writer
* Event Publisher

This state is isolated from every other engine.

---

# Design Principles

The Matching Engine follows several core principles:

### Deterministic Execution

Commands are executed in a fixed order.

The same input always produces the same output.

---

### Single Writer Principle

Only one goroutine mutates engine state.

This eliminates locking within the matching path.

---

### Price-Time Priority

Orders are matched by:

1. Best price
2. Earliest submission time

FIFO ordering is preserved within each price level.

---

### In-Memory Hot Path

The matching engine never performs synchronous database operations.

Persistence is asynchronous.

---

### Event-Driven Architecture

The engine publishes events instead of directly invoking downstream services.

This keeps the matching path lightweight and loosely coupled.

---

# Failure Recovery

The engine supports deterministic recovery using:

* Snapshots
* Write-Ahead Log
* Database recovery

Recovery sequence:

```text
Load Snapshot
        │
        ▼
Restore Engine
        │
        ▼
Replay WAL
        │
        ▼
Recover Remaining State
        │
        ▼
Resume Matching
```

---

# Current Capabilities

Supported order types:

* Market
* Limit
* IOC
* FOK
* Post Only
* Stop Market
* Stop Limit

Supported operations:

* Submit
* Cancel
* Modify
* Partial Fill
* Full Fill
* Multi-order Matching

---

# Performance Characteristics

Current architecture provides:

* One engine per symbol
* Lock-free matching path
* Single-threaded deterministic execution
* Asynchronous persistence
* Event-driven notifications

Future optimizations include:

* Ordered price tree (Red-Black Tree or similar)
* Allocation reduction
* Benchmark-driven optimization
* Engine sharding
* Multi-node deployment

---

# Related Documentation

* `02-system-architecture.md`
* `orderbook.md`
* `concurrency-model.md`
* `data-structures.md`
* `ADR-001-price-time-priority.md`
* `ADR-002-single-threaded-matching.md`
* `ADR-003-event-driven-engine.md`
* `ADR-004-engine-per-symbol.md`
* `ADR-005-fifo-price-levels.md`
