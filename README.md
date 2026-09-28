# Velocity

Velocity is a high-performance order matching and trading engine written in Go. It implements a price-time-priority matching engine with one dedicated engine instance per trading symbol, backed by a write-ahead log for durability, periodic snapshots for fast recovery, and an event-driven pipeline that settles trades, updates wallets and positions, and streams live market data over REST and WebSocket.

It is built as one service in a small microservice topology: Velocity owns trading, orders, the order book, wallets, and positions, while user authentication and identity are delegated to a separate Identity Service over gRPC.

## Core Features

- **Price-time-priority matching engine** — one engine goroutine per symbol, each with its own single-writer command queue, so order submission, cancellation, and modification for a symbol are always processed in strict sequence with no locking around the hot path.
- **Order types** — `LIMIT`, `MARKET`, `STOP_MARKET`, and `STOP_LIMIT`, with `GTC`, `IOC`, `FOK`, and `POST_ONLY` time-in-force handling.
- **Order book** — bids and asks are kept in heap-ordered price levels (max-heap for bids, min-heap for asks), with each price level holding a FIFO queue of resting orders for time priority.
- **Stop orders** — a separate stop book holds pending stop orders and triggers them into the matching engine once the trigger price is crossed.
- **Durability** — every accepted command is written to a per-symbol write-ahead log (WAL) before being applied, and periodic snapshots of engine state let the engine recover quickly after a restart without replaying the entire log history.
- **Event-driven settlement** — trades produced by the engine are published as events, consumed asynchronously by a worker process, and settled transactionally against wallets, positions, and order state in PostgreSQL. Failed settlements are retried by a background worker with a dead-letter path after repeated failures.
- **Risk validation** — pluggable validators (balance, quantity, price) run before an order is accepted by the engine.
- **Market data** — live order book depth, tickers, recent trades, market stats, and OHLC candles, computed from the trade/event stream and served over REST and WebSocket.
- **Wallets & positions** — per-user, per-asset wallet balances (deposit/withdraw) and derived trading positions per symbol.
- **Public & private WebSocket streams** — a public feed for market data and a per-user authenticated feed for order/trade/wallet updates.
- **Rate limiting** — configurable token-bucket limits for order submission, cancellation, and modification.
- **Observability** — structured logging (zap), Prometheus metrics, and optional distributed tracing.

## Architecture

Velocity separates the hot matching path from everything else:

```
                     ┌─────────────────────────┐
   HTTP / gRPC  ───▶ │   Risk validation        │
   (order intake)    └────────────┬─────────────┘
                                   ▼
                     ┌─────────────────────────┐
                     │  Per-symbol Engine        │
                     │  (single command queue)   │
                     │                            │
                     │  Matcher ──▶ Order Book    │
                     │     │        (heaps + FIFO │
                     │     ▼         price levels)│
                     │  WAL Writer  Stop Book     │
                     │  Snapshotter               │
                     └────────────┬─────────────┘
                                   │ trade / order events
                                   ▼
                     ┌─────────────────────────┐
                     │  Event bus / Kafka        │
                     └────────────┬─────────────┘
                                   ▼
                     ┌─────────────────────────┐
                     │  Worker process           │
                     │  Settlement service        │
                     │  (Postgres, transactional)  │
                     │  Wallets · Positions ·     │
                     │  Order state · Retries     │
                     └─────────────────────────┘
```

Key architectural decisions:

- **One engine per symbol** — each symbol gets its own goroutine, order book, stop book, WAL, and sequence counter. Symbols never contend with each other, and a lazily-created registry looks engines up (or spins up a new one) by symbol.
- **Single-writer command queue** — all mutating operations for a symbol (submit, cancel, modify) go through one buffered channel consumed by one goroutine, which removes the need for locking inside the matching loop.
- **WAL before match** — an accepted command is durably written to the write-ahead log before the matcher runs, so engine state can always be reconstructed after a crash.
- **Snapshots for fast recovery** — the engine periodically snapshots its order book so that recovery only has to replay the WAL entries written after the last snapshot, not the full history.
- **Settlement is decoupled from matching** — the engine only decides *what* trades happened; a separate worker process consumes those trade events off the queue and does the actual balance/position bookkeeping in Postgres transactions, so a slow database never blocks the matching engine.
- **Auth is delegated** — Velocity does not issue or manage credentials. Its HTTP auth middleware validates bearer tokens by calling out to an external Identity Service over gRPC (`AuthService.ValidateToken`), and exposes its own gRPC endpoint (`VelocityService.CreateUser`) so the Identity Service can provision a corresponding user record when someone registers.

