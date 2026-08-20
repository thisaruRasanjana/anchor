# Anchor

Anchor is a small, concurrent, persistent key-value store written in Go, exposing a simple TCP protocol for storing and retrieving string key-value pairs. It was built to explore the engineering trade-offs behind a persistent storage system — concurrency, write-ahead logging, crash recovery, TTL expiration, network protocols, and containerized deployment — while keeping the storage model itself simple enough that each mechanism can be implemented and tested directly.

## Features

- TCP server on port `7379`
- `SET`, `GET`, `DELETE`, and `SETTTL` commands
- In-memory key-value storage
- Thread-safe concurrent access
- TTL-based key expiration
- Background expiration worker
- Write-ahead logging with explicit file synchronization
- State recovery by replaying the WAL
- Handling of incomplete final WAL records
- Graceful shutdown on `SIGINT` / `SIGTERM`
- Concurrent client handling
- Race-detector tested
- Docker multi-stage build
- `linux/amd64` and `linux/arm64` container images
- Persistent Docker volume support
- Docker Compose deployment
- GitHub Actions CI
- Docker images published to GitHub Container Registry

## Quick Start

### Docker Compose

```bash
docker compose up -d
```

Then connect:

```bash
nc localhost 7379
```

Try:

```text
SET name Thisaru
GET name
SETTTL token abc123 10
GET token
```

Stop the service:

```bash
docker compose down
```

