# Performance Targets

## Status: proposed, not yet ratified

This document does not currently reflect a business requirement or an
SLA anyone has signed off on. `docs/performance/benchmark-results.md`
says so explicitly:

> "This document reports what the engine *can* sustain per symbol with
> the WAL as currently implemented — it does not answer whether that is
> *enough*. That depends on the peak per-symbol order rate this system
> needs to support, which has not yet been specified anywhere in this
> repo."

The numbers below are a **starting proposal derived from what has
already been measured**, so that there is something concrete to react
to, revise, or reject - not a specification to build against yet. Treat
every number here as provisional until someone with authority over
actual expected load signs off on it, and update the "Status" line above
once that happens.

## Proposed targets, by area

### Matching engine throughput (per symbol)

| Metric | Measured today | Proposed target | Basis |
|---|---:|---:|---|
| Sustained orders/sec/symbol, WAL enabled, non-crossing | ~5,000–7,000 | ≥ 5,000 | `benchmark-results.md`, Run 1/Run 2, non-crossing workload |
| Sustained orders/sec/symbol, WAL enabled, matching-heavy | ~5,000–7,000 (near-parity with non-crossing) | ≥ 5,000 | `benchmark-results.md`, matching-heavy section |
| Sustained orders/sec/symbol, no WAL (upper bound, not production-representative) | ~600,000+ | n/a - reference ceiling only | `benchmark-results.md` |

The proposed target simply restates the low end of what's already been
measured. If real order volume for any symbol is expected to exceed
~5,000/sec, the WAL write path needs work before that symbol goes live -
see "Candidate mitigation" in `benchmark-results.md` (group commit).

### API latency (not yet benchmarked - no data exists for these rows)

| Metric | Measured today | Proposed target |
|---|---|---|
| `POST /api/orders` p50 (accept, not fill) | not measured | TBD |
| `POST /api/orders` p99 | not measured | TBD |
| Order cancel p99 | not measured | TBD |
| WebSocket public feed fan-out latency (trade executed -> client receives) | not measured | TBD |

These rows are placeholders on purpose. Do not fill them in with guessed
numbers - run a load test against the HTTP/WebSocket layer (not just the
engine in isolation, which is all `benchmark-results.md` currently
covers) and record real numbers, machine spec, and methodology the same
way `benchmark-results.md` does.

### Recovery time

| Metric | Measured today | Proposed target |
|---|---|---|
| Engine recovery time after restart (snapshot + WAL replay), per symbol | not measured | TBD - depends on snapshot interval and WAL size since last snapshot |

`engine.snapshot_interval` (see `docs/architecture/02-configuration.md`)
directly trades off recovery time against steady-state snapshot-write
overhead. No target can be set here until someone measures replay time
at a realistic WAL size.

### Settlement

| Metric | Measured today | Proposed target |
|---|---|---|
| Trade -> settled (wallets/positions/order status committed) latency | not measured | TBD |

Note that settlement currently runs synchronously in-process inside the
`api` binary (see `docs/sequence-diagrams/full-fill.md` for why this
differs from the README's description of an async Kafka-consumed
worker). Any target set here should account for whichever architecture
is actually true at the time it's measured, since the two have very
different latency and failure characteristics.

## How to use this document

1. If you're deciding whether current performance is "good enough,"
   don't use this document as authority - it isn't ratified. Ask whoever
   owns the product requirement for the actual expected peak load per
   symbol, then compare that number to `benchmark-results.md`.
2. If you're setting a real target, replace the relevant "Proposed
   target" cell, update "Status" at the top of this file, and link to
   whatever decision or ticket established it.
3. If you add a new benchmark, add or update the matching row in this
   table with a link to the source in `benchmark-results.md`, so the two
   documents never drift.