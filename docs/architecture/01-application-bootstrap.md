# Application Bootstrap

## Purpose

The Application Bootstrap process is responsible for constructing, configuring, and wiring the entire Velocity application before it begins accepting client requests.

It acts as the **Composition Root** of the application, where every dependency is created exactly once and injected into the components that require it.

The bootstrap layer contains **no business logic**. Its sole responsibility is dependency construction, initialization, startup sequencing, and graceful shutdown.

---

# Objectives

The bootstrap process ensures that:

* All required dependencies are initialized in the correct order.
* Every subsystem receives its required dependencies through constructor injection.
* Startup failures terminate the application immediately (Fail Fast).
* The matching engine starts only after persistence and recovery have completed.
* Resources are released safely during shutdown.

---

# Responsibilities

The bootstrap process is responsible for:

* Loading application configuration
* Initializing structured logging
* Connecting to PostgreSQL
* Registering Prometheus metrics
* Creating SQLC repositories
* Initializing shared infrastructure
* Creating the event dispatcher
* Creating the market data hub
* Creating the private user websocket hub
* Creating the statistics manager
* Creating the candle manager
* Creating the Write-Ahead Log (WAL)
* Creating the snapshot manager
* Creating recovery services
* Creating background workers
* Creating the Engine Registry
* Recovering engine state
* Creating business services
* Initializing the Identity Service gRPC client
* Configuring authentication middleware
* Creating the Fiber application
* Registering middleware
* Registering HTTP routes
* Starting the HTTP server
* Handling graceful shutdown

---

# Startup Sequence

The startup process follows a strict dependency order.

```text
Application Starts
        │
        ▼
Load Configuration
        │
        ▼
Initialize Logger
        │
        ▼
Connect PostgreSQL
        │
        ▼
Register Prometheus Metrics
        │
        ▼
Create SQLC Repositories
        │
        ▼
Create Event Dispatcher
        │
        ▼
Create WAL Manager
        │
        ▼
Create Snapshot Manager
        │
        ▼
Create Market Data Hub
        │
        ▼
Create User WebSocket Hub
        │
        ▼
Create Statistics Manager
        │
        ▼
Create Candle Manager
        │
        ▼
Create Background Workers
        │
        ▼
Create Engine Registry
        │
        ▼
Recover Snapshots
        │
        ▼
Replay WAL (if applicable)
        │
        ▼
Recover Remaining Database State
        │
        ▼
Initialize Identity gRPC Client
        │
        ▼
Create Business Services
        │
        ▼
Create Fiber Application
        │
        ▼
Register Middleware
        │
        ▼
Register Routes
        │
        ▼
Start HTTP Server
        │
        ▼
Accept Client Requests
```

---

# High-Level Dependency Graph

```text
Configuration
        │
        ▼
Logger
        │
        ▼
Database
        │
        ▼
Repositories
        │
        ▼
Infrastructure
        │
        ├── Event Dispatcher
        ├── WAL
        ├── Snapshot
        ├── Recovery
        ├── Metrics
        ├── Market Data
        ├── User Stream
        ├── Statistics
        └── Candles
        │
        ▼
Engine Registry
        │
        ▼
Business Services
        │
        ▼
Transport Layer
        ├── HTTP
        ├── WebSocket
        └── gRPC Client
```

---

# Recovery During Bootstrap

Before the application begins accepting requests, the bootstrap process restores the in-memory trading state.

Recovery order:

1. Load snapshots.
2. Restore matching engines from snapshots.
3. Replay any required WAL entries.
4. Recover symbols that were not restored from snapshots.
5. Start all recovered engines.

This guarantees that engine state is restored before any new commands are processed.

---

# Authentication Initialization

Velocity delegates authentication to an external Identity Service.

During startup the bootstrap process:

* Creates the Identity Service gRPC client.
* Verifies connectivity (where applicable).
* Injects the client into the authentication middleware.
* Makes the middleware available to protected HTTP routes.

Velocity never validates JWTs directly; it relies on the Identity Service for token verification.

---

# Background Components

Several components created during bootstrap run independently after initialization:

* Matching engines
* Trade persistence worker
* Market data broadcaster
* Statistics subscriber
* Candle subscriber
* User event dispatcher
* WebSocket hubs

These components communicate through events while remaining loosely coupled.

---

# Design Principles

The bootstrap process follows these principles:

* Single application entry point (`cmd/api/main.go`)
* Constructor-based dependency injection
* Explicit dependency ownership
* Fail Fast startup
* Graceful shutdown
* No global mutable state
* One engine per symbol
* Deterministic single-threaded matching
* Asynchronous persistence
* Separation of infrastructure and business logic

---

# Graceful Shutdown

When the application receives a termination signal, shutdown occurs in the following order:

```text
Receive Shutdown Signal
        │
        ▼
Stop Accepting HTTP Requests
        │
        ▼
Drain In-flight Requests
        │
        ▼
Stop Background Workers
        │
        ▼
Stop Matching Engines
        │
        ▼
Flush WAL Buffers
        │
        ▼
Flush Snapshot/Persistence Buffers
        │
        ▼
Close WebSocket Connections
        │
        ▼
Close Database Connections
        │
        ▼
Sync Logger
        │
        ▼
Exit
```

---

# Folder Ownership

```text
cmd/api/
internal/app/
internal/bootstrap/
internal/engine/
internal/events/
internal/transport/
internal/persistence/
internal/service/
```

---

# Future Enhancements

The bootstrap process is designed to accommodate additional infrastructure without changing its overall philosophy.

Planned additions include:

* Redis
* Kafka or NATS
* OpenTelemetry
* Grafana integration
* Distributed engine nodes
* Scheduler services
* Background job framework
* Distributed cache
* Service discovery
* Kubernetes-specific initialization
* Multi-node engine coordination

Each new subsystem should be initialized through constructor injection while preserving the existing startup order and dependency boundaries.
