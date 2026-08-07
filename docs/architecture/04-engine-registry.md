# Engine Registry

## Purpose

The Engine Registry is responsible for managing the lifecycle of every matching engine within Velocity.

It acts as the central coordinator between the service layer and the matching engines by creating, storing, retrieving, recovering, and shutting down engine instances.

Each trading symbol is associated with exactly one matching engine, and the Engine Registry guarantees that only one engine exists for a given symbol at any point in time.

The registry is the only component responsible for engine ownership.

---

# Responsibilities

The Engine Registry is responsible for:

* Creating matching engines
* Maintaining a mapping between trading symbols and engines
* Returning engine instances to callers
* Lazily creating engines for new symbols
* Recovering engines during application startup
* Managing engine lifecycle
* Coordinating graceful shutdown
* Providing thread-safe engine lookup

The registry **does not** perform:

* Order matching
* Wallet operations
* Risk validation
* Settlement
* Authentication
* Market data generation

Those responsibilities belong to other components.

---

# Why a Registry?

Without a registry, every service would need to manage matching engines independently.

Instead, Velocity centralizes engine ownership.

```text
Order Service
        │
        ▼
Engine Registry
        │
        ├── BTCUSDT Engine
        ├── ETHUSDT Engine
        ├── SOLUSDT Engine
        └── ...
```

This provides:

* Single source of truth
* Controlled engine lifecycle
* Simplified dependency management
* Safe concurrent access
* Centralized recovery

---

# High-Level Architecture

```text
                 Business Services
                        │
                        ▼
                Engine Registry
                        │
      ┌─────────────────┼─────────────────┐
      │                 │                 │
BTCUSDT Engine    ETHUSDT Engine    SOLUSDT Engine
      │                 │                 │
      ▼                 ▼                 ▼
 Matching Engine   Matching Engine   Matching Engine
```

Every engine is completely isolated from every other engine.

---

# Engine Ownership

Each engine owns all in-memory state for its trading symbol.

Example:

```text
BTCUSDT Engine

├── OrderBook
├── StopBook
├── Matcher
├── Trade Queue
├── WAL
├── Snapshot Writer
├── Sequence Generator
└── Event Publisher
```

The registry owns the engine instance itself but never modifies the engine's internal state.

---

# One Engine Per Symbol

Velocity follows a strict one-engine-per-symbol model.

Example:

```text
BTCUSDT
    │
    ▼
Matching Engine
```

Every request involving BTCUSDT is routed to the same engine instance.

This guarantees:

* Deterministic execution
* Consistent ordering
* Symbol isolation

No two engines ever process commands for the same symbol.

---

# Lazy Engine Creation

Engines are created only when needed.

Workflow:

```text
Request Engine
        │
        ▼
Engine Exists?
     │        │
    Yes      No
     │        │
     ▼        ▼
 Return   Create Engine
              │
              ▼
        Store in Registry
              │
              ▼
        Return Engine
```

Advantages:

* Lower startup time
* Reduced memory usage
* Efficient resource utilization
* Automatic support for newly listed symbols

---

# Engine Lookup

Business services never construct engines directly.

Instead they request an engine from the registry.

Typical flow:

```text
HTTP Request
        │
        ▼
Order Service
        │
        ▼
Engine Registry
        │
        ▼
Matching Engine
```

This keeps engine ownership centralized.

---

# Thread Safety

Multiple goroutines may request engines simultaneously.

The registry therefore provides thread-safe access to its internal engine map.

Only the registry mutates the engine collection.

Individual engine state remains protected by the engine's single-threaded execution model.

---

# Startup Recovery

During application startup, the registry coordinates engine restoration.

Recovery sequence:

```text
Application Starts
        │
        ▼
Load Snapshots
        │
        ▼
Restore Engines
        │
        ▼
Replay WAL
        │
        ▼
Recover Remaining State
        │
        ▼
Register Restored Engines
        │
        ▼
Ready
```

Engines restored from snapshots are tracked to avoid duplicate recovery.

---

# Runtime Lifecycle

During normal operation, the registry manages engines throughout their lifetime.

```text
Application Starts
        │
        ▼
Create Registry
        │
        ▼
Recover Engines
        │
        ▼
Accept Requests
        │
        ▼
Lookup/Create Engines
        │
        ▼
Route Commands
        │
        ▼
Shutdown
```

---

# Graceful Shutdown

During shutdown, the registry coordinates engine termination.

Typical sequence:

```text
Shutdown Signal
        │
        ▼
Stop Accepting Requests
        │
        ▼
Stop Engine Processing
        │
        ▼
Flush WAL
        │
        ▼
Flush Snapshot
        │
        ▼
Stop Workers
        │
        ▼
Release Resources
```

The registry ensures engines stop in a controlled and deterministic manner.

---

# Design Principles

The Engine Registry follows several architectural principles.

## Centralized Ownership

All engine creation and destruction is handled in one place.

---

## Lazy Initialization

Resources are allocated only when required.

---

## Symbol Isolation

Each trading symbol is processed independently.

---

## Thread-Safe Lookup

Concurrent access to the registry is safe.

---

## Lifecycle Management

The registry owns engine creation, recovery, and shutdown.

---

## Separation of Concerns

The registry manages engines but never performs matching.

---

# Benefits

Using an Engine Registry provides several advantages:

* Single source of engine ownership
* Easier dependency injection
* Simplified service layer
* Better scalability
* Independent symbol processing
* Deterministic execution
* Simplified recovery
* Controlled shutdown

---

# Future Evolution

The current registry is designed to evolve as Velocity scales.

Future enhancements may include:

* Dynamic symbol listing
* Engine eviction for inactive symbols
* Distributed engine discovery
* Multi-node engine routing
* Engine sharding
* Leader election
* Cluster-aware registry
* Engine health monitoring
* Automatic engine migration

These enhancements should extend the registry without changing its responsibility as the owner of engine lifecycle.

---

# Related Documentation

* `02-system-architecture.md`
* `03-matching-engine.md`
* `concurrency-model.md`
* `orderbook.md`
* `ADR-002-single-threaded-matching.md`
* `ADR-004-engine-per-symbol.md`
