# ADR-002: Single Threaded Matching Per Symbol

## Status

Accepted

---

# Context

A matching engine must guarantee:

- Deterministic execution
- Price-time priority
- Consistent order book state
- Correct trade generation

The most important requirement is:

> The same sequence of incoming orders must always produce the same result.

This requirement becomes difficult when multiple threads attempt to modify the same order book simultaneously.

---

# Problem Statement

Consider the following orders arriving almost simultaneously:

```text
09:00:00.001 BUY 100 @ 1000
09:00:00.002 BUY 100 @ 1000
09:00:00.003 SELL 100 @ 1000
```

Expected result:

```text
SELL matches BUY #1
BUY #2 remains in the book
```

If multiple workers process these orders concurrently:

```text
Worker A -> BUY #1
Worker B -> BUY #2
Worker C -> SELL
```

The result becomes unpredictable.

Possible outcomes:

```text
SELL matches BUY #1
```

or

```text
SELL matches BUY #2
```

This violates price-time priority.

---

# Alternatives Considered

## Option 1: Multiple Matching Threads Per Symbol

Example:

```text
BTCUSDT
├── Worker 1
├── Worker 2
├── Worker 3
└── Worker 4
```

### Advantages

- Higher theoretical throughput.
- Better CPU utilization.

### Disadvantages

- Race conditions.
- Complex locking.
- Lock contention.
- Broken FIFO ordering.
- Difficult debugging.
- Non-deterministic behavior.

---

## Option 2: Global Lock Around Matching

Example:

```go
mutex.Lock()
matcher.Match(order)
mutex.Unlock()
```

### Advantages

- Preserves correctness.

### Disadvantages

- Heavy lock contention.
- Poor scalability.
- Increased latency.
- Wasted CPU resources.

---

## Option 3: Single Matching Thread Per Symbol

Example:

```text
BTCUSDT -> Worker #1
ETHUSDT -> Worker #2
AAPL    -> Worker #3
TSLA    -> Worker #4
```

### Advantages

- Deterministic execution.
- No matching locks.
- Simpler implementation.
- Better cache locality.
- Easy horizontal scaling.

### Disadvantages

- Single symbol throughput limited to one CPU core.

---

# Decision

Velocity will use:

```text
One Matching Goroutine Per Symbol
```

Each engine owns:

- One order queue
- One trade queue
- One matcher
- One order book
- One matching goroutine

---

# Implementation

Current architecture:

```text
Registry
   │
   ├── BTCUSDT Engine
   │      │
   │      ▼
   │   Order Queue
   │      │
   │      ▼
   │   Matching Worker
   │      │
   │      ▼
   │   Trade Queue
   │
   ├── ETHUSDT Engine
   │
   ├── AAPL Engine
   │
   └── RELIANCE Engine
```

---

## Matching Worker

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

Only this goroutine modifies:

- OrderBook
- Orders
- Trades

---

# Ownership Model

The matching worker owns all mutable state.

```text
Matching Worker
      │
      ▼
OrderBook
```

No other goroutine modifies the book.

This approach is known as:

```text
Single Writer Principle
```

---

# Why This Works

Because:

```text
Orders are processed sequentially.
```

Example:

```text
BUY #1
BUY #2
SELL #1
```

Execution order becomes:

```text
BUY #1 enters book
BUY #2 enters book
SELL #1 matches BUY #1
```

The result is always identical.

---

# Benefits

## Deterministic Execution

The same input always produces the same output.

Example:

```text
Replay identical orders
```

Produces:

```text
Identical trades
```

---

## No Matching Locks

Because only one goroutine modifies the order book:

```text
No mutex required inside matching path.
```

This reduces latency.

---

## Easier Debugging

Issues become easier to reproduce.

Example:

```text
Replay event log
```

Produces identical behavior.

---

## Better Cache Locality

One CPU core repeatedly accesses:

- order book
- matcher
- price levels

This improves cache performance.

---

## Simpler Code

No:

- deadlocks
- race conditions
- lock ordering issues
- priority inversion

---

# Drawbacks

## Single Symbol Throughput Limit

One symbol can only use:

```text
One CPU Core
```

Example:

```text
BTCUSDT maximum throughput
=
One worker capacity
```

---

## Extremely Hot Symbols

Very active symbols may eventually exceed the capacity of a single worker.

Potential solutions:

- Symbol partitioning
- Instrument sharding
- Distributed matching

These are future concerns.

---

# Horizontal Scaling

This architecture scales naturally.

Example:

```text
32 CPU Cores

BTCUSDT -> Core 1
ETHUSDT -> Core 2
AAPL    -> Core 3
TSLA    -> Core 4
...
```

Throughput scales approximately linearly with:

```text
Number of Symbols
```

---

# Industry Practices

Most modern exchanges use this model.

Examples:

- NASDAQ
- NYSE
- CME
- Binance
- Coinbase
- Kraken

The terminology varies:

- Single Writer
- Matching Thread
- Instrument Thread
- Symbol Thread

The underlying concept remains identical.

---

# Example Flow

Incoming orders:

```text
BUY 100 @ 1000
BUY 50 @ 1000
SELL 120 @ 1000
```

Processing:

```text
Order Queue
   │
   ▼
Worker
```

Result:

```text
Trade 100 with BUY #1
Trade 20 with BUY #2
BUY #2 remaining = 30
```

This result is guaranteed every time.

---

# Future Improvements

Possible future optimizations:

## CPU Pinning

```text
BTCUSDT -> CPU Core 1
ETHUSDT -> CPU Core 2
```

---

## NUMA Awareness

Improve memory locality on large servers.

---

## Lock-Free Queues

Replace channels with:

- Ring Buffers
- Disruptor Pattern

---

## Distributed Matching

Very active symbols may eventually require:

```text
Symbol Sharding
```

Example:

```text
BTCUSDT-A
BTCUSDT-B
BTCUSDT-C
```

This is outside the scope of Velocity v1.

---

# Consequences

Velocity chooses:

```text
Correctness over theoretical parallelism.
```

The project prioritizes:

- Determinism
- Fairness
- Simplicity
- Predictability

over:

- Maximum per-symbol throughput

---

# Final Decision

Velocity adopts:

```text
Single Threaded Matching Per Symbol
```

because it provides:

- Deterministic execution
- Correct price-time priority
- No race conditions
- Simple implementation
- Excellent horizontal scalability

This decision is one of the fundamental architectural principles of the Velocity Matching Engine.