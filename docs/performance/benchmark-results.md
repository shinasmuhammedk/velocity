# Benchmark Results

## Sustained order-submission throughput: no WAL vs. real WAL

**Source:** `test/stress/wal_backed_throughput_test.go` — `TestStress_SustainedThroughput_WithAndWithoutRealWAL`

**What was measured:** the same non-crossing SELL order workload
(8 concurrent producers, 10 seconds), run twice back-to-back in the same
process on the same machine:

1. `engine.New(symbol, nil, nil)` — no WAL writer attached. This is the
   configuration every existing benchmark in `test/load` uses.
2. `engine.New(symbol, walWriter, nil)` — a real `wal.Writer`, backed by an
   actual file on local disk, with the engine's normal fsync-per-write
   behavior (`internal/engine/wal/writer.go`) left untouched.

### Result (one run, one machine — see caveats below)

| Configuration | Orders submitted | Orders/sec |
|---|---:|---:|
| No WAL | 6,192,680 | 619,237.92 |
| Real WAL (fsync per write) | 47,397 | 4,738.49 |

**Slowdown factor: ~130.7x**

### Why this gap exists

`SubmitOrder` is synchronous per call — it blocks on the command's result
channel until the engine's single worker goroutine has fully processed the
command, WAL write included (`internal/engine/engine.go`). `wal.Writer.Write()`
calls `file.Write()` followed by a full `file.Sync()` on every single
command — not buffered, not batched. Sustained per-symbol throughput is
therefore bounded by how fast that one goroutine can push commands through
fsync, not by CPU, and not by how many producer goroutines are submitting
concurrently.

### Why this matters

**Every throughput number currently reported by `test/load`
(`TestEngineLoad_8Producers_30Seconds` and its siblings) uses the no-WAL
configuration.** Production always attaches a real `wal.Writer` per symbol
(`registry.Get()`), so those numbers do not reflect what any single symbol
can actually sustain in production. The WAL-backed figure above —
not the no-WAL one — is the number that should inform capacity planning
per symbol.

### Caveats

- **This number is disk- and filesystem-dependent.** fsync latency varies
  significantly across local NVMe, spinning disk, network-attached
  storage, and tmpfs-backed temp directories (some CI runners mount `/tmp`
  as tmpfs, which makes fsync artificially cheap and would understate the
  real-world gap). Re-run this test on target production storage before
  using the absolute number for capacity planning; the *existence* and
  *rough scale* of the gap is the durable finding, not the exact 130.7x.
- **Single run, single machine.** This has not yet been run repeatedly or
  across different hardware to establish a confidence interval.
- **Non-crossing workload only.** This measures the cheapest possible
  per-order WAL cost (one SUBMIT event, no matching). `test/load` also has
  matching-heavy scenarios (`TestEngineLoad_Matching_4Producers_30Seconds`,
  `TestEngineLoad_Matching_1Producer_30Seconds`) that have not yet been
  re-measured with a real WAL attached. Matching produces additional WAL
  events per trade (fills on both sides), so the gap for a matching-heavy
  workload is expected to differ from the non-crossing number above and
  is worth measuring separately.

## Open question: is this throughput acceptable?

This document reports what the engine *can* sustain per symbol with the
WAL as currently implemented — it does not answer whether that is
*enough*. That depends on the peak per-symbol order rate this system
needs to support, which has not yet been specified anywhere in this repo.
Establishing that target is a prerequisite for deciding whether the
WAL write path needs to change at all.

## Candidate mitigation (not yet implemented, not yet scoped)

If 4,738 orders/sec/symbol turns out to be insufficient, the standard
mitigation for exactly this shape of bottleneck is **group commit**:
batch several pending WAL writes and issue a single fsync for the batch,
rather than one fsync per command (the same technique Postgres's WAL and
Kafka's log segments both use). This trades a small amount of added
per-order latency (time spent waiting to batch) for a large multiple in
throughput. This has deliberately not been designed or implemented as
part of this benchmark — it should be scoped against an actual target
throughput and latency budget, not built speculatively.

## Methodology for future benchmark runs

To reproduce or extend these results:

```
go test ./test/stress/... -run TestStress_SustainedThroughput -v
```

When adding new benchmark results to this file, always report:

- the exact workload (order type, crossing vs. non-crossing, producer count)
- WAL configuration (attached or not; disk type if known)
- duration and raw submitted/error counts, not just the derived rate
- the machine/environment the numbers came from

so that future readers can judge how much weight to put on any given
number, the same way the caveats above apply to this one.