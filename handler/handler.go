// Package handler provides the HTTP handlers for the sync server.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stevemurr/simple-sync-server/schema"
	"github.com/stevemurr/simple-sync-server/store"
)

// Handler holds the server dependencies and registers routes.
type Handler struct {
	store store.Store
	mux   *http.ServeMux
}

// New creates a Handler and wires up all routes.
func New(s store.Store) *Handler {
	h := &Handler{store: s, mux: http.NewServeMux()}
	h.routes()
	return h
}

// ServeHTTP makes Handler an http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) routes() {
	// Health / status
	h.mux.HandleFunc("GET /", h.root)
	h.mux.HandleFunc("GET /health", h.health)

	// --- Backward-compatible notes endpoints ---
	h.mux.HandleFunc("GET /notes", h.getAllItems("notes"))
	h.mux.HandleFunc("GET /notes/since/{timestamp}", h.getItemsSince("notes"))
	h.mux.HandleFunc("GET /notes/{key}", h.getItem("notes"))
	h.mux.HandleFunc("PUT /notes/{key}", h.upsertItem("notes"))
	h.mux.HandleFunc("DELETE /notes/{key}", h.deleteItem("notes"))
	h.mux.HandleFunc("POST /sync", h.syncCollection("notes"))

	// --- Generic collection endpoints ---
	h.mux.HandleFunc("GET /collections", h.listCollections)
	h.mux.HandleFunc("GET /collections/{collection}/items", h.getAllItemsDynamic)
	h.mux.HandleFunc("GET /collections/{collection}/items/since/{timestamp}", h.getItemsSinceDynamic)
	h.mux.HandleFunc("GET /collections/{collection}/items/{key}", h.getItemDynamic)
	h.mux.HandleFunc("PUT /collections/{collection}/items/{key}", h.upsertItemDynamic)
	h.mux.HandleFunc("DELETE /collections/{collection}/items/{key}", h.deleteItemDynamic)
	h.mux.HandleFunc("POST /collections/{collection}/sync", h.syncCollectionDynamic)

	// --- Schema endpoints ---
	h.mux.HandleFunc("GET /schemas", h.listSchemas)
	h.mux.HandleFunc("GET /schemas/{collection}", h.getSchema)
	h.mux.HandleFunc("PUT /schemas/{collection}", h.putSchema)
	h.mux.HandleFunc("DELETE /schemas/{collection}", h.deleteSchema)
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"detail": msg})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func parseISO(s string) (time.Time, error) {
	s = strings.Replace(s, "Z", "+00:00", 1)
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try without timezone
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid timestamp: %s", s)
}

// ---------- status endpoints ----------

func (h *Handler) root(w http.ResponseWriter, r *http.Request) {
	// Only match exact root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "Simple Sync Server",
	})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// ---------- collection list ----------

func (h *Handler) listCollections(w http.ResponseWriter, r *http.Request) {
	names, err := h.store.ListCollections()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if names == nil {
		names = []string{}
	}
	writeJSON(w, http.StatusOK, names)
}

// ---------- item CRUD (fixed collection) ----------

func (h *Handler) getAllItems(collection string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.doGetAllItems(w, r, collection)
	}
}

func (h *Handler) getItem(collection string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.doGetItem(w, r, collection, r.PathValue("key"))
	}
}

func (h *Handler) upsertItem(collection string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.doUpsertItem(w, r, collection, r.PathValue("key"))
	}
}

func (h *Handler) deleteItem(collection string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.doDeleteItem(w, r, collection, r.PathValue("key"))
	}
}

func (h *Handler) getItemsSince(collection string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.doGetItemsSince(w, r, collection, r.PathValue("timestamp"))
	}
}

func (h *Handler) syncCollection(collection string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.doSync(w, r, collection)
	}
}

// ---------- item CRUD (dynamic collection from path) ----------

func (h *Handler) getAllItemsDynamic(w http.ResponseWriter, r *http.Request) {
	h.doGetAllItems(w, r, r.PathValue("collection"))
}

func (h *Handler) getItemDynamic(w http.ResponseWriter, r *http.Request) {
	h.doGetItem(w, r, r.PathValue("collection"), r.PathValue("key"))
}

func (h *Handler) upsertItemDynamic(w http.ResponseWriter, r *http.Request) {
	h.doUpsertItem(w, r, r.PathValue("collection"), r.PathValue("key"))
}

func (h *Handler) deleteItemDynamic(w http.ResponseWriter, r *http.Request) {
	h.doDeleteItem(w, r, r.PathValue("collection"), r.PathValue("key"))
}

func (h *Handler) getItemsSinceDynamic(w http.ResponseWriter, r *http.Request) {
	h.doGetItemsSince(w, r, r.PathValue("collection"), r.PathValue("timestamp"))
}

func (h *Handler) syncCollectionDynamic(w http.ResponseWriter, r *http.Request) {
	h.doSync(w, r, r.PathValue("collection"))
}

// ---------- core logic ----------