See the [Docker](#docker) section for what's happening under the hood, and [Run Locally](#run-locally) if you'd rather run it without containers.

## Architecture

```text
                         TCP Clients
                              │
                              ▼
                    ┌──────────────────┐
                    │    TCP Server    │
                    │     :7379        │
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │    Database      │
                    │                  │
                    │ WAL + Store      │
                    └───────┬───┬──────┘
                            │   │
                 ┌──────────┘   └──────────┐
                 ▼                         ▼
          ┌──────────────┐          ┌──────────────┐
          │ In-Memory    │          │     WAL      │
          │ Store        │          │   store.wal  │
          └──────────────┘          └──────┬───────┘
                                           │
                                           ▼
                                   Persistent Volume
```

For mutations, the server passes the request to the database layer. The database appends the operation to the WAL, synchronizes it to disk using `os.File.Sync()`, and then updates the in-memory store. See [Design Decisions](#design-decisions) for why this ordering matters.

On startup, the WAL is replayed to reconstruct the in-memory state. Expired TTL records are not restored.

## Protocol

Commands are sent as newline-delimited plain text over TCP, one command per line. The server responds with a single line containing the requested value, `OK`, `NOT FOUND`, or an `ERROR:` message (for example, `ERROR: usage: SET <key> <value>` or `ERROR: unknown command`).

## Commands

### SET

```text
SET <key> <value>
```

Example:

```text
SET name Thisaru
```

Response:

```text
OK
```

### GET

```text
GET <key>
```

If the key does not exist:

```text
NOT FOUND
```

### DELETE

```text
DELETE <key>
```

### SETTTL

```text
SETTTL <key> <value> <ttl>
```

The TTL is specified in seconds.

Example:

```text
SETTTL token abc123 10
```

After expiration:

```text
GET token
```

returns:

```text
NOT FOUND
```

Keys and values currently cannot contain whitespace.

## Run Locally

Requirements:

- Go 1.26.1 or newer

Run:

```bash
go run .
```

The server listens on `:7379`.

Connect with:

```bash
nc localhost 7379
```

Example:

```text
SET name Thisaru
OK
GET name
Thisaru
SETTTL token abc123 10
OK
GET token
abc123
```

The WAL is stored in `store.wal`.

## Testing

Run the tests:

```bash
go test -count=1 ./...
```

Run with the race detector:

```bash
go test -race -count=1 ./...
```

Run static analysis:

```bash
go vet ./...
```

Check formatting:

```bash
gofmt -l .
```

Tests cover database operations, TTL behavior, persistence and recovery, concurrent workloads, TCP commands, invalid input, concurrent clients, WAL replay, malformed records, incomplete final records, and concurrent WAL access.

## Validation

The project is validated with:

- Unit and integration tests
- Concurrent workload tests
- TCP client concurrency tests
- WAL recovery tests
- Malformed and incomplete WAL tests
- Go race detector
- `go vet`
- Automated GitHub Actions CI
- Multi-platform Docker builds

## Docker

### Local Build

Build locally:

```bash
docker build -t anchor .
```

Run:

```bash
docker run --name anchor \
  -p 7379:7379 \
  -v anchor-data:/app/data \
  anchor
```

The image uses a multi-stage build with a Go builder and a minimal Alpine runtime image.

### Docker Compose

Start:

```bash
docker compose up -d
```

Check status:

```bash
docker compose ps
```

View logs:

```bash
docker compose logs
```

Connect:

```bash
nc localhost 7379
```

Stop:

```bash
docker compose down
```

The Compose configuration uses a named volume for persistent data. Removing the container with `docker compose down` does not remove that volume.

To remove the stored data as well:

```bash
docker compose down -v
```

### Prebuilt Image (GHCR)

The development image is published to GitHub Container Registry:

```bash
docker pull ghcr.io/thisarurasanjana/anchor:dev
```

Run it with persistent storage:

```bash
docker run --name anchor \
  -p 7379:7379 \
  -v anchor-data:/app/data \
  ghcr.io/thisarurasanjana/anchor:dev
```

Published images support:

```text
linux/amd64
linux/arm64
```

Docker automatically selects the appropriate platform.

## CI/CD

GitHub Actions runs on pushes and pull requests targeting `main` or `dev`.

The validation pipeline performs:

```text
Checkout
   │
   ▼
Go setup
   │
   ├── gofmt validation
   ├── go vet
   ├── go test
   └── go test -race
           │
           ▼
       Docker Build
           │
           ▼
          GHCR
```

The Docker job runs only after the test job succeeds. Docker images are built for `linux/amd64` and `linux/arm64` and published to GitHub Container Registry.

## Persistence and Recovery

Anchor uses a write-ahead log to provide durable state.

Example WAL records:

```text
SET name Thisaru
DELETE name
SET_EXPIRED token abc123 2026-...
```

Each successful mutation is recorded in the WAL and synchronized to disk using `os.File.Sync()` before the in-memory store is changed.

At startup, the WAL is replayed to reconstruct the store. Expired TTL records are skipped.

The recovery logic also handles an incomplete final WAL record, which can occur if a process terminates while a record is being written.

## Concurrency

Each TCP connection is handled in its own goroutine.

Synchronization is handled at multiple layers:

- **Store locking** protects the in-memory map from concurrent reads/writes.
- **Database locking** serializes WAL-backed mutations so that the WAL write and in-memory update occur in a controlled order.
- **WAL locking** protects concurrent file operations against the log.

The race detector is part of the project's validation process, both locally and in CI:

```bash
go test -race -count=1 ./...
```

## Project Structure

```text
.
├── database.go
├── database_test.go
├── main.go
├── server.go
├── server_test.go
├── store.go
├── store_test.go
├── wal.go
├── wal_test.go
├── Dockerfile
├── .dockerignore
├── compose.yaml
├── go.mod
└── .github/
    └── workflows/
        └── ci.yml
```

`store.wal` is runtime persistence data and should not be committed to the repository.

## Design Decisions

### Write-ahead logging with explicit file synchronization

Each mutation is appended to the WAL and synchronized using `os.File.Sync()` before the in-memory store is updated, prioritizing durability over maximum write throughput.

### Serialized mutations

A mutation involves both durable persistence and an in-memory update. Database-level locking ensures these operations maintain a consistent ordering.

### Multi-stage, multi-arch Docker build

A Go builder stage compiles the binary, which is then copied into a minimal Alpine runtime image for a small final image size. Images are built for both `linux/amd64` and `linux/arm64`.

## Current Limitations

This is intentionally a small systems-oriented database rather than a production database.

Current limitations include:

- Single-process architecture
- Single WAL file, no compaction or snapshots
- No replication or clustering
- No authentication
- No encryption
- No multi-command transactions
- Fixed TCP port
- Keys and values cannot contain whitespace
- No command batching
- No metrics or observability system

## License

This project is for educational and portfolio purposes.