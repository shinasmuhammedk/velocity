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

### Results (see caveats below — numbers vary by machine/disk)

| Run | Configuration | Orders submitted | Orders/sec |
|---|---|---:|---:|
| Run 1 (original) | No WAL | 6,192,680 | 619,237.92 |
| Run 1 (original) | Real WAL (fsync per write) | 47,397 | 4,738.49 |
| Run 2 (reporter's local machine, Windows, disk type unconfirmed) | No WAL | 6,361,017 | 636,077.27 |
| Run 2 (reporter's local machine, Windows, disk type unconfirmed) | Real WAL (fsync per write) | 68,405 | 6,839.50 |

**Slowdown factor: ~130.7x (Run 1), ~93.0x (Run 2)**

The two runs land in the same rough order of magnitude but disagree by
~40% on both the no-WAL ceiling and the WAL-backed floor — exactly the
kind of machine-to-machine variance the caveats below warned about
before there was a second run to confirm it. Treat "roughly 5,000–7,000
orders/sec/symbol with WAL, on the hardware measured so far" as the
current planning range, not either single number.

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
- **Non-crossing workload only, in this section.** This measures the
  cheapest possible per-order WAL cost (one SUBMIT event, no matching).
  See "Matching-heavy sustained throughput" below for the crossing case,
  which was unmeasured when this section was first written.

## Matching-heavy sustained throughput: no WAL vs. real WAL

**Source:** `test/stress/wal_backed_matching_throughput_test.go` —
`TestStress_SustainedMatchingThroughput_WithAndWithoutRealWAL`

**What was measured:** a 100%-crossing workload — every submitted BUY
matches immediately against a large resting SELL, so every order produces
a trade — run the same two-phase way as the non-crossing test above (no
WAL, then a real `wal.Writer` with fsync-per-write), in one process on one
machine (reporter's local Windows machine, disk type unconfirmed; same
run as "Run 2" above, so the two are directly comparable).

### Result (one run, one machine — see caveats below)

| Configuration | Orders submitted | Trades | Orders/sec | Trades/sec |
|---|---:|---:|---:|---:|
| No WAL | 7,587,874 | 7,587,874 | 758,753.46 | 758,753.46 |
| Real WAL (fsync per write) | 67,762 | 67,762 | 6,776.03 | 6,776.03 |

**Slowdown factor: ~112.0x**

### This contradicts the prediction this document made

The caveat this section replaces predicted matching would be
meaningfully *slower* than the non-crossing case under a real WAL,
reasoning that a fill produces additional WAL events beyond the SUBMIT
event a non-crossing order writes. The measured result doesn't bear that
out: **6,776 orders/sec (matching) vs. 6,839 orders/sec (non-crossing,
same run) — about 1% apart**, well within normal run-to-run noise, not a
meaningfully different regime.

The likely explanation: `SubmitOrder` is synchronous and each WAL write
is a full `file.Write()` + `file.Sync()` on the same single-writer
goroutine either way (see "Why this gap exists" above). fsync latency —
not the number of logical WAL events per order — appears to dominate the
cost in both workloads, so whatever extra event(s) a fill adds doesn't
show up as a proportionally extra fsync in this engine's current WAL
write path. This is a plausible explanation, not a confirmed one — it
would take instrumenting WAL write counts per order (not just orders/sec)
to state it as fact rather than inference.

**Practical takeaway:** for capacity planning, the non-crossing number is
not a "best case that matching will undercut" — on this evidence, treat
both workload shapes as landing in the same ~5,000–7,000 orders/sec/symbol
range until re-measured on more hardware.

### Caveats

- Same disk/filesystem/single-run/single-machine caveats as the
  non-crossing section apply here — see above.
- **Producer count differs from the non-crossing run it's compared
  against** (4 producers here vs. 8 for non-crossing "Run 2"), because
  this mirrors `TestEngineLoad_Matching_4Producers_30Seconds`'s producer
  count rather than the non-crossing test's. Since `SubmitOrder` is
  synchronous and throughput is bounded by the single engine goroutine
  rather than producer count once enough producers are saturating it,
  this is unlikely to explain the near-parity result above — but it
  hasn't been controlled for directly (i.e., no run exists yet at matched
  producer counts).
- The "likely explanation" above is inference from the aggregate
  orders/sec number, not a direct measurement of WAL writes per order.

## Open question: is this throughput acceptable?

This document reports what the engine *can* sustain per symbol with the
WAL as currently implemented — it does not answer whether that is
*enough*. That depends on the peak per-symbol order rate this system
needs to support, which has not yet been specified anywhere in this repo.
Establishing that target is a prerequisite for deciding whether the
WAL write path needs to change at all.

## Candidate mitigation (not yet implemented, not yet scoped)

If a throughput in the ~5,000–7,000 orders/sec/symbol range (both
workload shapes measured so far) turns out to be insufficient, the
standard mitigation for exactly this shape of bottleneck is **group
commit**: batch several pending WAL writes and issue a single fsync for
the batch, rather than one fsync per command (the same technique
Postgres's WAL and Kafka's log segments both use). This trades a small
amount of added per-order latency (time spent waiting to batch) for a
large multiple in throughput. This has deliberately not been designed or
implemented as part of this benchmark — it should be scoped against an
actual target throughput and latency budget, not built speculatively.

## Methodology for future benchmark runs

To reproduce or extend these results:

```
go test ./test/stress/... -run TestStress_SustainedThroughput -v
go test ./test/stress/... -run TestStress_SustainedMatchingThroughput -v
```

When adding new benchmark results to this file, always report:

- the exact workload (order type, crossing vs. non-crossing, producer count)
- WAL configuration (attached or not; disk type if known)
- duration and raw submitted/error counts, not just the derived rate
- the machine/environment the numbers came from

so that future readers can judge how much weight to put on any given
number, the same way the caveats above apply to this one.