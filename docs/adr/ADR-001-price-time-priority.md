# ADR-001: Price-Time Priority Matching

## Status

Accepted

---

# Context

A matching engine requires a deterministic rule for deciding which orders receive execution priority.

When multiple orders exist in the order book, the engine must decide:

- Which price should execute first?
- Which order should execute first if multiple orders exist at the same price?

This decision directly affects:

- fairness
- predictability
- market integrity
- performance characteristics

Selecting an inappropriate matching algorithm can result in:

- unfair execution
- non-deterministic behavior
- opportunities for market manipulation
- reduced trader confidence

---

# Problem Statement

Consider the following order book:

## Buy Orders

| Time | Price | Quantity |
|------|------|----------|
| 09:00:00 | 1000 | 100 |
| 09:00:01 | 1000 | 100 |
| 09:00:02 | 995 | 100 |

Incoming order:

```text
SELL 150 @ 1000
```

The engine must decide:

1. Which price receives priority?
2. Which order receives priority within the same price?

---

# Alternatives Considered

## 1. Price-Time Priority

Rules:

1. Better price executes first.
2. Earlier order executes first if prices are equal.

Example:

```text
BUY 100 @ 1000 (09:00:00)
BUY 100 @ 1000 (09:00:01)
BUY 100 @ 995  (09:00:02)
```

Execution order:

```text
1. BUY 100 @ 1000 (09:00:00)
2. BUY 100 @ 1000 (09:00:01)
3. BUY 100 @ 995
```

---

## 2. Pro-Rata Matching

Orders are matched proportionally according to order size.

Example:

Order Book:

```text
BUY 100 @ 1000
BUY 200 @ 1000
BUY 700 @ 1000
```

Incoming Sell:

```text
SELL 100 @ 1000
```

Allocation:

```text
10
20
70
```

Advantages:

- Encourages liquidity providers.

Disadvantages:

- Rewards larger participants.
- More complex implementation.
- Less fair for retail traders.

Used by:

- Some futures exchanges.

---

## 3. Size Priority

Largest order executes first.

Example:

```text
BUY 100 @ 1000
BUY 500 @ 1000
BUY 300 @ 1000
```

Execution order:

```text
500
300
100
```

Advantages:

- Encourages larger orders.

Disadvantages:

- Unfair to smaller traders.
- Easily exploitable.

Rarely used.

---

## 4. Random Matching

Orders are selected randomly.

Advantages:

- Prevents queue positioning strategies.

Disadvantages:

- Non-deterministic.
- Difficult to audit.
- Not acceptable for financial exchanges.

---

# Decision

Velocity will use:

```text
Price-Time Priority
```

Matching rules:

## Rule 1

Better price receives priority.

For BUY orders:

```text
Highest bid wins.
```

For SELL orders:

```text
Lowest ask wins.
```

---

## Rule 2

If prices are equal:

```text
Earlier order wins.
```

Example:

```text
BUY 100 @ 1000 (09:00:00)
BUY 100 @ 1000 (09:00:01)
BUY 100 @ 1000 (09:00:02)
```

Execution order:

```text
Order 1
Order 2
Order 3
```

---

# Implementation

Velocity implements price-time priority using:

## Price Priority

Price levels:

```go
map[int64]*PriceLevel
```

Current best prices:

```text
Best Bid -> Highest price
Best Ask -> Lowest price
```

---

## Time Priority

Orders inside a price level are stored using:

```go
container/list
```

Example:

```text
Price Level 1000

Order A
Order B
Order C
```

Orders are appended to the back:

```go
PushBack(order)
```

Orders are matched from the front:

```go
Front()
```

This guarantees FIFO ordering.

---

# Consequences

## Advantages

### Fairness

Participants offering better prices receive priority.

Participants arriving earlier at the same price receive priority.

---

### Deterministic Execution

Given identical order sequences:

```text
Input Sequence A
```

always produces:

```text
Output Sequence A
```

This is essential for:

- testing
- auditing
- recovery
- replay systems

---

### Industry Standard

Price-time priority is the dominant matching model used by:

- NASDAQ
- NYSE
- Binance
- Coinbase
- CME
- NSE
- BSE

---

### Simple Implementation

The model maps naturally to:

```text
Price Levels
+
FIFO Queues
```

making the engine easier to reason about.

---

## Disadvantages

### Queue Position Becomes Valuable

Participants compete for earlier placement within a price level.

This creates:

- queue positioning strategies
- latency competition

---

### Incentivizes Low Latency Infrastructure

Faster participants may receive execution priority.

This is a known characteristic of nearly all electronic exchanges.

---

# Data Structure Implications

Price-time priority requires:

## Price Ordering

Current implementation:

```text
Hash Map + Scan
```

Future implementation:

```text
Heap
```

---

## FIFO Ordering

Current implementation:

```text
Linked List
```

Operations:

| Operation | Complexity |
|-----------|-----------|
| Insert | O(1) |
| Front | O(1) |
| Remove Front | O(1) |

---

# Example Execution

Order Book:

```text
BIDS

1005 -> Order A
1000 -> Order B
1000 -> Order C
995  -> Order D
```

Incoming Sell:

```text
SELL 150 @ 995
```

Execution:

```text
1. Order A executes first.
2. Order B executes second.
3. Order C remains.
```

---

# Future Considerations

Velocity may support additional matching models in future versions:

- Pro-Rata
- Hybrid Matching
- Auction Matching
- Opening Auction
- Closing Auction

However:

```text
Price-Time Priority
```

will remain the default matching algorithm.

---

# Final Decision

Velocity adopts:

```text
Price-Time Priority Matching
```

because it provides:

- fairness
- determinism
- simplicity
- industry compatibility
- exchange-grade behavior

This decision forms one of the core architectural foundations of the Velocity Matching Engine.