package store

import (
	"encoding/json"
	"sort"
	"sync"
)

// MemoryStore keeps everything in memory. Data is lost on restart.
// Safe for concurrent use.
type MemoryStore struct {
	mu          sync.RWMutex
	collections map[string]map[string]map[string]any
	schemas     map[string]map[string]any
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		collections: make(map[string]map[string]map[string]any),
		schemas:     make(map[string]map[string]any),
	}
}

// deepCopy returns a deep copy of a document by round-tripping through JSON.
func deepCopy(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	b, _ := json.Marshal(src)
	var dst map[string]any
	_ = json.Unmarshal(b, &dst)
	return dst
}

func (m *MemoryStore) GetAll(collection string) (map[string]map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	coll, ok := m.collections[collection]
	if !ok {
		return map[string]map[string]any{}, nil
	}
	result := make(map[string]map[string]any, len(coll))
	for k, v := range coll {
		result[k] = deepCopy(v)
	}
	return result, nil
}

func (m *MemoryStore) Get(collection, key string) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	coll, ok := m.collections[collection]
	if !ok {
		return nil, nil
	}
	doc, ok := coll[key]
	if !ok {
		return nil, nil
	}
	return deepCopy(doc), nil
}

func (m *MemoryStore) Put(collection, key string, data map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.collections[collection]; !ok {
		m.collections[collection] = make(map[string]map[string]any)
	}
	m.collections[collection][key] = deepCopy(data)
	return nil
}

func (m *MemoryStore) Delete(collection, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	coll, ok := m.collections[collection]
	if !ok {
		return false, nil
	}
	if _, exists := coll[key]; !exists {
		return false, nil
	}
	delete(coll, key)
	return true, nil
}

func (m *MemoryStore) ListCollections() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var names []string
	for name, docs := range m.collections {
		if len(docs) > 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func (m *MemoryStore) GetSchema(collection string) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.schemas[collection]
	if !ok {
		return nil, nil
	}
	return deepCopy(s), nil
}

func (m *MemoryStore) PutSchema(collection string, schema map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schemas[collection] = deepCopy(schema)
	return nil
}

func (m *MemoryStore) DeleteSchema(collection string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.schemas[collection]; !ok {
		return false, nil
	}
	delete(m.schemas, collection)
	return true, nil
}

func (m *MemoryStore) ListSchemas() (map[string]map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]map[string]any, len(m.schemas))
	for k, v := range m.schemas {
		result[k] = deepCopy(v)
	}
	return result, nil
}
