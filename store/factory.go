package store

import (
	"fmt"
	"path/filepath"
)

// New creates a Store based on the backend name.
//
// Supported backends:
//
//	"json"   - JSON files in dataDir (default)
//	"sqlite" - SQLite database at dataDir/sync.db
//	"memory" - In-memory (ephemeral, for testing)
func New(backend, dataDir string) (Store, error) {
	switch backend {
	case "json", "":
		return NewJsonFileStore(dataDir)
	case "sqlite":
		dbPath := filepath.Join(dataDir, "sync.db")
		return NewSqliteStore(dbPath)
	case "memory":
		return NewMemoryStore(), nil
	default:
		return nil, fmt.Errorf("unknown store backend: %q (supported: json, sqlite, memory)", backend)
	}
}
