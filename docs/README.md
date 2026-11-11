# Velocity Documentation

This folder is the source of truth for how Velocity is designed and why.
If the code and a document here ever disagree, that's a bug in one of
them - please open an issue (or fix it) rather than trusting whichever
one you read first. See in particular the note in
[`sequence-diagrams/full-fill.md`](sequence-diagrams/full-fill.md) about
a known documentation/code mismatch around settlement.

## Where to start

- New to the codebase? Read
  [`architecture/02-system-architecture.md`](architecture/02-system-architecture.md)
  first, then [`architecture/01-application-bootstrap.md`](architecture/01-application-bootstrap.md)
  to see how the pieces get wired together at startup.
- Working on the matching engine specifically? Start with
  [`architecture/03-matching-engine.md`](architecture/03-matching-engine.md)
  and [`architecture/orderbook.md`](architecture/orderbook.md).
- Trying to understand *why* something was built a particular way rather
  than *how* it works? That's what the ADRs are for - see below.

## Architecture Decision Records (`adr/`)

Each ADR documents one deliberate design decision, the alternatives that
were considered, and why the chosen one won. Read these when you're
tempted to change something fundamental - the reasoning for *not* doing
it the obvious other way is usually already written down.

| ADR | Decision |
|---|---|
| [ADR-001](adr/ADR-001-price-time-priority.md) | Price-time priority matching |
| [ADR-002](adr/ADR-002-single-threaded-matching.md) | Single-threaded matching per symbol |
| [ADR-003](adr/ADR-003-event-driven-engine.md) | Event-driven matching engine architecture |
| [ADR-004](adr/ADR-004-engine-per-symbol.md) | One matching engine per symbol |
| [ADR-005](adr/ADR-005-fifo-price-levels.md) | In-memory order book architecture |

## Architecture (`architecture/`)

| Doc | Covers |
|---|---|
| [01-application-bootstrap.md](architecture/01-application-bootstrap.md) | How `cmd/api` wires up the whole dependency graph at startup |
| [02-system-architecture.md](architecture/02-system-architecture.md) | The system as a whole - services, data stores, request paths |
| [02-configuration.md](architecture/02-configuration.md) | How config is loaded, layered, and overridden by environment |
| [03-logging.md](architecture/03-logging.md) | Structured logging conventions |
| [03-matching-engine.md](architecture/03-matching-engine.md) | The matching engine's internal design |
| [04-engine-registry.md](architecture/04-engine-registry.md) | Per-symbol engine lifecycle - creation, lookup, shutdown |
| [04-git-workflow.md](architecture/04-git-workflow.md) | Branching model (see also `/CONTRIBUTING.md`) |
| [05-event-driven-architecture.md](architecture/05-event-driven-architecture.md) | The in-process event dispatcher and its Kafka bridge |
| [06-recovery-flow.md](architecture/06-recovery-flow.md) | WAL + snapshot replay on startup |
| [concurrency-model.md](architecture/concurrency-model.md) | Goroutine and channel design across the engine |
| [data-structures.md](architecture/data-structures.md) | Core in-memory data structures (heaps, FIFO price levels, etc.) |
| [matching-engine-architecture.md](architecture/matching-engine-architecture.md) | Extended matching engine design notes |
| [orderbook.md](architecture/orderbook.md) | Order book internals in detail |

## Database (`database/`)

| Doc | Covers |
|---|---|
| [01-database-architecture.md](database/01-database-architecture.md) | Postgres access layer conventions (repositories, SQLC, migrations) |

## Sequence diagrams (`sequence-diagrams/`)

Step-by-step traces through the codebase for the three core moments in
an order's life, each with exact source references:

| Doc | Covers |
|---|---|
| [order-placement.md](sequence-diagrams/order-placement.md) | HTTP request → risk check → fund lock → engine acceptance |
| [partial-fill.md](sequence-diagrams/partial-fill.md) | What happens inside the matcher on a partial match |
| [full-fill.md](sequence-diagrams/full-fill.md) | Trade execution → settlement → wallet/position updates. **Also documents a real mismatch between the README and the actual settlement wiring — read this one if you're relying on the README's architecture diagram.** |

## Performance (`performance/`)

| Doc | Covers |
|---|---|
| [benchmark-results.md](performance/benchmark-results.md) | Measured throughput numbers, methodology, and caveats |
| [performance-targets.md](performance/performance-targets.md) | Proposed (not yet ratified) targets derived from the benchmarks above |

## Keeping this folder honest

A few conventions this folder already follows, worth keeping up:

- Cite the exact file/function a claim comes from, so a reader can go
  check it themselves.
- State machine/environment specifics for any measured number
  (`benchmark-results.md` is the model to follow).
- Flag disagreements between docs and code explicitly rather than
  quietly picking one - see the settlement note in `full-fill.md`.
- Mark anything not yet measured or not yet decided as such (`TBD`,
  "not yet measured") instead of guessing a plausible-looking number.