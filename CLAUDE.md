# CLAUDE.md

Project guide for AI assistants working on this codebase.

## What this is

A lightweight HTTP sync server written in Go. Supports named collections of JSON documents with last-write-wins conflict resolution, pluggable backing stores, and optional JSON Schema validation.

## Build & run

```bash
go build -o sync-server .   # build
go run .                     # run directly
go test ./...                # run all tests
```

CGO is required (SQLite driver uses `github.com/mattn/go-sqlite3`). Ensure `libsqlite3-dev` (or equivalent) is installed.

## Project structure

```
main.go           Entry point, CORS middleware, env var config
store/
  store.go        Store interface — the backing store abstraction
  factory.go      Creates a Store from STORE_BACKEND env var
  memory.go       In-memory store (ephemeral)
  json_file.go    JSON file store (one .json per collection)
  sqlite.go       SQLite store (single sync.db, WAL mode)
  store_test.go   Tests all three store implementations
schema/
  schema.go       JSON Schema validation (draft-07 subset)
  schema_test.go  Schema validation tests
handler/
  handler.go      All HTTP routes and request handling
  handler_test.go HTTP integration tests
```

## Architecture decisions

- **Store interface** (`store.Store`): all data access goes through this interface. To add a new backend, implement the interface and add a case in `store/factory.go`.
- **Collections**: the server is generic — any named collection works. The `/notes` endpoints are backward-compatible aliases that route to the `"notes"` collection.
- **Schemas**: optional per-collection. Stored via the Store interface alongside data. Validation runs on every write (PUT and sync). No schema = no validation.
- **Conflict resolution**: last-write-wins based on the `updatedAt` field (ISO 8601 timestamp comparison).
- **Sync key detection**: during sync, documents are keyed by the first non-empty field in `["dateKey", "key", "id"]`.
- **No external HTTP framework**: uses Go 1.22+ `http.ServeMux` with method-based routing (`"GET /path"`).

## Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `HOST` | `0.0.0.0` | Bind address |
| `PORT` | `8080` | Listen port |
| `DATA_DIR` | `./data` | Where files/DB are stored |
| `STORE_BACKEND` | `json` | `json`, `sqlite`, or `memory` |
| `ALLOWED_ORIGINS` | `*` | CORS origins |

## Key patterns

- All store implementations are safe for concurrent use (mutex-protected).
- `handler.Handler` takes a `store.Store` — easy to test with `MemoryStore`.
- Schema validation is a separate package (`schema/`) with no store dependency.
- HTTP tests use `httptest.NewServer` with a `MemoryStore` for fast, isolated tests.
