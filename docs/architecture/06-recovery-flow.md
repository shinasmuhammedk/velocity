# Recovery Flow

## Purpose

The Recovery Flow is responsible for restoring Velocity to a consistent and operational state after an unexpected shutdown or application restart.

Its primary objective is to rebuild the in-memory state of every matching engine before the system begins accepting new client requests.

Velocity achieves this using a layered recovery strategy based on:

* Snapshots
* Write-Ahead Log (WAL)
* Persistent database state

This minimizes startup time while preserving deterministic engine behavior.

---

# Design Goals

The recovery system is designed to provide:

* Crash recovery
* Deterministic state restoration
* Minimal recovery time
* Data consistency
* Fault tolerance
* Safe startup sequencing
* Idempotent recovery
* Symbol-level isolation

---

# Why Recovery Is Required

The matching engine maintains its active trading state entirely in memory.

This includes:

* Order books
* Stop books
* Sequence numbers
* Last trade price
* Active engine state

If the application terminates unexpectedly, this in-memory state is lost.

Recovery reconstructs that state before trading resumes.

---

# Recovery Architecture

```text id="h3q9v2"
                Application Starts
                        │
                        ▼
              Initialize Infrastructure
                        │
                        ▼
              Snapshot Recovery
                        │
                        ▼
                 WAL Recovery
                        │
                        ▼
            Database State Recovery
                        │
                        ▼
              Engine Registration
                        │
                        ▼
              Accept Client Requests
```

Recovery is completed before the HTTP server begins processing requests.

---

# Recovery Sources

Velocity restores state from three independent sources.

## 1. Snapshots

Snapshots contain serialized engine state captured periodically during normal operation.

Typical snapshot contents include:

* Order book
* Stop book
* Sequence number
* Last trade price

Snapshots provide the fastest recovery mechanism.

---

## 2. Write-Ahead Log (WAL)

The WAL records every engine command before execution.

Typical commands include:

* Submit Order
* Cancel Order
* Modify Order

If commands exist after the latest snapshot, they can be replayed to restore the latest engine state.

---

## 3. Database

Persistent entities stored in PostgreSQL provide the final source of truth.

Examples include:

* Orders
* Trades
* Positions
* Users
* Wallets
* Symbols

Database recovery reconstructs symbols or state not already restored through snapshots.

---

# Startup Recovery Sequence

Velocity follows a strict startup order.

```text id="m2qsl6"
Load Configuration
        │
        ▼
Connect Database
        │
        ▼
Create Infrastructure
        │
        ▼
Load Snapshots
        │
        ▼
Restore Matching Engines
        │
        ▼
Replay WAL Entries
        │
        ▼
Recover Remaining Database State
        │
        ▼
Register Engines
        │
        ▼
Start Matching Engines
        │
        ▼
Start HTTP Server
```

This ordering guarantees that engine state is available before client requests are accepted.

---

# Snapshot Recovery

The snapshot manager scans available snapshots during startup.

For each snapshot:

1. Deserialize engine state.
2. Reconstruct the matching engine.
3. Restore in-memory structures.
4. Register the engine.

Benefits include:

* Faster startup
* Reduced WAL replay
* Lower database load

---

# WAL Replay

After restoring snapshots, the recovery process replays any commands that occurred after the snapshot was taken.

Replay sequence:

```text id="b3zkpn"
Load Snapshot
        │
        ▼
Read WAL Entries
        │
        ▼
Replay Commands
        │
        ▼
Restore Latest State
```

Because the engine is deterministic, replaying the same command sequence reproduces the original engine state.

---

# Database Recovery

Not every symbol will necessarily have a snapshot.

Symbols without snapshots are restored from the database.

Typical process:

```text id="8yk4ng"
Load Active Symbols
        │
        ▼
Find Open Orders
        │
        ▼
Rebuild Order Book
        │
        ▼
Create Engine
```

This ensures every active market becomes operational after startup.

---

# Duplicate Recovery Prevention

A symbol may appear in multiple recovery sources.

For example:

```text id="g7n3n2"
BTCUSDT
    │
Snapshot Exists
    │
Database Contains Orders
```

Without protection, the engine could be reconstructed twice.

Velocity prevents this by tracking symbols already restored from snapshots.

Recovery logic:

```text id="w6c2j9"
Snapshot Restored?
      │
   Yes │ No
      │
 Skip Database Recovery
      │
      ▼
 Continue
```

This guarantees that every symbol is restored exactly once.

---

# Recovery Guarantees

After successful recovery:

* Order books are restored.
* Stop books are restored.
* Sequence numbers continue correctly.
* Matching engines resume processing.
* Duplicate engine creation is prevented.
* Trading resumes from a consistent state.

Recovery completes before any new commands enter the engine.

---

# Deterministic Replay

The recovery process depends on deterministic engine execution.

Given:

* Identical snapshot
* Identical WAL
* Identical command order

The engine always reconstructs the same state.

This property is essential for correctness and simplifies debugging.

---

# Failure Handling

If recovery encounters an unrecoverable error:

* Startup is aborted.
* The application exits.
* No HTTP endpoints become available.

Velocity follows a **Fail Fast** startup philosophy to avoid serving inconsistent engine state.

---

# Recovery Lifecycle

```text id="c8hx7x"
Unexpected Shutdown
        │
        ▼
Application Restart
        │
        ▼
Snapshot Recovery
        │
        ▼
WAL Replay
        │
        ▼
Database Recovery
        │
        ▼
Register Engines
        │
        ▼
Resume Trading
```

---

# Design Principles

The recovery subsystem follows several architectural principles.

## Snapshot First

Snapshots minimize startup time.

---

## WAL for Consistency

Commands written after the snapshot are recovered through replay.

---

## Database as Persistent Storage

Persistent records provide recovery for symbols not restored by snapshots.

---

## One Recovery Per Symbol

Each symbol is reconstructed exactly once.

---

## Deterministic Restoration

Recovery always produces the same engine state for the same recovery inputs.

---

## Fail Fast

Startup terminates immediately if recovery cannot guarantee a consistent state.

---

# Future Enhancements

The recovery system is designed to evolve alongside the exchange.

Potential future improvements include:

* Incremental snapshots
* Snapshot compression
* Parallel symbol recovery
* WAL integrity verification
* Snapshot versioning
* Checkpoint scheduling
* Distributed recovery
* Multi-node engine restoration
* Cloud-based snapshot storage

These enhancements should preserve the existing recovery guarantees while improving startup performance and operational resilience.

---

# Related Documentation

* `01-application-bootstrap.md`
* `02-system-architecture.md`
* `03-matching-engine.md`
* `04-engine-registry.md`
* `05-event-driven-architecture.md`
* `wal.md`
* `snapshots.md`
