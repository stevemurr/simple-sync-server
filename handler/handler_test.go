package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stevemurr/simple-sync-server/handler"
	"github.com/stevemurr/simple-sync-server/store"
)

func setup() (*httptest.Server, store.Store) {
	s := store.NewMemoryStore()
	h := handler.New(s)
	ts := httptest.NewServer(h)
	return ts, s
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func decodeJSON(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func decodeJSONArray(t *testing.T, r io.Reader) []any {
	t.Helper()
	var v []any
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestRootAndHealth(t *testing.T) {
	ts, _ := setup()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := decodeJSON(t, resp.Body)
	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", body["status"])
	}

	resp, err = http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestNotesCRUD(t *testing.T) {
	ts, _ := setup()
	defer ts.Close()

	// GET /notes - empty
	resp, _ := http.Get(ts.URL + "/notes")
	items := decodeJSONArray(t, resp.Body)
	if len(items) != 0 {
		t.Fatalf("expected 0 notes, got %d", len(items))
	}

	// PUT /notes/2024-01-15
	note := map[string]any{
		"dateKey":   "2024-01-15",
		"content":   "Hello world",
		"updatedAt": "2024-01-15T10:00:00Z",
	}
	req, _ := http.NewRequest("PUT", ts.URL+"/notes/2024-01-15", bytes.NewReader(mustJSON(t, note)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// GET /notes/2024-01-15
	resp, _ = http.Get(ts.URL + "/notes/2024-01-15")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	got := decodeJSON(t, resp.Body)
	if got["content"] != "Hello world" {
		t.Fatalf("expected content=Hello world, got %v", got["content"])
	}

	// PUT older - should not update
	olderNote := map[string]any{
		"dateKey":   "2024-01-15",
		"content":   "Older content",
		"updatedAt": "2024-01-14T10:00:00Z",
	}
	req, _ = http.NewRequest("PUT", ts.URL+"/notes/2024-01-15", bytes.NewReader(mustJSON(t, olderNote)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	got = decodeJSON(t, resp.Body)
	if got["content"] != "Hello world" {
		t.Fatalf("expected content unchanged, got %v", got["content"])
	}

	// GET /notes - should have 1
	resp, _ = http.Get(ts.URL + "/notes")
	items = decodeJSONArray(t, resp.Body)
	if len(items) != 1 {
		t.Fatalf("expected 1 note, got %d", len(items))
	}

	// DELETE /notes/2024-01-15
	req, _ = http.NewRequest("DELETE", ts.URL+"/notes/2024-01-15", nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// GET should return 404
	resp, _ = http.Get(ts.URL + "/notes/2024-01-15")
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestCollectionsCRUD(t *testing.T) {
	ts, _ := setup()
	defer ts.Close()

	// PUT item in a custom collection
	item := map[string]any{
		"id":        "task-1",
		"title":     "Buy milk",
		"updatedAt": "2024-06-01T12:00:00Z",
	}
	req, _ := http.NewRequest("PUT", ts.URL+"/collections/tasks/items/task-1", bytes.NewReader(mustJSON(t, item)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// GET all items
	resp, _ = http.Get(ts.URL + "/collections/tasks/items")
	items := decodeJSONArray(t, resp.Body)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	// GET specific item
	resp, _ = http.Get(ts.URL + "/collections/tasks/items/task-1")
	got := decodeJSON(t, resp.Body)
	if got["title"] != "Buy milk" {
		t.Fatalf("expected title=Buy milk, got %v", got["title"])
	}

	// List collections
	resp, _ = http.Get(ts.URL + "/collections")
	var names []string
	json.NewDecoder(resp.Body).Decode(&names)
	found := false
	for _, n := range names {
		if n == "tasks" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'tasks' in collections, got %v", names)
	}
}

func TestSync(t *testing.T) {
	ts, _ := setup()
	defer ts.Close()

	// First sync: push notes, get back all
	syncReq := map[string]any{
		"notes": []any{
			map[string]any{
				"dateKey":   "2024-01-01",
				"content":   "Note 1",
				"updatedAt": "2024-01-01T12:00:00Z",
			},
			map[string]any{
				"dateKey":   "2024-01-02",
				"content":   "Note 2",
				"updatedAt": "2024-01-02T12:00:00Z",
			},
		},
		"lastSyncTime": nil,
	}
	resp, _ := http.Post(ts.URL+"/sync", "application/json", bytes.NewReader(mustJSON(t, syncReq)))
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	syncResp := decodeJSON(t, resp.Body)
	notes := syncResp["notes"].([]any)
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes in sync response, got %d", len(notes))
	}
	if syncResp["serverTime"] == nil {
		t.Fatal("expected serverTime")
	}

	// Incremental sync
	syncReq2 := map[string]any{
		"notes": []any{
			map[string]any{
				"dateKey":   "2024-01-03",
				"content":   "Note 3",
				"updatedAt": "2024-01-03T12:00:00Z",
			},
		},
		"lastSyncTime": "2024-01-02T00:00:00Z",
	}
	resp, _ = http.Post(ts.URL+"/sync", "application/json", bytes.NewReader(mustJSON(t, syncReq2)))
	syncResp = decodeJSON(t, resp.Body)
	notes = syncResp["notes"].([]any)
	// Should return notes 2 and 3 (both after lastSyncTime)
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes in incremental sync, got %d", len(notes))
	}
}

func TestCollectionSync(t *testing.T) {
	ts, _ := setup()
	defer ts.Close()

	// Sync using the generic collection endpoint with "items" field
	syncReq := map[string]any{
		"items": []any{
			map[string]any{
				"id":        "t1",
				"title":     "Task 1",
				"updatedAt": "2024-06-01T12:00:00Z",
			},
		},
		"lastSyncTime": nil,
	}
	resp, _ := http.Post(ts.URL+"/collections/tasks/sync", "application/json", bytes.NewReader(mustJSON(t, syncReq)))
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	syncResp := decodeJSON(t, resp.Body)
	items := syncResp["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestSchemaCRUD(t *testing.T) {
	ts, _ := setup()
	defer ts.Close()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
		"required": []any{"name"},
	}

	// PUT schema
	req, _ := http.NewRequest("PUT", ts.URL+"/schemas/users", bytes.NewReader(mustJSON(t, schema)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// GET schema
	resp, _ = http.Get(ts.URL + "/schemas/users")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	got := decodeJSON(t, resp.Body)
	if got["type"] != "object" {
		t.Fatalf("expected type=object, got %v", got["type"])
	}

	// LIST schemas
	resp, _ = http.Get(ts.URL + "/schemas")
	schemas := decodeJSON(t, resp.Body)
	if _, ok := schemas["users"]; !ok {
		t.Fatal("expected 'users' in schema list")
	}

	// DELETE schema
	req, _ = http.NewRequest("DELETE", ts.URL+"/schemas/users", nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// GET should 404
	resp, _ = http.Get(ts.URL + "/schemas/users")
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestSchemaValidationOnPut(t *testing.T) {
	ts, s := setup()
	defer ts.Close()

	// Set a schema requiring "name" as string
	s.PutSchema("users", map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required": []any{"name"},
	})

	// Valid document
	doc := map[string]any{"name": "Alice", "updatedAt": "2024-01-01T00:00:00Z"}
	req, _ := http.NewRequest("PUT", ts.URL+"/collections/users/items/u1", bytes.NewReader(mustJSON(t, doc)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	// Invalid: missing required "name"
	badDoc := map[string]any{"age": float64(30), "updatedAt": "2024-01-01T00:00:00Z"}
	req, _ = http.NewRequest("PUT", ts.URL+"/collections/users/items/u2", bytes.NewReader(mustJSON(t, badDoc)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 422 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, body)
	}

	// Invalid: wrong type for "name"
	badDoc2 := map[string]any{"name": float64(123), "updatedAt": "2024-01-01T00:00:00Z"}
	req, _ = http.NewRequest("PUT", ts.URL+"/collections/users/items/u3", bytes.NewReader(mustJSON(t, badDoc2)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 422 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, body)
	}
}

func TestGetNotesSince(t *testing.T) {
	ts, _ := setup()
	defer ts.Close()

	// Add some notes
	for _, n := range []map[string]any{
		{"dateKey": "d1", "content": "old", "updatedAt": "2024-01-01T00:00:00Z"},
		{"dateKey": "d2", "content": "new", "updatedAt": "2024-06-01T00:00:00Z"},
	} {
		req, _ := http.NewRequest("PUT", ts.URL+"/notes/"+n["dateKey"].(string), bytes.NewReader(mustJSON(t, n)))
		req.Header.Set("Content-Type", "application/json")
		http.DefaultClient.Do(req)
	}

	resp, _ := http.Get(ts.URL + "/notes/since/2024-03-01T00:00:00Z")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	items := decodeJSONArray(t, resp.Body)
	if len(items) != 1 {
		t.Fatalf("expected 1 item since March, got %d", len(items))
	}
}
