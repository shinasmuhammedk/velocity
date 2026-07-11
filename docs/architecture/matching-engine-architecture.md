# Velocity Matching Engine Architecture

## Overview

Velocity is a high-performance, event-driven matching engine written in Go and designed to simulate the core architecture of modern electronic exchanges such as NASDAQ, Binance, Coinbase, and NSE.

The project aims to achieve low-latency order matching, deterministic execution, and horizontal scalability across multiple trading symbols.

The architecture separates the system into independent services:

- Velocity Engine
- Velocity API

This document describes the design and implementation of the Engine component only.

---

# Vision

To build a production-grade matching engine capable of handling:

- 100,000+ requests per second
- Low-latency order matching
- Multi-symbol trading
- Horizontal scaling
- Event-driven processing

while maintaining deterministic execution and exchange-grade correctness.

---

# Goals

## Functional Goals

- Limit Orders
- Market Orders
- Full Fills
- Partial Fills
- Price-Time Priority
- Multi Symbol Support
- Trade Generation
- Order Book Management
- Event Driven Architecture

## Non Functional Goals

- Low latency
- High throughput
- Deterministic execution
- Horizontal scalability
- Fault tolerance
- Testability
- Clean architecture

---

# Non Goals

The following are intentionally excluded from the engine itself:

- Authentication
- Authorization
- User Management
- REST APIs
- WebSocket APIs
- Database Persistence
- Risk Management
- Portfolio Management
- Margin Calculation
- Settlement

These responsibilities belong to the API service.

---

# High Level Architecture

```
Client
   │
   ▼
Velocity API
   │
   ▼
Registry
   │
   ├──────── BTCUSDT Engine
   │
   ├──────── ETHUSDT Engine
   │
   ├──────── AAPL Engine
   │
   └──────── RELIANCE Engine
```

Each symbol receives its own isolated engine instance.

---

# Component Architecture

```
Order
   │
   ▼
Order Queue
   │
   ▼
Matcher
   │
   ▼
Order Book
   │
   ├── Bid Side
   │
   └── Ask Side
   │
   ▼
Trade Generation
   │
   ▼
Trade Queue
```

---

# Core Components

## Engine

Responsible for:

- Accepting incoming orders
- Managing asynchronous order processing
- Managing trade streams
- Delegating matching to the matcher
- Owning the order book

### Responsibilities

- Own order queue
- Own trade queue
- Start matching worker
- Process incoming orders

---

## Matcher

Responsible for:

- Matching incoming orders
- Applying price-time priority
- Updating order states
- Creating trades

### Supported Matching

- Full fill
- Partial fill
- Multi-level matching
- Multiple trade generation

---

## OrderBook

Responsible for:

- Maintaining bid side
- Maintaining ask side
- Managing price levels
- Finding best bid
- Finding best ask

---

## PriceLevel

Represents a single price in the order book.

Example:

```
1000
 ├── Order A
 ├── Order B
 └── Order C
```

Orders are stored using FIFO ordering.

---

## Registry

Responsible for:

- Managing engine instances
- Creating engines lazily
- Returning existing engines
- Supporting multi-symbol trading

Example:

```
BTCUSDT -> Engine
ETHUSDT -> Engine
AAPL -> Engine
```

---

# Order Lifecycle

## Step 1

API receives order.

Example:

```
BUY BTCUSDT
100 Quantity
1000 Price
```

---

## Step 2

Registry selects engine.

```
registry.Get("BTCUSDT")
```

---

## Step 3

Order is pushed into order queue.

```
orderQueue <- order
```

---

## Step 4

Matching worker consumes order.

```
for order := range orderQueue
```

---

## Step 5

Matcher processes order.

Possible outcomes:

- Full Fill
- Partial Fill
- Resting Order

---

## Step 6

Trades are generated.

```
tradeQueue <- trade
```

---

## Step 7

Trade events become available for:

- WebSocket streaming
- Persistence
- Analytics
- Market data feeds

---

# Trade Lifecycle

```
Order
   │
   ▼
Matcher
   │
   ▼
Trade
   │
   ▼
Trade Queue
   │
   ├── Database
   ├── WebSocket
   ├── Analytics
   └── Risk Engine
```

