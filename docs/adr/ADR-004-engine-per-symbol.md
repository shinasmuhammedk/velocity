# ADR-004: One Matching Engine Per Symbol

## Status

Accepted

---

# Context

Velocity is designed as a multi-symbol matching system.

Examples:

- BTCUSDT
- ETHUSDT
- SOLUSDT
- AAPL
- TSLA
- RELIANCE

Each symbol has:

- Independent liquidity
- Independent order flow
- Independent market state
- Independent matching logic

The system must determine how these symbols are managed internally.

---

# Problem Statement

Consider the following incoming orders:

```text
BUY BTCUSDT 100 @ 100000
SELL ETHUSDT 50 @ 3000
BUY AAPL 10 @ 250
SELL TSLA 5 @ 400
```

Should all symbols share:

```text
One Global Engine
```

or

```text
One Engine Per Symbol
```

This decision directly impacts:

- Throughput
- Concurrency
- Isolation
- Scalability
- Fault tolerance

---

# Alternatives Considered

---

## Option 1: Single Global Matching Engine

Architecture:

```text
Global Engine
    │
    ├── BTCUSDT Book
    ├── ETHUSDT Book
    ├── AAPL Book
    └── TSLA Book
```

---

### Advantages

- Simpler implementation.
- Easier startup logic.
- Centralized management.

---

### Disadvantages

- Every symbol shares the same worker.
- High contention.
- Poor scalability.
- One busy symbol slows every other symbol.
- Difficult to distribute across CPUs.

---

### Example

```text
BTCUSDT receives 100k orders/sec
ETHUSDT receives 100 orders/sec
```

ETHUSDT orders may experience delays because BTCUSDT dominates the engine.

---

## Option 2: One Engine Per Symbol

Architecture:

```text
BTCUSDT -> Engine #1
ETHUSDT -> Engine #2
AAPL    -> Engine #3
TSLA    -> Engine #4
```

---

### Advantages

- Complete isolation.
- Horizontal scalability.
- Independent throughput.
- Independent failure domains.
- Better CPU utilization.

---

### Disadvantages

- More memory usage.
- More goroutines.
- Slightly more management complexity.

---

# Decision

Velocity adopts:

```text
One Matching Engine Per Symbol
```

Each symbol owns:

- Order Book
- Matcher
- Order Queue
- Trade Queue
- Matching Worker

---

# Architecture

```text
Registry
│
├── BTCUSDT Engine
│      ├── OrderBook
│      ├── Matcher
│      ├── Order Queue
│      └── Trade Queue
│
├── ETHUSDT Engine
│      ├── OrderBook
│      ├── Matcher
│      ├── Order Queue
│      └── Trade Queue
│
├── AAPL Engine
│
└── TSLA Engine
```

---

# Registry

Velocity uses a registry to manage engines.

Current implementation:

```go
type Registry struct {
    engines map[string]*engine.Engine
    mu      sync.RWMutex
}
```

---

## Purpose

The registry acts as:

```text
Symbol -> Engine Lookup
```

Example:

```go
engine := registry.Get("BTCUSDT")
```

---

# Lazy Engine Creation

Engines are created only when needed.

Example:

```go
engine := registry.Get("BTCUSDT")
```

If the engine does not exist:

```go
engine.New("BTCUSDT")
```

is automatically called.

---

## Benefits

Avoids creating thousands of unused engines.

---

# Isolation Benefits

## Performance Isolation

Example:

```text
BTCUSDT -> 50,000 orders/sec
ETHUSDT -> 500 orders/sec
```

ETHUSDT performance remains unaffected.

---

## Failure Isolation

Example:

```text
BTCUSDT engine crashes
```

Effects:

```text
BTCUSDT unavailable
```

But:

```text
ETHUSDT remains operational
AAPL remains operational
TSLA remains operational
```

---

## Resource Isolation

Each engine owns:

- Memory
- Queues
- Order Book
- Worker

This prevents resource contention.

---

# Horizontal Scaling

Example:

```text
CPU Core 1 -> BTCUSDT
CPU Core 2 -> ETHUSDT
CPU Core 3 -> AAPL
CPU Core 4 -> TSLA
```

Throughput scales approximately linearly with CPU count.

---

# Example Flow

Incoming order:

```text
BUY BTCUSDT 100 @ 100000
```

---

Step 1:

Registry lookup:

```go
engine := registry.Get("BTCUSDT")
```

---

Step 2:

Order submission:

```go
engine.SubmitOrder(order)
```

---

Step 3:

BTCUSDT worker processes order.

No other symbol is affected.

---

# Memory Cost

Each engine owns:

```text
Order Book
Matcher
Queues
Worker
```

Memory usage increases with symbol count.

However:

Memory is significantly cheaper than lock contention and reduced throughput.

Velocity prioritizes performance and isolation over minimal memory usage.

---

# Concurrency Benefits

Because engines are isolated:

```text
BTCUSDT worker
```

can run simultaneously with:

```text
ETHUSDT worker
```

without synchronization.

---

## Example

```text
BTCUSDT Matching
ETHUSDT Matching
AAPL Matching
TSLA Matching
```

All execute concurrently.

---

# Future Improvements

---

## Engine Eviction

Inactive symbols may eventually be removed.

Example:

```text
No activity for 1 hour
```

Engine automatically shuts down.

---

## Dynamic Scaling

Future versions may move engines across machines.

Example:

```text
Server A
    BTCUSDT
    ETHUSDT

Server B
    AAPL
    TSLA
```

---

## Distributed Registry

The registry may eventually become:

```text
Distributed Service Discovery
```

using:

- Consul
- etcd
- Kubernetes

---

## Engine Replication

Future versions may support:

```text
Primary Engine
Replica Engine
```

for high availability.

---

# Industry Practice

Most modern exchanges use some variation of:

```text
One Instrument
=
One Matching Unit
```

Examples:

- NASDAQ
- NYSE
- CME
- Binance
- Coinbase

The terminology varies:

- Instrument Thread
- Matching Partition
- Symbol Worker
- Matching Unit

The underlying principle remains identical.

---

# Consequences

Velocity accepts:

- increased memory usage
- more goroutines
- registry complexity

in exchange for:

- scalability
- isolation
- performance
- simplicity of matching logic

---

# Final Decision

Velocity adopts:

```text
One Matching Engine Per Symbol
```

because it provides:

- Horizontal scalability
- Failure isolation
- Independent throughput
- Deterministic matching
- Efficient CPU utilization

This architecture forms one of the core scalability principles of the Velocity Matching Engine.