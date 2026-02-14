package store_test

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stevemurr/simple-sync-server/store"
)

// runStoreTests runs a common test suite against any Store implementation.
func runStoreTests(t *testing.T, s store.Store) {
	t.Helper()

	t.Run("GetAll empty", func(t *testing.T) {
		docs, err := s.GetAll("test")
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 0 {
			t.Fatalf("expected 0 docs, got %d", len(docs))
		}
	})

	t.Run("Put and Get", func(t *testing.T) {
		doc := map[string]any{"title": "hello", "count": float64(42)}
		if err := s.Put("col1", "k1", doc); err != nil {
			t.Fatal(err)
		}
		got, err := s.Get("col1", "k1")
		if err != nil {
			t.Fatal(err)
		}
		if got == nil {
			t.Fatal("expected doc, got nil")
		}
		if got["title"] != "hello" {
			t.Fatalf("expected title=hello, got %v", got["title"])
		}
		if got["count"] != float64(42) {
			t.Fatalf("expected count=42, got %v", got["count"])
		}
	})

	t.Run("Get missing", func(t *testing.T) {
		got, err := s.Get("col1", "missing")
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("Put overwrites", func(t *testing.T) {
		doc := map[string]any{"title": "updated"}
		if err := s.Put("col1", "k1", doc); err != nil {
			t.Fatal(err)
		}
		got, err := s.Get("col1", "k1")
		if err != nil {
			t.Fatal(err)
		}
		if got["title"] != "updated" {
			t.Fatalf("expected title=updated, got %v", got["title"])
		}
	})

	t.Run("GetAll returns all", func(t *testing.T) {
		if err := s.Put("col1", "k2", map[string]any{"title": "second"}); err != nil {
			t.Fatal(err)
		}
		docs, err := s.GetAll("col1")
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 2 {
			t.Fatalf("expected 2 docs, got %d", len(docs))
		}
	})

	t.Run("Delete existing", func(t *testing.T) {
		existed, err := s.Delete("col1", "k1")
		if err != nil {
			t.Fatal(err)
		}
		if !existed {
			t.Fatal("expected existed=true")
		}
		got, err := s.Get("col1", "k1")
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Fatal("expected nil after delete")
		}
	})

	t.Run("Delete missing", func(t *testing.T) {
		existed, err := s.Delete("col1", "nope")
		if err != nil {
			t.Fatal(err)
		}
		if existed {
			t.Fatal("expected existed=false")
		}
	})

	t.Run("ListCollections", func(t *testing.T) {
		names, err := s.ListCollections()
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, n := range names {
			if n == "col1" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected col1 in list, got %v", names)
		}
	})

	// Schema tests
	t.Run("GetSchema missing", func(t *testing.T) {
		sch, err := s.GetSchema("nope")
		if err != nil {
			t.Fatal(err)
		}
		if sch != nil {
			t.Fatalf("expected nil, got %v", sch)
		}
	})

	t.Run("PutSchema and GetSchema", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
			"required": []any{"name"},
		}
		if err := s.PutSchema("users", schema); err != nil {
			t.Fatal(err)
		}
		got, err := s.GetSchema("users")
		if err != nil {
			t.Fatal(err)
		}
		if got == nil {
			t.Fatal("expected schema, got nil")
		}
		if got["type"] != "object" {
			t.Fatalf("expected type=object, got %v", got["type"])
		}
	})

	t.Run("ListSchemas", func(t *testing.T) {
		schemas, err := s.ListSchemas()
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := schemas["users"]; !ok {
			t.Fatal("expected 'users' in schemas")
		}
	})

	t.Run("DeleteSchema", func(t *testing.T) {
		existed, err := s.DeleteSchema("users")
		if err != nil {
			t.Fatal(err)
		}
		if !existed {
			t.Fatal("expected existed=true")
		}
		got, err := s.GetSchema("users")
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Fatal("expected nil after delete")
		}
	})
}

func TestMemoryStore(t *testing.T) {
	s := store.NewMemoryStore()
	runStoreTests(t, s)
}

func TestJsonFileStore(t *testing.T) {
	dir := t.TempDir()
	s, err := store.NewJsonFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	runStoreTests(t, s)
}

func TestSqliteStore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.NewSqliteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	runStoreTests(t, s)
}

func TestFactory(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		backend string
	}{
		{"json"},
		{"sqlite"},
		{"memory"},
		{""},
	}
	for _, tc := range tests {
		t.Run(tc.backend, func(t *testing.T) {
			s, err := store.New(tc.backend, filepath.Join(dir, tc.backend))
			if err != nil {
				t.Fatal(err)
			}
			_ = s
		})
	}

	t.Run("unknown", func(t *testing.T) {
		_, err := store.New("redis", dir)
		if err == nil {
			t.Fatal("expected error for unknown backend")
		}
	})
}

func TestJsonFileStoreIsolation(t *testing.T) {
	dir := t.TempDir()
	s, err := store.NewJsonFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Put in two different collections
	s.Put("a", "k1", map[string]any{"x": float64(1)})
	s.Put("b", "k1", map[string]any{"x": float64(2)})

	aDoc, _ := s.Get("a", "k1")
	bDoc, _ := s.Get("b", "k1")

	if aDoc["x"] != float64(1) {
		t.Fatalf("collection a: expected x=1, got %v", aDoc["x"])
	}
	if bDoc["x"] != float64(2) {
		t.Fatalf("collection b: expected x=2, got %v", bDoc["x"])
	}

	// Verify separate files
	if _, err := os.Stat(filepath.Join(dir, "a.json")); err != nil {
		t.Fatalf("expected a.json to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "b.json")); err != nil {
		t.Fatalf("expected b.json to exist: %v", err)
	}
}
