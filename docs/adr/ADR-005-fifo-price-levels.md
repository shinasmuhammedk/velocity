# ADR-005: In-Memory Order Book Architecture

## Status

Accepted

---

# Context

A matching engine operates on the critical path of an exchange.

For every incoming order the engine must:

1. Locate the best opposite order.
2. Match quantities.
3. Generate trades.
4. Update the order book.

This process occurs for every order entering the system.

Velocity targets future throughput goals of:

- 100,000+ orders/sec
- Sub-millisecond matching latency
- Deterministic execution

The storage architecture chosen for the order book directly determines whether these goals are achievable.

---

# Problem Statement

Should the order book live:

```text
Inside Memory
```

or

```text
Inside Database Storage
```

This decision affects:

- Latency
- Throughput
- Scalability
- Recovery
- Complexity

---

# Alternatives Considered

---

## Option 1: Database Backed Order Book

Architecture:

```text
Incoming Order
    ↓
Database Query
    ↓
Find Best Price
    ↓
Update Database
    ↓
Generate Trade
```

---

### Advantages

- Persistent state.
- Easier recovery.
- Simpler operational model.

---

### Disadvantages

- Extremely high latency.
- Database becomes bottleneck.
- Disk I/O limits throughput.
- Impossible to achieve exchange-grade performance.

---

### Example

Even a fast database query:

```text
0.5 ms
```

limits throughput to approximately:

```text
2000 orders/sec
```

This is insufficient for a modern exchange.

---

## Option 2: In-Memory Order Book

Architecture:

```text
Incoming Order
    ↓
RAM Lookup
    ↓
Matching
    ↓
Trade Generation
```

---

### Advantages

- Extremely low latency.
- Millions of operations per second.
- CPU cache utilization.
- No disk bottleneck.

---

### Disadvantages

- State lost on crash.
- Requires recovery mechanisms.
- Additional persistence layer needed.

---

# Decision

Velocity adopts:

```text
In-Memory Order Books
```

for all active symbols.

Persistence becomes a downstream concern.

---

# Current Architecture

Each engine owns:

```text
Engine
│
├── OrderBook
│
├── Matcher
│
├── Order Queue
│
└── Trade Queue
```

The order book exists entirely inside memory.

---

# Order Book Structure

Current implementation:

```go
type OrderBook struct {
    Symbol string

    Bids map[int64]*pricelevel.PriceLevel
    Asks map[int64]*pricelevel.PriceLevel

    mu sync.RWMutex
}
```

---

# Bid Side

```go
Bids map[int64]*PriceLevel
```

Example:

```text
1000 -> PriceLevel
999  -> PriceLevel
998  -> PriceLevel
```

---

# Ask Side

```go
Asks map[int64]*PriceLevel
```

Example:

```text
1001 -> PriceLevel
1002 -> PriceLevel
1003 -> PriceLevel
```

---

# Price Level Structure

Current implementation:

```go
type PriceLevel struct {
    Price  int64
    Orders *list.List
}
```

---

Example:

```text
Price = 1000

Order1
Order2
Order3
```

Orders maintain FIFO ordering.

---

# Matching Flow

Example:

```text
BUY 100 @ 1000
```

Engine performs:

```text
Best Ask Lookup
```

then:

```text
Quantity Matching
```

then:

```text
Trade Generation
```

All operations occur in memory.

---

# Why Memory Matters

Memory access latency:

```text
50 - 100 nanoseconds
```

SSD access latency:

```text
50,000 - 100,000 nanoseconds
```

Database query latency:

```text
500,000+ nanoseconds
```

Memory is several thousand times faster.

---

# Performance Comparison

## Memory Lookup

```text
~100 ns
```

---

## Redis Lookup

```text
~100,000 ns
```

---

## PostgreSQL Query

```text
~500,000 ns
```

---

## Disk Read

```text
~1,000,000 ns
```

---

The difference is massive at exchange scale.

---

# Example

100,000 orders/sec means:

```text
10 microseconds budget per order.
```

Database calls alone exceed that budget.

Therefore:

```text
Database on matching path is impossible.
```

---

# Persistence Strategy

Velocity separates:

```text
Matching
```

from

```text
Persistence
```

---

Architecture:

```text
Matcher
   ↓
Trade Queue
   ↓
Persistence Worker
   ↓
Database
```

---

The database never blocks matching.

---

# Recovery Strategy

Future versions will use:

```text
Event Replay
```

Example:

```text
Order Accepted Event
Trade Generated Event
Cancel Event
```

After restart:

```text
Replay Events
↓
Rebuild Order Book
```

---

# Snapshot Strategy

Future versions may periodically store:

```text
Order Book Snapshots
```

Example:

```text
Snapshot every 1 second
```

Recovery becomes:

```text
Load Snapshot
Replay Remaining Events
```

---

# Future Improvements

---

## Memory Pools

Reduce allocations using:

```text
sync.Pool
```

---

## Arena Allocation

Possible future optimization:

```text
Object Arenas
```

---

## Lock-Free Structures

Future replacements:

```text
Lock-Free Ring Buffers
```

---

## NUMA Awareness

Improve memory locality on large servers.

---

## Huge Pages

Reduce TLB misses for very large books.

---

# Industry Practice

Every major exchange keeps active books in memory.

Examples:

- NASDAQ
- NYSE
- CME
- Binance
- Coinbase
- Kraken

Databases are used only for:

- Persistence
- Recovery
- Analytics
- Reporting

Never for matching.

---

# Consequences

Velocity accepts:

- memory usage
- recovery complexity
- snapshot requirements

in exchange for:

- ultra-low latency
- high throughput
- exchange-grade performance

---

# Final Decision

Velocity adopts:

```text
In-Memory Order Books
```

because they provide:

- Nanosecond access times
- Extremely high throughput
- Low latency matching
- Better CPU cache utilization

Persistence is intentionally moved outside the matching path.

This decision is fundamental to achieving the long-term goal of:

```text
100,000+ orders per second
```

within the Velocity Matching Engine.