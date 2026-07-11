# Velocity Engine Concurrency Model

## Overview

Modern exchanges process enormous amounts of orders while maintaining one critical requirement:

> Deterministic execution.

The same set of orders arriving in the same sequence must always produce the same result.

Velocity achieves this using a **single-writer event-driven architecture** where every trading symbol owns its own matching thread.

---

# Design Goals

The concurrency model is designed to provide:

- Deterministic execution
- Zero race conditions in matching logic
- Low latency
- High throughput
- Horizontal scalability
- Simple reasoning about state

---

# Fundamental Principle

## One Symbol = One Matching Worker

Each trading symbol owns exactly one matching goroutine.

Example:

```text
BTCUSDT -> Matching Worker #1
ETHUSDT -> Matching Worker #2
AAPL    -> Matching Worker #3
TSLA    -> Matching Worker #4
```

Each worker processes orders independently.

---

# Why This Approach?

Price-time priority requires strict ordering.

Example:

```text
09:00:00.001 BUY 100 @ 1000
09:00:00.002 BUY 100 @ 1000
09:00:00.003 BUY 100 @ 1000
```

Execution order must be:

```text
Order 1
Order 2
Order 3
```

Parallel matching could violate this guarantee.

---

# Why Not Multiple Workers Per Symbol?

Suppose two goroutines process the same order book.

```text
Worker 1 -> BUY Order
Worker 2 -> SELL Order
```

Possible issues:

- race conditions
- inconsistent state
- double execution
- lost orders
- broken FIFO ordering

---

# Industry Practice

Most real exchanges use:

```text
Single writer per symbol
```

Examples include:

- NASDAQ
- CME
- Binance
- Coinbase
- NYSE

---

# Current Architecture

```text
API Layer
    │
    ▼
Registry
    │
    ▼
Engine
    │
    ▼
Buffered Order Queue
    │
    ▼
Matching Worker
    │
    ▼
Trade Queue
```

---

# Order Flow

## Step 1

API receives order.

```text
BUY BTCUSDT
100 @ 1000
```

---

## Step 2

Registry finds engine.

```go
engine := registry.Get("BTCUSDT")
```

---

## Step 3

Order enters order queue.

```go
engine.orderQueue <- order
```

---

## Step 4

Matching worker receives order.

```go
for order := range orderQueue {
    matcher.Match(order)
}
```

---

## Step 5

Trades are generated.

```go
tradeQueue <- trade
```

---

# Engine Worker

Current implementation:

```go
func (e *Engine) start() {
    go func() {
        for order := range e.orderQueue {

            trades, err := e.matcher.Match(order)
            if err != nil {
                continue
            }

            for _, trade := range trades {
                e.tradeQueue <- trade
            }
        }
    }()
}
```

---

# Why Channels?

Channels provide:

- synchronization
- backpressure
- lock avoidance
- safe communication

---

# Order Queue

Structure:

```go
orderQueue chan *order.Order
```

Capacity:

```text
100000
```

---

## Purpose

The order queue separates:

```text
API threads
```

from

```text
Matching thread
```

This prevents the API layer from blocking matching operations.

---

## Example

```text
10 API requests arrive simultaneously

API Threads
     │
     ▼
Order Queue
     │
     ▼
Single Matching Worker
```

Orders are processed in arrival order.

---

# Trade Queue

Structure:

```go
tradeQueue chan *trade.Trade
```

Capacity:

```text
100000
```

---

## Purpose

Trade generation should never block matching.

Instead:

```text
Matcher
   │
   ▼
Trade Queue
   │
   ├── Persistence
   ├── WebSocket Broadcast
   ├── Analytics
   └── Risk Engine
```

---

# Locking Strategy

## Registry

Uses:

```go
sync.RWMutex
```

Reason:

- Many reads
- Few writes

---

## Order Book

Current implementation uses:

```go
sync.RWMutex
```

Example:

```go
mu sync.RWMutex
```

---

## Read Operations

```go
BestBid()
BestAsk()
```

use:

```go
RLock()
```

---

## Write Operations

```go
AddOrder()
RemovePriceLevel()
```

use:

```go
Lock()
```

---

# Future Lock Removal

Current locks exist because:

- tests access the order book directly
- external readers may inspect the book

However matching itself is already single-threaded.

Future versions may move to:

```text
Single Writer Principle
```

where:

- only matching worker modifies order book
- readers receive snapshots
- no locks required

---

# Single Writer Principle

Only one goroutine owns mutable state.

Example:

```text
Matching Worker
      │
      ▼
Order Book
```

No other goroutine can modify the book.

Advantages:

- zero lock contention
- no races
- predictable latency

---

# Horizontal Scaling

Because symbols are isolated:

```text
BTCUSDT -> CPU Core 1
ETHUSDT -> CPU Core 2
AAPL    -> CPU Core 3
TSLA    -> CPU Core 4
```

Scaling becomes linear with CPU count.

---

# Example

Machine:

```text
32 CPU cores
```

Potential:

```text
32 symbols matched simultaneously
```

without contention.

---

# Throughput Model

Current throughput:

```text
1 Symbol
1 Matching Worker
```

Future:

```text
100 Symbols
100 Matching Workers
```

Total throughput:

```text
Per Symbol Throughput
×
Number of Symbols
```

---

# Backpressure

If incoming orders exceed processing speed:

```text
API
 ↓
Queue fills
 ↓
Backpressure begins
```

Possible future strategies:

- reject new orders
- dynamic queue resizing
- overload protection
- load shedding

---

# Failure Isolation

Because engines are isolated:

```text
BTCUSDT failure
```

does not affect:

```text
ETHUSDT
AAPL
TSLA
```

This significantly improves reliability.

---

# Future Concurrency Improvements

## Worker Pools

For:

- persistence
- analytics
- notifications

---

## Lock-Free Queues

Replace channels with:

- ring buffers
- disruptor pattern

---

## CPU Pinning

Assign engine workers to dedicated cores.

Example:

```text
BTCUSDT -> Core 1
ETHUSDT -> Core 2
```

---

## NUMA Awareness

Large servers often have multiple memory regions.

Future versions may optimize:

- memory locality
- cache usage
- inter-core communication

---

## Kernel Bypass Networking

Possible future technologies:

- DPDK
- Solarflare OpenOnload
- RDMA

---

# Why Determinism Matters

Given:

```text
BUY 100 @ 1000
BUY 100 @ 1000
SELL 100 @ 1000
```

Every execution must produce:

```text
Trade #1 with Buy Order #1
```

Never:

```text
Trade #1 with Buy Order #2
```

Determinism is mandatory for exchanges.

---

# Current Concurrency Summary

| Component | Concurrency Model |
|-----------|------------------|
| Registry | RWMutex |
| Order Book | RWMutex |
| Matching | Single Goroutine |
| Order Queue | Buffered Channel |
| Trade Queue | Buffered Channel |
| Symbol Isolation | Separate Engine |

---

# Final Architecture

```text
API
 │
 ▼
Registry
 │
 ├──── BTCUSDT Engine
 │         │
 │         ▼
 │    Order Queue
 │         │
 │         ▼
 │    Matching Worker
 │         │
 │         ▼
 │     Trade Queue
 │
 ├──── ETHUSDT Engine
 │
 ├──── AAPL Engine
 │
 └──── TSLA Engine
```

---

# Conclusion

Velocity follows the same fundamental concurrency principles used by modern electronic exchanges:

- Single writer matching
- Event-driven processing
- Symbol isolation
- Queue-based communication
- Horizontal scalability

This model enables deterministic matching while providing a clear path toward handling hundreds of thousands of requests per second.