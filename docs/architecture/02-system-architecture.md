# System Architecture

## Purpose

This document provides a high-level overview of the Velocity architecture, its major subsystems, and how they interact to form a production-grade cryptocurrency exchange backend.

It serves as the primary architectural reference for contributors and should be read before exploring individual components such as the matching engine, services, or persistence layer.

---

# What is Velocity?

Velocity is a production-grade cryptocurrency exchange backend inspired by modern digital asset exchanges such as Binance, Coinbase, Bybit, and Kraken.

Its primary responsibility is to provide a reliable, deterministic, and scalable trading platform capable of processing large volumes of orders while maintaining strict correctness guarantees.

Velocity is **not** a trading bot or exchange frontend. It is the backend infrastructure responsible for:

* User authentication and authorization
* Order management
* Trade execution
* Wallet management
* Risk validation
* Settlement
* Market data generation
* Real-time event distribution
* Persistence and recovery

---

# Design Goals

Velocity is designed with the following objectives:

* Deterministic order execution
* Price-Time Priority matching
* Low-latency order processing
* High throughput
* Fault tolerance
* Crash recovery
* Horizontal scalability
* Modular architecture
* Event-driven communication
* Observability and monitoring

Long-term performance targets include:

* 100,000+ orders per second
* Sub-millisecond matching latency
* Multi-symbol parallelism
* Distributed deployment

---

# High-Level Architecture

```text
                         Clients
                            │
        ┌───────────────────┴───────────────────┐
        │                                       │
    HTTP REST API                         WebSocket
        │                                       │
        └───────────────┬───────────────────────┘
                        │
                Authentication Middleware
                        │
                        ▼
            Identity Service (gRPC)
                        │
                JWT Validation
                        │
                        ▼
                 Business Services
                        │
        ┌───────────────┼────────────────┐
        │               │                │
   Wallet Service   Order Service   Market Service
        │               │                │
        └───────────────┴────────────────┘
                        │
                        ▼
               Engine Registry
                        │
        ┌───────────────┼───────────────┐
        │               │               │
     BTCUSDT         ETHUSDT         SOLUSDT
      Engine          Engine          Engine
        │
        ▼
 ┌───────────────────────────────┐
 │ OrderBook                     │
 │ Matcher                       │
 │ StopBook                      │
 │ Command Processing            │
 │ Trade Queue                   │
 │ WAL                           │
 │ Snapshot                      │
 │ Event Publisher               │
 └───────────────────────────────┘
                        │
                        ▼
                 Event Dispatcher
                        │
     ┌──────────┬──────────┬──────────┬──────────┐
     │          │          │          │          │
 Statistics  Candles  Market Data  Settlement  Persistence
     │                                  │
     └──────────────────────────────────┴─────────────► PostgreSQL
```

---

# Core Components

## Transport Layer

The transport layer exposes Velocity to external clients.

Current transports include:

* HTTP REST API (Fiber)
* Public WebSocket
* Private authenticated WebSocket
* gRPC client for Identity Service communication

The transport layer contains no business logic and delegates requests to the service layer.

---

## Authentication

Authentication is delegated to an external Identity Service.

Request flow:

```text
Client
    │
JWT Token
    │
Authentication Middleware
    │
Identity Service (gRPC)
    │
Validate Token
    │
Authenticated User
    │
Request Context
```

Velocity trusts the Identity Service as the source of authentication and maintains only trading-related user data locally.

---

## Service Layer

The service layer coordinates business operations and acts as the boundary between transport and the matching engine.

Current services include:

* Order Service
* Wallet Service
* User Service
* Risk Service
* Settlement Service
* Position Service
* Market Service
* Statistics Service
* Candle Service

Services orchestrate workflows but do not perform order matching directly.

---

## Engine Registry

The Engine Registry owns every matching engine instance.

Each trading symbol has its own independent engine.

Example:

```text
Registry
    │
    ├── BTCUSDT Engine
    ├── ETHUSDT Engine
    ├── SOLUSDT Engine
    └── ...
```

Engines are created lazily and remain isolated from one another.

---

## Matching Engine

The matching engine is the core of Velocity.

Each engine is responsible for a single trading symbol and processes commands sequentially in a dedicated goroutine.

Major components:

* OrderBook
* Matcher
* StopBook
* Command Processing
* Trade Queue
* WAL Writer
* Snapshot Writer
* Event Publisher
* Sequence Generator

This design guarantees deterministic execution without locking inside the matching path.

---

## Event System

Velocity follows an event-driven architecture.

After successful trade execution, the engine publishes events that are consumed by independent subscribers.

Current consumers include:

* Market Data
* Statistics
* Candle Generation
* Settlement
* Persistence
* User Notifications

This separation keeps the matching engine lightweight and allows subsystems to evolve independently.

---

## Persistence

Persistent storage is provided by PostgreSQL.

Trading state is stored asynchronously through background workers to avoid blocking the matching engine.

Persistent entities include:

* Users
* Wallets
* Orders
* Trades
* Positions
* Symbols

Velocity also uses:

* Write-Ahead Log (WAL)
* Snapshots

to support deterministic crash recovery.

---

## Recovery

During startup, Velocity restores engine state before accepting client requests.

Recovery sequence:

1. Load snapshots.
2. Restore engine state.
3. Replay WAL entries when required.
4. Recover remaining state from the database.
5. Start matching engines.

This ensures consistency between in-memory state and persistent storage.

---

## Market Data

The market data subsystem generates real-time exchange information.

Current capabilities include:

* Trades
* Order Book Depth
* Ticker
* Market Statistics
* Candlestick Data

Updates are distributed over public WebSocket connections.

---

## Wallet, Risk, and Settlement

Before an order reaches the matching engine:

1. The authenticated user is synchronized locally.
2. Risk validation is performed.
3. Wallet balances are checked.
4. Required funds are locked.

After execution:

* Trades are persisted.
* Wallet balances are updated.
* Positions are updated.
* User events are published.

This separation keeps trading logic isolated from financial accounting.

---

# Request Lifecycle

A typical authenticated order request follows this path:

```text
Client
    │
HTTP Request
    │
Authentication Middleware
    │
Identity Service (gRPC)
    │
User Synchronization
    │
Order Service
    │
Risk Validation
    │
Wallet Validation
    │
Engine Registry
    │
Matching Engine
    │
Trade Events
    │
Persistence
    │
Settlement
    │
Market Data
    │
WebSocket Broadcast
```

---

# Architectural Principles

Velocity follows several core architectural principles:

* Separation of concerns
* Constructor-based dependency injection
* Single responsibility
* Event-driven communication
* Deterministic execution
* One engine per symbol
* Single-threaded matching
* Asynchronous persistence
* Fail-fast startup
* Graceful shutdown
* Modular system boundaries

These principles guide all new development and architectural decisions.

---

# Repository Overview

The project is organized into the following major areas:

```text
cmd/
    Application entry points

internal/
    Core business logic

configs/
    Application configuration

deployments/
    Deployment assets

docs/
    Technical documentation

pkg/
    Shared reusable packages
```

---

# Future Evolution

The current architecture forms the foundation for future enhancements, including:

* Redis caching
* Kafka or NATS event streaming
* OpenTelemetry tracing
* Grafana dashboards
* Multi-node engine deployment
* Engine sharding
* Kubernetes orchestration
* Distributed recovery
* Futures and perpetual markets
* Liquidation engine
* FIX protocol support

These additions are intended to extend the existing architecture while preserving its modular and deterministic design.
