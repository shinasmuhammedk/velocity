# Application Bootstrap

## Purpose

The Application Bootstrap process is responsible for initializing the Velocity application and preparing every required dependency before the server begins accepting client requests.

It acts as the application's **Composition Root**, where all dependencies are created, configured, and wired together exactly once.

No business logic should exist inside the bootstrap process.

---

# Responsibilities

The bootstrap process is responsible for:

* Loading application configuration
* Initializing the logger
* Connecting to PostgreSQL
* Creating shared infrastructure components
* Creating repositories
* Creating services
* Creating the Engine Registry
* Initializing the HTTP server
* Registering routes and middleware
* Starting the server
* Handling graceful shutdown
* Stop HTTP server
* Drain request queue
* Stop symbol engines
* Flush persistence buffers
* Close database connections
* Flush logs

---

# Startup Flow

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
Initialize Metrics
        │
        ▼
Initialize Infrastructure
        │
        ▼
Create Repository Layer
        │
        ▼
Create Service Layer
        │
        ▼
Create Engine Registry
        │
        ▼
Load Active Symbols
        │
        ▼
Create One Engine Per Symbol
        │
        ▼
Recover Open Orders
        │
        ▼
Recover Stop Orders
        │
        ▼
Create HTTP Server
        │
        ▼
Register Routes
        │
        ▼
Start Fiber Server
        │
        ▼
Accept Client Requests
```

---

# Design Principles

* Single entry point (`cmd/api/main.go`)
* Dependency Injection
* Constructor-based initialization
* Fail Fast on startup errors
* Graceful shutdown
* No global mutable state
* One matching engine per symbol
* Single-threaded deterministic matching
* In-memory hot path
* Database outside matching path

---

# Folder Ownership

```text
cmd/api/
internal/app/
internal/bootstrap/
internal/engine/registry/
```

---

# Future Expansion

The bootstrap process will later initialize:

* Redis
* Kafka/NATS
* WebSocket Hub
* Worker Pool
* Background Jobs
* Distributed Tracing
* Risk Engine
* Snapshot Recovery
* Event Sourcing
* Persistence Worker
* Market Data Service

These components should be added without changing the startup philosophy.
