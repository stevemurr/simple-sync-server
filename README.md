# Simple Sync Server

A lightweight JSON-based sync server for notes and documents. Provides a REST API with last-write-wins conflict resolution for syncing data between multiple clients.

## Features

- REST API for CRUD operations on notes
- Two-way sync with last-write-wins conflict resolution
- Incremental sync (fetch only changes since last sync)
- JSON file storage (easy to backup and inspect)
- Docker support for easy deployment
- Health check endpoint for container orchestration

## Quick Start

### Local Development

```bash
# Clone the repo
git clone https://github.com/stevemurr/simple-sync-server.git
cd simple-sync-server

# Run with the helper script (creates venv automatically)
./run.sh
```

The server will start on `http://localhost:8080`.

### Docker

```bash
# Build
docker build -t simple-sync-server .

# Run
docker run -p 8080:8080 -v $(pwd)/data:/app/data simple-sync-server
```

### Docker Compose

```yaml
version: '3.8'
services:
  sync-server:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - ALLOWED_ORIGINS=https://myapp.com
    restart: unless-stopped
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | Server status |
| GET | `/health` | Health check |
| GET | `/notes` | Get all notes |
| GET | `/notes/{key}` | Get a specific note |
| PUT | `/notes/{key}` | Create or update a note |
| DELETE | `/notes/{key}` | Delete a note |
| POST | `/sync` | Two-way sync |
| GET | `/notes/since/{timestamp}` | Get notes updated since timestamp |

## Data Model

```json
{
  "dateKey": "2024-01-15",
  "content": "Note content here",
  "updatedAt": "2024-01-15T10:30:00Z",
  "chatMessages": [],
  "conversationStarted": false
}
```

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

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST` | `0.0.0.0` | Server bind address |
| `PORT` | `8080` | Server port |
| `DATA_DIR` | `./data` | Directory for JSON storage |
| `ALLOWED_ORIGINS` | `*` | Comma-separated list of allowed CORS origins |

## License

MIT
