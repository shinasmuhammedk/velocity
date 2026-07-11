# Velocity Order Book Architecture

## Overview

The Order Book is the core data structure of the matching engine.

Its responsibility is to maintain all active buy and sell orders for a single trading symbol while preserving:

- Price Priority
- Time Priority
- Deterministic Matching
- Low Latency Access

Each symbol owns exactly one independent order book.

Examples:

- BTCUSDT
- ETHUSDT
- AAPL
- TSLA

Each symbol's order book is completely isolated.

---

# What Is An Order Book?

An order book is a collection of:

## Buy Orders (Bids)

Traders willing to buy.

Example:

| Price | Quantity |
|------|---------|
| 1010 | 50 |
| 1005 | 120 |
| 1000 | 300 |

---

## Sell Orders (Asks)

Traders willing to sell.

Example:

| Price | Quantity |
|------|---------|
| 1015 | 100 |
| 1020 | 75 |
| 1025 | 200 |

---

# Matching Rule

Matching occurs when:

```text
Best Bid >= Best Ask
```

Example:

```text
BUY 100 @ 1010
SELL 100 @ 1000
```

Since:

```text
1010 >= 1000
```

The trade executes immediately.

---

# Order Book Structure

Current implementation:

```go
type OrderBook struct {
    Symbol string

    Bids map[int64]*PriceLevel
    Asks map[int64]*PriceLevel

    mu sync.RWMutex
}
```

---

# Symbol

Each order book belongs to one symbol.

Example:

```go
book.Symbol
```

Output:

```text
BTCUSDT
```

---

# Bid Side

Stores all BUY orders.

Structure:

```go
map[int64]*PriceLevel
```

Example:

```text
1010 -> PriceLevel
1005 -> PriceLevel
1000 -> PriceLevel
995  -> PriceLevel
```

Key:

```text
Price
```

Value:

```text
PriceLevel
```

---

# Ask Side

Stores all SELL orders.

Structure:

```go
map[int64]*PriceLevel
```

Example:

```text
1000 -> PriceLevel
1005 -> PriceLevel
1010 -> PriceLevel
1020 -> PriceLevel
```

---

# Why Use Maps?

Hash maps provide:

```text
O(1)
```

lookup performance.

Example:

```go
level := bids[1000]
```

This is critical for low latency matching.

---

# Price Levels

Each price contains a FIFO queue.

Example:

```text
Price: 1000

Order 1
Order 2
Order 3
```

Orders are matched in arrival order.

---

# Price-Time Priority

Velocity follows standard exchange rules:

## Rule 1

Better price wins.

Example:

```text
BUY 1010 executes before BUY 1000
```

---

## Rule 2

If prices are equal:

Earlier order wins.

Example:

```text
09:00:00 BUY 100 @ 1000
09:00:01 BUY 100 @ 1000
```

Execution order:

```text
Order 1
Order 2
```

---

# Adding Orders

Example:

```go
book.AddOrder(order)
```

Process:

1. Determine side
2. Find price level
3. Create level if needed
4. Append order to FIFO queue

---

## Example

Incoming order:

```text
BUY 100 @ 1000
```

Before:

```text
1000 -> [Order A]
```

After:

```text
1000 -> [Order A, Order B]
```

---

# Best Bid

Best Bid represents:

```text
Highest Buy Price
```

Example:

| Price |
|------|
| 1010 |
| 1005 |
| 1000 |

Best Bid:

```text
1010
```

---

## Current Implementation

```go
func (b *OrderBook) BestBid() *PriceLevel
```

Current complexity:

```text
O(N)
```

---

# Best Ask

Best Ask represents:

```text
Lowest Sell Price
```

Example:

| Price |
|------|
| 1000 |
| 1005 |
| 1010 |

Best Ask:

```text
1000
```

---

## Current Implementation

```go
func (b *OrderBook) BestAsk() *PriceLevel
```

Current complexity:

```text
O(N)
```

---

# Removing Empty Levels

When the last order at a price is matched:

```text
1000 -> []
```

The price level is removed.

Example:

```go
book.RemoveBidLevel(1000)
```

or

```go
book.RemoveAskLevel(1000)
```

---

# Current Time Complexity

| Operation | Complexity |
|-----------|-----------|
| Add Order | O(1) |
| Find Price Level | O(1) |
| Remove Price Level | O(1) |
| Best Bid | O(N) |
| Best Ask | O(N) |

---

# Current Matching Example

Initial book:

```text
BIDS
1010 -> 100
1000 -> 50

ASKS
1020 -> 200
1030 -> 300
```

Incoming order:

```text
BUY 150 @ 1025
```

Matching result:

```text
BUY matches SELL 200 @ 1020
```

Remaining:

```text
BUY 50 @ 1025
```

Book becomes:

```text
BIDS
1025 -> 50
1010 -> 100
1000 -> 50

ASKS
1020 -> 50
1030 -> 300
```

---

# Thread Safety

Current implementation uses:

```go
sync.RWMutex
```

---

## Read Operations

Use:

```go
RLock()
```

Examples:

- BestBid()
- BestAsk()

---

## Write Operations

Use:

```go
Lock()
```

Examples:

- AddOrder()
- RemoveBidLevel()
- RemoveAskLevel()

---

# Future Optimization

The current implementation scans all prices to find:

- Best Bid
- Best Ask

This becomes expensive with thousands of price levels.

---

## Future Bid Structure

```text
Max Heap
```

Example:

```text
        1010
       /    \
    1005   1000
```

Complexity:

```text
O(1)
```

Best Bid retrieval.

---

## Future Ask Structure

```text
Min Heap
```

Example:

```text
        1000
       /    \
    1005   1010
```

Complexity:

```text
O(1)
```

Best Ask retrieval.

---

# Future Order Lookup

Current engine does not support direct lookup:

```text
OrderID -> Order
```

Future structure:

```go
map[string]*Order
```

Benefits:

- Fast cancellation
- Fast modification
- Fast order queries

Complexity:

```text
O(1)
```

---

# Future Snapshot Support

Future versions may support:

```text
Order Book Snapshot
```

Example:

```json
{
  "symbol": "BTCUSDT",
  "bids": [...],
  "asks": [...]
}
```

Used for:

- REST APIs
- WebSockets
- Recovery
- Replication

---

# Future Persistence

Future order books may periodically persist:

- snapshots
- event logs
- recovery checkpoints

---

# Exchange Comparison

Most exchanges use a variation of:

```text
Price Level Tree
+
FIFO Queue
```

Examples:

- NASDAQ
- NYSE
- Binance
- Coinbase
- CME

Velocity currently uses:

```text
HashMap
+
Linked List FIFO
```

with a roadmap toward:

```text
Heap
+
Linked List FIFO
```

---

# Architecture Diagram

```text
Order Book
│
├── Bid Side
│   ├── 1010 -> FIFO Queue
│   ├── 1005 -> FIFO Queue
│   └── 1000 -> FIFO Queue
│
└── Ask Side
    ├── 1020 -> FIFO Queue
    ├── 1025 -> FIFO Queue
    └── 1030 -> FIFO Queue
```

---

# Final Summary

The Velocity Order Book provides:

- O(1) price level access
- FIFO execution
- Price-time priority
- Deterministic matching
- Thread safety
- Clear scalability path

This architecture forms the foundation of the entire matching engine and is designed to evolve into an exchange-grade order book capable of supporting very high throughput workloads.