# Velocity Engine Data Structures

## Overview

The choice of data structures determines the throughput, latency, scalability, and correctness of a matching engine.

Velocity is designed to support exchange-grade workloads with a long-term target of:

- 100,000+ requests per second
- Low latency execution
- Deterministic matching
- Horizontal scalability

This document describes the current and future data structures used throughout the engine.

---

# Design Principles

Velocity chooses data structures based on:

1. Time complexity
2. Cache locality
3. Memory efficiency
4. Deterministic behavior
5. Concurrent access patterns
6. Simplicity of implementation

---

# Registry

## Purpose

Stores one matching engine instance per symbol.

Example:

```
BTCUSDT -> Engine
ETHUSDT -> Engine
AAPL -> Engine
```

---

## Current Structure

```go
map[string]*Engine
```

---

## Complexity

| Operation | Complexity |
|----------|-----------|
| Insert | O(1) |
| Lookup | O(1) |
| Delete | O(1) |

---

## Why Map?

Because symbol lookup must be extremely fast.

Example:

```go
engine := registry.Get("BTCUSDT")
```

A hashmap provides constant time access regardless of the number of symbols.

---

# Order Book

The order book consists of two independent sides:

- Bid Side
- Ask Side

---

# Bid Side

Stores all BUY orders.

Current structure:

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

---

## Complexity

| Operation | Complexity |
|----------|-----------|
| Add Price Level | O(1) |
| Lookup Price Level | O(1) |
| Remove Price Level | O(1) |

---

## Best Bid Lookup

Current implementation:

```go
for price := range bids {
    if price > best {
        best = price
    }
}
```

Complexity:

```text
O(N)
```

where:

```
N = Number of bid price levels
```

---

## Future Optimization

Velocity plans to replace linear scanning with:

```text
Max Heap
```

Example:

```text
          1010
         /    \
      1005    1000
      /
    995
```

Complexity:

| Operation | Complexity |
|----------|-----------|
| Insert | O(log N) |
| Remove Best | O(log N) |
| Best Bid | O(1) |

---

# Ask Side

Stores all SELL orders.

Current structure:

```go
map[int64]*PriceLevel
```

Example:

```text
995  -> PriceLevel
1000 -> PriceLevel
1005 -> PriceLevel
1010 -> PriceLevel
```

---

## Complexity

| Operation | Complexity |
|----------|-----------|
| Add Price Level | O(1) |
| Lookup Price Level | O(1) |
| Remove Price Level | O(1) |

---

## Best Ask Lookup

Current implementation:

```go
for price := range asks {
    if price < best {
        best = price
    }
}
```

Complexity:

```text
O(N)
```

---

## Future Optimization

Velocity plans to use:

```text
Min Heap
```

Example:

```text
          995
         /   \
      1000   1005
      /
    1010
```

Complexity:

| Operation | Complexity |
|----------|-----------|
| Insert | O(log N) |
| Remove Best | O(log N) |
| Best Ask | O(1) |

---

# Price Level

## Purpose

Represents all orders at a specific price.

Example:

```text
Price = 1000

Order A
Order B
Order C
Order D
```

---

## Current Structure

```go
type PriceLevel struct {
    Price  int64
    Orders *list.List
}
```

---

# Why Linked List?

Price-Time Priority requires:

- Append new order to end
- Read oldest order
- Remove oldest order

These operations must be extremely fast.

---

## Operations

### Add Order

```go
Orders.PushBack(order)
```

Complexity:

```text
O(1)
```

---

### Get Oldest Order

```go
Orders.Front()
```

Complexity:

```text
O(1)
```

---

### Remove Oldest Order

```go
Orders.Remove(front)
```

Complexity:

```text
O(1)
```

---

## Complexity Table

| Operation | Complexity |
|----------|-----------|
| Insert | O(1) |
| Peek Front | O(1) |
| Remove Front | O(1) |

---

# Why Not Slice?

Using:

```go
[]*Order
```

would make FIFO removal expensive.

Example:

```go
orders = orders[1:]
```

This creates:

- memory fragmentation
- extra allocations
- copying overhead

Complexity:

```text
O(N)
```

---

# Order Queue

## Purpose

Decouples API threads from matching threads.

---

## Structure

```go
chan *order.Order
```

---

## Example

```text
API
 ↓
Order Queue
 ↓
Matcher
```

---

## Benefits

- Backpressure handling
- Async processing
- Lock avoidance
- Burst absorption

---

## Current Capacity

```text
100000
```

---

# Trade Queue

## Purpose

Decouples matching from downstream systems.

---

## Structure

```go
chan *trade.Trade
```

---

## Example

```text
Matcher
   ↓
Trade Queue
   ↓
Persistence
WebSocket
Analytics
Risk Engine
```

---

## Benefits

- Non-blocking matching
- Event-driven architecture
- Easy extensibility

---

## Current Capacity

```text
100000
```

---

# Matcher State

The matcher itself maintains no state.

Structure:

```go
type Matcher struct {
    book *OrderBook
}
```

---

## Benefits

- Stateless design
- Easy testing
- Simple scaling
- Easy replacement

---

# Engine State

The engine owns:

- Order Book
- Matcher
- Order Queue
- Trade Queue

---

## Structure

```go
type Engine struct {
    book       *OrderBook
    matcher    *Matcher
    orderQueue chan *Order
    tradeQueue chan *Trade
}
```

---

# Complexity Summary

| Component | Data Structure | Complexity |
|-----------|---------------|-----------|
| Registry | map | O(1) |
| Bid Lookup | map | O(1) |
| Ask Lookup | map | O(1) |
| Best Bid | linear scan | O(N) |
| Best Ask | linear scan | O(N) |
| FIFO Insert | linked list | O(1) |
| FIFO Remove | linked list | O(1) |
| Order Queue | channel | O(1) |
| Trade Queue | channel | O(1) |

---

# Future Data Structures

Velocity plans to evolve toward:

| Component | Future Structure |
|-----------|-----------------|
| Best Bid | Max Heap |
| Best Ask | Min Heap |
| Order Lookup | Hash Map |
| Snapshots | Immutable Structures |
| Persistence Queue | Ring Buffer |
| Event Bus | Kafka/NATS |
| Object Allocation | sync.Pool |

---

# High Frequency Trading Optimizations

Future versions may include:

- Heap optimized order books
- Lock-free queues
- CPU pinning
- NUMA awareness
- Memory pooling
- Object reuse
- Kernel bypass networking

---

# Final Architecture

```
Registry
    ↓
Engine
    ↓
Order Queue
    ↓
Matcher
    ↓
Order Book
    ├── Bid Side
    ├── Ask Side
    └── Price Levels
    ↓
Trade Queue
```

---

# Conclusion

Velocity intentionally prioritizes:

- Correctness
- Deterministic execution
- Low latency
- High throughput
- Simplicity

The current implementation provides a strong foundation for future evolution into a production-grade matching engine capable of supporting hundreds of thousands of requests per second.