## Project Layout

```
cmd/
  api/        entry point: gRPC + HTTP servers (order intake, market data, wallets, positions, admin)
  worker/     entry point: Kafka consumer that runs settlement and analytics
  migrate/    database migration CLI
  seed/       development data seeder

internal/
  engine/           the matching engine itself
    orderbook/      heap-based bid/ask book
    pricelevel/     FIFO price-level queues
    matcher/        matching algorithm (limit/market/stop, TIF handling)
    stopbook/       pending stop-order book
    wal/            write-ahead log writer/reader
    snapshot/       engine state snapshotting and loading
    recovery/       WAL + snapshot replay on startup
    registry/       per-symbol engine lifecycle
    command/        commands accepted by the engine (submit/cancel/modify)
    events/         engine-emitted domain events
  domain/           core types: order, trade, position, risk, depth, market data
  service/          application services: order, risk, settlement, wallet, position,
                     trade, market, user
  transport/
    http/           Fiber-based REST API (handlers, routers, middleware, DTOs)
    ws/             public WebSocket feed
    userws/         authenticated per-user WebSocket feed
    grpc/           gRPC server + identity-service client
  persistence/      Postgres repositories, transactions, retry/DLQ workers
  infrastructure/   Kafka, Redis, metrics, logging, tracing, security (JWT, password hashing)
  analytics/        candle (OHLC) and rolling-stats computation from the trade stream
  eventbus/         in-process pub/sub used to fan out engine events

pkg/                shared utilities: ID generation (Snowflake), errors, validation,
                     time helpers, response envelopes

migrations/         SQL schema migrations (users, symbols, orders, trades, positions,
                     wallets, position history, failed settlements)
seed/                data seeders for local development (users, symbols, wallets, positions)
configs/            environment configuration (development / staging / production)
deployments/        Dockerfiles, docker-compose, Kubernetes manifests, CI workflows
test/                unit, integration, load, stress, and chaos tests; benchmarks
```

## API Surface

All authenticated routes require `Authorization: Bearer <token>`, validated against the Identity Service.

### Orders (`/api/orders`) — auth required
| Method | Path | Description |
|---|---|---|
| POST | `/api/orders` | Submit a new order (rate-limited) |
| GET | `/api/orders/open` | List the caller's open orders |
| GET | `/api/orders/history` | List the caller's order history |
| GET | `/api/orders/:id` | Get a single order by ID |
| PATCH | `/api/orders/:id` | Modify an open order (rate-limited) |
| DELETE | `/api/orders/:id` | Cancel an order (rate-limited) |

### Wallets (`/api/wallets`) — auth required
| Method | Path | Description |
|---|---|---|
| GET | `/api/wallets/` | List the caller's wallet balances |
| GET | `/api/wallets/:asset` | Get balance for a specific asset |
| POST | `/api/wallets/deposit` | Deposit funds |
| POST | `/api/wallets/withdraw` | Withdraw funds |

### Positions (`/api/positions`) — auth required
| Method | Path | Description |
|---|---|---|
| GET | `/api/positions/` | List the caller's positions |
| GET | `/api/positions/:symbol` | Get position for a specific symbol |