func (h *Handler) doGetAllItems(w http.ResponseWriter, _ *http.Request, collection string) {
	docs, err := h.store.GetAll(collection)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		items = append(items, doc)
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) doGetItem(w http.ResponseWriter, _ *http.Request, collection, key string) {
	doc, err := h.store.Get(collection, key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if doc == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) doUpsertItem(w http.ResponseWriter, r *http.Request, collection, key string) {
	var incoming map[string]any
	if err := readJSON(r, &incoming); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate against schema if one exists
	if err := h.validateAgainstSchema(collection, incoming); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "schema validation failed: "+err.Error())
		return
	}

	// Last-write-wins: only update if incoming is newer
	existing, err := h.store.Get(collection, key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing != nil {
		if existingTS, ok := existing["updatedAt"].(string); ok {
			if incomingTS, ok := incoming["updatedAt"].(string); ok {
				et, err1 := parseISO(existingTS)
				nt, err2 := parseISO(incomingTS)
				if err1 == nil && err2 == nil && !nt.After(et) {
					writeJSON(w, http.StatusOK, existing)
					return
				}
			}
		}
	}

	if err := h.store.Put(collection, key, incoming); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, incoming)
}

func (h *Handler) doDeleteItem(w http.ResponseWriter, _ *http.Request, collection, key string) {
	if _, err := h.store.Delete(collection, key); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "key": key})
}

func (h *Handler) doGetItemsSince(w http.ResponseWriter, _ *http.Request, collection, timestamp string) {
	since, err := parseISO(timestamp)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid timestamp format")
		return
	}
	docs, err := h.store.GetAll(collection)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var result []map[string]any
	for _, doc := range docs {
		if ts, ok := doc["updatedAt"].(string); ok {
			t, err := parseISO(ts)
			if err == nil && t.After(since) {
				result = append(result, doc)
			}
		}
	}
	if result == nil {
		result = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) doSync(w http.ResponseWriter, r *http.Request, collection string) {
	var req struct {
		Items        []map[string]any `json:"items"`
		Notes        []map[string]any `json:"notes"` // backward compat
		LastSyncTime *string          `json:"lastSyncTime"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Support both "items" and "notes" fields for backward compatibility
	incoming := req.Items
	if len(incoming) == 0 && len(req.Notes) > 0 {
		incoming = req.Notes
	}

	serverTime := time.Now().UTC().Format(time.RFC3339)

	var lastSync *time.Time
	if req.LastSyncTime != nil && *req.LastSyncTime != "" {
		t, err := parseISO(*req.LastSyncTime)
		if err == nil {
			lastSync = &t
		}
	}

	serverDocs, err := h.store.GetAll(collection)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Determine the key field: use "dateKey" if present (backward compat), else "key", else "id"
	keyOf := func(doc map[string]any) string {
		for _, field := range []string{"dateKey", "key", "id"} {
			if v, ok := doc[field].(string); ok && v != "" {
				return v
			}
		}
		return ""
	}

	// Merge incoming
	for _, doc := range incoming {
		key := keyOf(doc)
		if key == "" {
			continue
		}

		// Validate against schema
		if err := h.validateAgainstSchema(collection, doc); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "schema validation failed: "+err.Error())
			return
		}

		newTS, _ := doc["updatedAt"].(string)
		newTime, err := parseISO(newTS)
		if err != nil {
			continue
		}

		if existing, ok := serverDocs[key]; ok {
			existTS, _ := existing["updatedAt"].(string)
			existTime, err := parseISO(existTS)
			if err == nil && !newTime.After(existTime) {
				continue
			}
		}
		serverDocs[key] = doc
		if err := h.store.Put(collection, key, doc); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Build response: items newer than lastSyncTime
	var toReturn []map[string]any
	for _, doc := range serverDocs {
		if lastSync == nil {
			toReturn = append(toReturn, doc)
		} else {
			ts, _ := doc["updatedAt"].(string)
			t, err := parseISO(ts)
			if err == nil && t.After(*lastSync) {
				toReturn = append(toReturn, doc)
			}
		}
	}
	if toReturn == nil {
		toReturn = []map[string]any{}
	}

	// Return using both field names for backward compat with notes
	resp := map[string]any{
		"items":      toReturn,
		"serverTime": serverTime,
	}
	if collection == "notes" {
		resp["notes"] = toReturn
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------- schema endpoints ----------

func (h *Handler) listSchemas(w http.ResponseWriter, r *http.Request) {
	schemas, err := h.store.ListSchemas()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if schemas == nil {
		schemas = map[string]map[string]any{}
	}
	writeJSON(w, http.StatusOK, schemas)
}

func (h *Handler) getSchema(w http.ResponseWriter, r *http.Request) {
	collection := r.PathValue("collection")
	s, err := h.store.GetSchema(collection)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if s == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("no schema for collection %q", collection))
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) putSchema(w http.ResponseWriter, r *http.Request) {
	collection := r.PathValue("collection")
	var s map[string]any
	if err := readJSON(r, &s); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := h.store.PutSchema(collection, s); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) deleteSchema(w http.ResponseWriter, r *http.Request) {
	collection := r.PathValue("collection")
	existed, err := h.store.DeleteSchema(collection)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !existed {
		writeError(w, http.StatusNotFound, fmt.Sprintf("no schema for collection %q", collection))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "collection": collection})
}

// ---------- schema validation helper ----------

func (h *Handler) validateAgainstSchema(collection string, doc map[string]any) error {
	s, err := h.store.GetSchema(collection)
	if err != nil {
		return err
	}
	if s == nil {
		return nil // no schema = no validation
	}
	return schema.Validate(s, doc)
}
