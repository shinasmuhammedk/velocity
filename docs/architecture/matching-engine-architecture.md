# Matching Engine Architecture

## Purpose

The Velocity Matching Engine is a dedicated microservice responsible for
order matching inside the Velocity trading platform.

It is designed as a low-latency, in-memory, deterministic matching
engine implementing a Price-Time Priority algorithm similar to modern
centralized exchanges.

The engine service is isolated from authentication, user management, and
persistence concerns.

------------------------------------------------------------------------

## Service Boundaries

``` text
velocity-api
    │
    │ gRPC
    ▼
velocity-engine
```

### velocity-api Responsibilities

-   Authentication
-   User management
-   Wallets and balances
-   Position tracking
-   Order persistence
-   Trade persistence
-   Risk checks

### velocity-engine Responsibilities

-   Order matching
-   Order book management
-   Trade generation
-   Price-time priority execution

------------------------------------------------------------------------

## Architecture Overview

``` text
Registry
    ↓
Engine
    ↓
Matcher
    ↓
OrderBook
    ↓
PriceLevel
```

------------------------------------------------------------------------

## Registry

The Registry owns every trading engine instance in memory.

Example:

``` text
BTCUSDT → Engine
ETHUSDT → Engine
AAPL    → Engine
RELIANCE → Engine
```

Orders belonging to one symbol never interact with another symbol.

------------------------------------------------------------------------

## Engine

The Engine represents a single trading symbol.

Responsibilities:

-   Receive orders
-   Delegate matching to Matcher
-   Return generated trades
-   Expose OrderBook access

The engine contains no matching logic.

------------------------------------------------------------------------

## Matcher

The Matcher is responsible for exchange rules.

Responsibilities:

-   Match incoming orders
-   Apply price priority
-   Apply time priority
-   Generate trades
-   Update order states
-   Remove filled orders
-   Keep partially filled orders resting

### Buy Matching

Buy orders consume liquidity from the lowest ask first.

Matching stops when:

``` text
BestAsk > BuyPrice
```

### Sell Matching

Sell orders consume liquidity from the highest bid first.

Matching stops when:

``` text
BestBid < SellPrice
```

------------------------------------------------------------------------

## OrderBook

The OrderBook owns all bid and ask price levels.

``` text
OrderBook
├── Bids
└── Asks
```

Implementation:

``` text
map[int64]*PriceLevel
```

### Bid Side

Highest price wins.

### Ask Side

Lowest price wins.

------------------------------------------------------------------------

## PriceLevel

PriceLevel is a FIFO queue for orders sharing the same price.

Implementation:

``` text
container/list
```

Responsibilities:

-   AddOrder()
-   Front()
-   RemoveFront()
-   IsEmpty()
-   Size()

Orders at the same price are executed in arrival order.

------------------------------------------------------------------------

## Price-Time Priority

Priority order:

1.  Price Priority
2.  Time Priority

Example:

``` text
SELL 50 @1000
SELL 30 @1005
SELL 100 @1010

BUY 120 @1010
```

Results:

``` text
50 @1000
30 @1005
40 @1010
```

------------------------------------------------------------------------

## Domain Models

### Order

Fields:

-   ID
-   UserID
-   Symbol
-   Side
-   Type
-   TimeInForce
-   Status
-   Price
-   Quantity
-   Remaining
-   Filled
-   CreatedAt
-   UpdatedAt

### Trade

Fields:

-   ID
-   BuyOrderID
-   SellOrderID
-   BuyerID
-   SellerID
-   Symbol
-   Price
-   Quantity
-   ExecutedAt

------------------------------------------------------------------------

## Current Implementation Status

### Completed

-   Buy matching
-   Sell matching
-   Full fills
-   Partial fills
-   FIFO execution
-   Multiple price levels
-   Trade generation

### Test Coverage

#### Matcher

-   Full fill
-   Partial fill
-   FIFO
-   Multi-level matching

#### PriceLevel

-   AddOrder
-   RemoveFront
-   FIFO
-   Size
-   IsEmpty

#### OrderBook

-   BestBid
-   BestAsk
-   RemoveBidLevel
-   RemoveAskLevel

#### Engine

-   Full fill flow
-   Partial fill flow
-   Resting order flow

------------------------------------------------------------------------

## Future Expansion

-   Registry implementation
-   gRPC API
-   Persistence layer
-   Market data streaming
-   Redis
-   Kafka/NATS
-   Risk engine
-   Replay engine
-   Metrics and tracing

------------------------------------------------------------------------

## Design Principles

-   Single Responsibility Principle
-   Dependency Injection
-   Constructor-based initialization
-   Thread safety
-   Deterministic behavior
-   Horizontal scalability
-   In-memory execution