### Market Data (`/api/market`) — public
| Method | Path | Description |
|---|---|---|
| GET | `/api/market/symbols` | List tradable symbols |
| GET | `/api/market/orderbook/:symbol` | Current order book depth |
| GET | `/api/market/ticker/:symbol` | Latest ticker |
| GET | `/api/market/trades/:symbol` | Recent public trades |
| GET | `/api/market/stats/:symbol` | Market statistics |
| GET | `/api/market/:symbol/candles` | OHLC candles |
| GET | `/api/market/trades/user` | Caller's own trade history (auth required) |

### Admin (`/api/admin`) — auth + admin role required
| Method | Path | Description |
|---|---|---|
| POST | `/api/admin/symbols` | Create a new trading symbol |
| PATCH | `/api/admin/symbols/:symbol` | Update a symbol's status |

### Health
| Method | Path | Description |
|---|---|---|
| GET | `/health/` | Liveness probe |
| GET | `/health/ready` | Readiness probe |

### WebSocket
| Path | Description |
|---|---|
| `/ws` | Public feed: order book, tickers, trades |
| `/ws/private` | Authenticated per-user feed: order updates, fills, wallet changes |

### gRPC
- `VelocityService.CreateUser` — provisions a Velocity-side user record (called by the Identity Service on registration).
- Velocity, in turn, calls the Identity Service's `AuthService.ValidateToken` to authenticate incoming HTTP/WebSocket requests.

## Getting Started

### Prerequisites
- Go 1.25+
- PostgreSQL
- Apache Kafka
- Redis (optional, used for caching/rate-limit state)
- `golang-migrate` CLI (for running migrations outside of `make`)

### Configuration

Configuration is loaded via Viper from `configs/config.<environment>.yaml` (`development`, `staging`, `production`), covering the app, HTTP server, database, logger, engine tuning (queue size, worker count, snapshot interval), JWT, WebSocket, metrics, Redis, Kafka, tracing, and rate-limit sections.

Before running locally, update the `database` section (and any other secrets) in `configs/config.development.yaml` rather than relying on the checked-in defaults — the shipped values are placeholders for local development only and should never be reused for anything beyond a throwaway local database.

### Running infrastructure

A docker-compose file is provided for Kafka (KRaft mode, no ZooKeeper):

```bash
docker compose -f deployments/compose/docker-compose.yml up -d
```

You'll additionally need a PostgreSQL instance reachable with the credentials in your config file (and Redis, if enabled).

### Database setup

`cmd/migrate` reads its connection string from `DATABASE_URL`. If unset, it
falls back to a hardcoded local dev DSN matching the placeholder credentials
in `configs/config.development.yaml` — set `DATABASE_URL` explicitly for
anything beyond a throwaway local database.

```bash
export DATABASE_URL="postgres://postgres:Shinas@localhost:5432/velocity?sslmode=disable"

make migrate-up      # apply all migrations
make migrate-down     # roll back one migration
make migrate-reset    # drop everything and reapply from scratch
go run ./cmd/seed     # (optional) seed development data — users, symbols, wallets, positions
```

### Running the services

```bash
make run              # runs cmd/api — HTTP + gRPC servers
go run ./cmd/worker    # Kafka consumer: settlement + analytics pipeline
```

The API server listens on the HTTP port from your config (default `8080`) and starts a gRPC server alongside it. Prometheus metrics are exposed on the configured metrics port/path (default `:9090/metrics`).

### Running tests

```bash
make test             # full test suite
make engine-test       # engine package tests only
make matcher-test      # matcher package tests only
make bench             # benchmarks
```

Integration, load, stress, and chaos tests live under `test/` alongside unit tests and shared fixtures/mocks.

## Deployment

Dockerfiles are provided per service (`deployments/docker/api.Dockerfile`, `worker.Dockerfile`), along with base and overlay Kubernetes manifests under `deployments/kubernetes/` and a GitHub Actions CI workflow under `deployments/ci/github-actions/`.

## Notes on Persisted Data

Running the engine locally will create per-symbol `.wal` files under `wal/` and periodic snapshots under `snapshots/`. These are runtime artifacts of the matching engine's durability mechanism, not source files — treat them as disposable in a development environment (delete them to start the engine from a clean state) and back them up appropriately in any environment where order-book state must survive a restart.