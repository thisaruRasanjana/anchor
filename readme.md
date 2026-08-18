
# KV Store
 
A concurrent TCP-based key-value store written in Go, featuring TTL expiration, durable Write-Ahead Logging, crash recovery, and graceful shutdown. The goal was to explore practical systems engineering concepts by building a small database from first principles. Deployment and infrastructure components are being developed incrementally on top of the tested database core.
 
**Go · TCP · Concurrency · WAL · TTL · Crash Recovery**
 
## Architecture
 
```text
                         TCP Clients
                              │
                              ▼
                     ┌─────────────────┐
                     │  Go TCP Server  │
                     └────────┬────────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │    Database     │
                     │                 │
                     │ Mutation        │
                     │ Ordering        │
                     └───────┬─────────┘
                             │
                   ┌─────────┴─────────┐
                   ▼                   ▼
            ┌─────────────┐     ┌─────────────┐
            │    Store    │     │     WAL     │
            │             │     │             │
            │  In-memory  │     │   Durable   │
            │  TTL        │     │   Replay    │
            │  Locking    │     │   fsync     │
            └─────────────┘     └─────────────┘
```
 
Mutations are persisted to the WAL and synchronized to disk before the in-memory state is updated. See [Design Decisions](#design-decisions) for why.
 
## Features
 
- TCP client/server communication
- In-memory key-value storage
- `SET`, `GET`, and `DELETE` operations
- TTL-based key expiration
- Background expiration worker
- Write-Ahead Logging (WAL)
- WAL durability using `fsync`
- Crash recovery through WAL replay
- Concurrent client handling
- Thread-safe data access
- Graceful shutdown on `SIGINT` / `SIGTERM`
- Automated unit, integration, concurrency, and recovery tests
- Go race detector validation
- WAL validation and incomplete-record testing
## Protocol
 
Commands are sent as newline-delimited plain text over TCP, one command per line. The server responds with a single line: the requested value, `OK` for a successful mutation, `NOT FOUND` for a missing key, or `ERROR <reason>` for a malformed or unrecognized command.
 
## Commands
 
| Command | Description |
|---|---|
| `SET key value` | Store a value |
| `GET key` | Retrieve a value |
| `DELETE key` | Remove a key |
| `SETTTL key value seconds` | Store a value with expiration |
 
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
 
DELETE name
OK
 
GET name
NOT FOUND
```
 
Keys and values currently cannot contain whitespace.
 
## Persistence & Recovery
 
The active database state is kept in memory while mutations are persisted through a Write-Ahead Log before the store is updated:
 
```text
SET / DELETE / SETTTL
          │
          ▼
         WAL
          │
        fsync
          │
          ▼
   In-memory Store
```
 
On startup, the WAL is replayed to reconstruct the database:
 
```text
store.wal
    │
    ▼
 WAL Replay
    │
    ▼
Recovered Store
```
 
For example:
 
```text
SET name Thisaru
SET age 22
DELETE age
```
 
recovers to:
 
```text
name = Thisaru
age  = NOT FOUND
```
 
TTL records are also restored with their original expiration timestamps. Records that have already expired are not restored.
 
## Concurrency
 
Each TCP connection is handled in its own goroutine.
 
```text
Client 1 ──┐
Client 2 ──┤
Client 3 ──┼──► TCP Server ──► Database
Client 4 ──┤
Client 5 ──┘
```
 
Synchronization is handled at multiple layers:
 
- **Store locking** protects the in-memory map.
- **Database locking** serializes durable mutations and preserves WAL/store ordering.
- **WAL locking** protects concurrent file operations.
## Testing
 
The project includes unit, integration, concurrency, persistence, recovery, and failure-oriented tests.
 
Run the complete test suite:
 
```bash
go test ./...
```
 
Run with the race detector:
 
```bash
go test -race ./...
```
 
Run a fresh verbose race-enabled test run:
 
```bash
go test -race -count=1 -v ./...
```
 
Run static analysis:
 
```bash
go vet ./...
```
 
Format the project:
 
```bash
gofmt -w *.go
```
 
The test suite covers:
 
- Store operations
- TTL expiration
- Concurrent Store access
- Database persistence
- WAL replay
- Concurrent WAL appends
- Concurrent TCP clients
- Mixed concurrent operations
- Concurrent workload recovery
- Valid and expired TTL recovery
- Malformed WAL records
- Incomplete final WAL records
- Invalid WAL expiration timestamps
- Unknown WAL commands
- Graceful server shutdown
## Quick Start
 
**Prerequisites:** Go 1.21+ (match this to your `go.mod`)
 
Clone the repository:
 
```bash
git clone https://github.com/thisaruRasanjana/kvstore.git
cd kvstore
```
 
Start the server:
 
```bash
go run .
```
 
The server listens on:
 
```text
:7379
```
 
You should see:
 
```text
Recovery complete
Server listening on :7379
```
 
Connect using `netcat`:
 
```bash
nc localhost 7379
```
 
Then run commands such as:
 
```text
SET name Thisaru
GET name
SETTTL token abc123 30
GET token
DELETE name
```
 
## Graceful Shutdown
 
The server handles `SIGINT` and `SIGTERM`.
 
Pressing `Ctrl+C` triggers:
 
```text
Shutting down server...
Server stopped.
```
 
During shutdown:
 
1. The server stops accepting new connections.
2. Active connections are closed.
3. Connection handlers are waited on.
4. The WAL is closed.
5. The process exits.
## Project Structure
 
```text
kvstore/
│
├── main.go
├── server.go
├── database.go
├── store.go
├── wal.go
│
├── server_test.go
├── database_test.go
├── store_test.go
├── wal_test.go
│
├── go.mod
└── README.md
```
 
`store.wal` is runtime persistence data and should not be committed to the repository.
 
## Design Decisions
 
### In-memory storage
 
An in-memory map provides simple and fast key-value access while allowing the project to focus on concurrency, persistence, and recovery.
 
### Write-Ahead Logging
 
The WAL provides a durable sequence of mutations that can be replayed after a process restart.
 
### `fsync` per mutation
 
Each mutation is appended to the WAL and synchronized to disk before the in-memory state is updated, prioritizing durability over maximum write throughput.
 
### Serialized mutations
 
A mutation involves both durable persistence and an in-memory update. Database-level locking ensures these operations maintain a consistent ordering.
 
## Current Limitations
 
This is intentionally a small systems-oriented database implementation rather than a production database.
 
Current limitations include:
 
- Single-process architecture
- Single WAL file
- No replication or clustering
- No authentication
- No encryption
- No multi-command transactions
- Fixed TCP port
- Keys and values cannot contain whitespace
- No WAL compaction or snapshots
- No command batching
- No metrics or observability system