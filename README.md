# Simple Sync Server

A lightweight sync server with pluggable backing stores and optional JSON Schema validation. Provides a REST API with last-write-wins conflict resolution for syncing data between multiple clients.

Rewritten in Go for better performance and a single-binary deployment.

## Features

- **Pluggable backing stores**: JSON files (default), SQLite, or in-memory
- **Schema validation**: Define JSON Schemas per collection; documents are validated on write
- **Generic collections**: Store any type of data in named collections
- **Backward-compatible**: Original `/notes` endpoints still work unchanged
- **Two-way sync** with last-write-wins conflict resolution
- **Incremental sync** (fetch only changes since last sync)
- Docker support with multi-stage build

## Quick Start

### Local Development

```bash
# Build and run
go build -o sync-server .
./sync-server

# Or run directly
go run .
```

The server starts on `http://localhost:8080`.

### Docker

```bash
docker build -t simple-sync-server .
docker run -p 8080:8080 -v $(pwd)/data:/app/data simple-sync-server
```

### Docker Compose

```bash
docker compose up
```

## Backing Stores

Set `STORE_BACKEND` to choose a storage backend:

| Backend | Value | Description |
|---------|-------|-------------|
| JSON files | `json` (default) | One `.json` file per collection in `DATA_DIR` |
| SQLite | `sqlite` | Single `sync.db` database in `DATA_DIR` |
| In-memory | `memory` | Ephemeral, data lost on restart (useful for testing) |

```bash
# Use SQLite
STORE_BACKEND=sqlite ./sync-server

# Use in-memory
STORE_BACKEND=memory ./sync-server
```

## API Endpoints

### Status

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | Server status |
| GET | `/health` | Health check |

### Notes (backward-compatible)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/notes` | Get all notes |
| GET | `/notes/{key}` | Get a specific note |
| PUT | `/notes/{key}` | Create or update a note |
| DELETE | `/notes/{key}` | Delete a note |
| POST | `/sync` | Two-way sync for notes |
| GET | `/notes/since/{timestamp}` | Get notes updated since timestamp |

### Collections (generic)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/collections` | List all collections |
| GET | `/collections/{name}/items` | Get all items in a collection |
| GET | `/collections/{name}/items/{key}` | Get a specific item |
| PUT | `/collections/{name}/items/{key}` | Create or update an item |
| DELETE | `/collections/{name}/items/{key}` | Delete an item |
| POST | `/collections/{name}/sync` | Two-way sync for a collection |
| GET | `/collections/{name}/items/since/{ts}` | Items updated since timestamp |

### Schemas

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/schemas` | List all schemas |
| GET | `/schemas/{collection}` | Get schema for a collection |
| PUT | `/schemas/{collection}` | Set schema for a collection |
| DELETE | `/schemas/{collection}` | Remove schema for a collection |

## Schemas

Define a JSON Schema for a collection to validate documents on write. Documents that fail validation are rejected with `422 Unprocessable Entity`.

### Define a schema

```bash
curl -X PUT http://localhost:8080/schemas/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "object",
    "properties": {
      "title": {"type": "string", "minLength": 1},
      "done": {"type": "boolean"},
      "priority": {"type": "integer", "minimum": 1, "maximum": 5}
    },
    "required": ["title"]
  }'
```

### Write a valid document

```bash
curl -X PUT http://localhost:8080/collections/tasks/items/t1 \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy milk", "done": false, "priority": 2, "updatedAt": "2024-06-01T12:00:00Z"}'
```

### Write an invalid document (rejected)

```bash
curl -X PUT http://localhost:8080/collections/tasks/items/t2 \
  -H "Content-Type: application/json" \
  -d '{"done": true, "updatedAt": "2024-06-01T12:00:00Z"}'
# 422: missing required field "title"
```

### Supported JSON Schema keywords

- `type` (string, number, integer, boolean, object, array, null)
- `properties`, `required`, `additionalProperties`
- `items` (array item validation)
- `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`
- `minLength`, `maxLength`
- `minItems`, `maxItems`
- `enum`

## Sync Protocol

### Full Sync (First Time)

```bash
curl -X POST http://localhost:8080/sync \
  -H "Content-Type: application/json" \
  -d '{"notes": [], "lastSyncTime": null}'
```

### Incremental Sync

```bash
curl -X POST http://localhost:8080/sync \
  -H "Content-Type: application/json" \
  -d '{
    "notes": [{"dateKey": "2024-01-15", "content": "Updated", "updatedAt": "2024-01-15T10:30:00Z"}],
    "lastSyncTime": "2024-01-14T00:00:00Z"
  }'
```

### Collection Sync

```bash
curl -X POST http://localhost:8080/collections/tasks/sync \
  -H "Content-Type: application/json" \
  -d '{
    "items": [{"id": "t1", "title": "Buy milk", "updatedAt": "2024-06-01T12:00:00Z"}],
    "lastSyncTime": null
  }'
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST` | `0.0.0.0` | Server bind address |
| `PORT` | `8080` | Server port |
| `DATA_DIR` | `./data` | Directory for data storage |
| `STORE_BACKEND` | `json` | Storage backend: `json`, `sqlite`, or `memory` |
| `ALLOWED_ORIGINS` | `*` | Comma-separated list of allowed CORS origins |

## Testing

```bash
go test ./...
```

## License

MIT
