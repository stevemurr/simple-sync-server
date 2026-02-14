package store

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// SqliteStore stores all collections in a single SQLite database.
//
// Tables:
//
//	documents(collection, key, data)  PRIMARY KEY (collection, key)
//	schemas(collection, schema)       PRIMARY KEY (collection)
type SqliteStore struct {
	mu sync.RWMutex
	db *sql.DB
}

func NewSqliteStore(dbPath string) (*SqliteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS documents (
		collection TEXT NOT NULL,
		key TEXT NOT NULL,
		data TEXT NOT NULL,
		PRIMARY KEY (collection, key)
	)`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schemas (
		collection TEXT PRIMARY KEY,
		schema TEXT NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	return &SqliteStore{db: db}, nil
}

func (s *SqliteStore) Close() error {
	return s.db.Close()
}

func (s *SqliteStore) GetAll(collection string) (map[string]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT key, data FROM documents WHERE collection = ?", collection)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]map[string]any)
	for rows.Next() {
		var key, raw string
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			continue
		}
		result[key] = doc
	}
	return result, rows.Err()
}

func (s *SqliteStore) Get(collection, key string) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var raw string
	err := s.db.QueryRow(
		"SELECT data FROM documents WHERE collection = ? AND key = ?",
		collection, key,
	).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *SqliteStore) Put(collection, key string, data map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO documents (collection, key, data) VALUES (?, ?, ?)
		 ON CONFLICT(collection, key) DO UPDATE SET data = excluded.data`,
		collection, key, string(b),
	)
	return err
}

func (s *SqliteStore) Delete(collection, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(
		"DELETE FROM documents WHERE collection = ? AND key = ?",
		collection, key,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SqliteStore) ListCollections() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT DISTINCT collection FROM documents ORDER BY collection")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (s *SqliteStore) GetSchema(collection string) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var raw string
	err := s.db.QueryRow("SELECT schema FROM schemas WHERE collection = ?", collection).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		return nil, err
	}
	return schema, nil
}

func (s *SqliteStore) PutSchema(collection string, schema map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO schemas (collection, schema) VALUES (?, ?)
		 ON CONFLICT(collection) DO UPDATE SET schema = excluded.schema`,
		collection, string(b),
	)
	return err
}

func (s *SqliteStore) DeleteSchema(collection string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec("DELETE FROM schemas WHERE collection = ?", collection)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SqliteStore) ListSchemas() (map[string]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT collection, schema FROM schemas")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]map[string]any)
	for rows.Next() {
		var name, raw string
		if err := rows.Scan(&name, &raw); err != nil {
			return nil, err
		}
		var schema map[string]any
		if err := json.Unmarshal([]byte(raw), &schema); err != nil {
			continue
		}
		result[name] = schema
	}
	return result, rows.Err()
}