---

# Matching Rules

Velocity uses strict price-time priority.

## Buy Orders

Buy orders execute against:

- Lowest available ask price.

Example:

```
BUY 100 @ 1000

ASKS

990
995
1000
1005
```

Execution starts from:

```
990
```

---

## Sell Orders

Sell orders execute against:

- Highest available bid price.

Example:

```
SELL 100 @ 1000

BIDS

1010
1005
1000
995
```

Execution starts from:

```
1010
```

---

# FIFO Priority

Orders with identical prices execute in arrival order.

Example:

```
BUY 100 @ 1000 (09:00:01)
BUY 100 @ 1000 (09:00:02)
BUY 100 @ 1000 (09:00:03)
```

Execution order:

```
Order 1
Order 2
Order 3
```

---

# Data Structures

## Registry

```
map[string]*Engine
```

Complexity:

```
O(1)
```

---

## Bid Side

```
map[int64]*PriceLevel
```

Current best price lookup:

```
O(N)
```

Future:

```
Max Heap
O(log N)
```

---

## Ask Side

```
map[int64]*PriceLevel
```

Current best price lookup:

```
O(N)
```

Future:

```
Min Heap
O(log N)
```

---

## Orders inside Price Level

```
container/list
```

Operations:

| Operation | Complexity |
|-----------|-----------|
| Insert | O(1) |
| Remove Front | O(1) |
| Peek Front | O(1) |

---

# Concurrency Model

Velocity uses:

## Single Matching Goroutine Per Symbol

Example:

```
BTCUSDT -> Goroutine #1
ETHUSDT -> Goroutine #2
AAPL -> Goroutine #3
```

This guarantees:

- Deterministic execution
- No race conditions
- No matching locks

---

# Why Matching Is Not Parallel

Modern exchanges typically avoid parallel matching for a single symbol because:

- Ordering guarantees become difficult
- Race conditions appear
- Price-time priority breaks

Velocity follows the same model.

---

# Current Architecture

```
API Thread
    │
    ▼
Buffered Order Queue
    │
    ▼
Single Matching Worker
    │
    ▼
Buffered Trade Queue
```

---

# Queue Sizes

## Order Queue

```
100,000
```

## Trade Queue

```
100,000
```

---

# Performance Targets

## Current Target

```
100,000 requests/sec
```

---

## Future Target

```
1,000,000+ requests/sec
```

---

# Planned Optimizations

- Heap based price discovery
- Order indexing
- Memory pooling
- Object reuse
- Lock free queues
- Snapshot recovery
- Persistence layer
- NUMA awareness
- CPU pinning
- Kernel bypass networking

---

# Current Features

## Implemented

- Limit Orders
- Market Orders
- Full Fill
- Partial Fill
- Price-Time Priority
- Registry
- Event Driven Processing
- Trade Queue
- Unit Tests

---

## Planned

- IOC Orders
- FOK Orders
- Order Cancellation
- Self Trade Prevention
- Snapshot Recovery
- Persistence
- WebSocket Market Data
- Distributed Matching

---

# Testing

Current test coverage includes:

- PriceLevel
- OrderBook
- Matcher
- Engine
- Registry

---

# Future Architecture

```
Client
   │
   ▼
API Gateway
   │
   ▼
gRPC
   │
   ▼
Registry
   │
   ├── BTCUSDT Engine
   ├── ETHUSDT Engine
   ├── AAPL Engine
   └── RELIANCE Engine
```

---

# Repository Structure

```
internal/
│
├── domain/
│   ├── order
│   └── trade
│
├── engine/
│   ├── engine
│   ├── matcher
│   ├── orderbook
│   ├── pricelevel
│   └── registry
│
└── transport/
```

---

# Conclusion

Velocity Engine is designed around the same fundamental principles used by real-world exchanges:

- Price-Time Priority
- Single Writer Matching
- Event Driven Processing
- Symbol Isolation
- Horizontal Scalability

The current implementation forms the foundation for future development of:

- Velocity API
- Market Data Streaming
- Persistence
- Risk Management
- Distributed Trading Infrastructure