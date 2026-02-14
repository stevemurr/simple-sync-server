package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// JsonFileStore stores each collection as a separate JSON file on disk.
//
// Layout:
//
//	data_dir/
//	  _schemas.json   # schema registry
//	  notes.json      # "notes" collection
//	  tasks.json      # "tasks" collection
type JsonFileStore struct {
	mu  sync.RWMutex
	dir string
}

func NewJsonFileStore(dir string) (*JsonFileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &JsonFileStore{dir: dir}, nil
}

func (s *JsonFileStore) collectionPath(collection string) string {
	return filepath.Join(s.dir, collection+".json")
}

func (s *JsonFileStore) schemasPath() string {
	return filepath.Join(s.dir, "_schemas.json")
}

func (s *JsonFileStore) loadFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]any{}, nil
	}
	return result, nil
}

func (s *JsonFileStore) saveFile(path string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// loadCollection loads a file as map[string]map[string]any.
func (s *JsonFileStore) loadCollection(path string) (map[string]map[string]any, error) {
	raw, err := s.loadFile(path)
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string]any, len(raw))
	for k, v := range raw {
		if doc, ok := v.(map[string]any); ok {
			result[k] = doc
		}
	}
	return result, nil
}

func (s *JsonFileStore) GetAll(collection string) (map[string]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadCollection(s.collectionPath(collection))
}

func (s *JsonFileStore) Get(collection, key string) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	coll, err := s.loadCollection(s.collectionPath(collection))
	if err != nil {
		return nil, err
	}
	doc, ok := coll[key]
	if !ok {
		return nil, nil
	}
	return doc, nil
}

func (s *JsonFileStore) Put(collection, key string, data map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.collectionPath(collection)
	coll, err := s.loadCollection(path)
	if err != nil {
		return err
	}
	coll[key] = data
	return s.saveFile(path, coll)
}

func (s *JsonFileStore) Delete(collection, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.collectionPath(collection)
	coll, err := s.loadCollection(path)
	if err != nil {
		return false, err
	}
	if _, ok := coll[key]; !ok {
		return false, nil
	}
	delete(coll, key)
	return true, s.saveFile(path, coll)
}

func (s *JsonFileStore) ListCollections() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "_") || !strings.HasSuffix(name, ".json") {
			continue
		}
		names = append(names, strings.TrimSuffix(name, ".json"))
	}
	sort.Strings(names)
	return names, nil
}

func (s *JsonFileStore) GetSchema(collection string) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	schemas, err := s.loadFile(s.schemasPath())
	if err != nil {
		return nil, err
	}
	raw, ok := schemas[collection]
	if !ok {
		return nil, nil
	}
	if schema, ok := raw.(map[string]any); ok {
		return schema, nil
	}
	return nil, nil
}

func (s *JsonFileStore) PutSchema(collection string, schema map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.schemasPath()
	schemas, err := s.loadFile(path)
	if err != nil {
		return err
	}
	schemas[collection] = schema
	return s.saveFile(path, schemas)
}

func (s *JsonFileStore) DeleteSchema(collection string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.schemasPath()
	schemas, err := s.loadFile(path)
	if err != nil {
		return false, err
	}
	if _, ok := schemas[collection]; !ok {
		return false, nil
	}
	delete(schemas, collection)
	return true, s.saveFile(path, schemas)
}

func (s *JsonFileStore) ListSchemas() (map[string]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw, err := s.loadFile(s.schemasPath())
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string]any, len(raw))
	for k, v := range raw {
		if schema, ok := v.(map[string]any); ok {
			result[k] = schema
		}
	}
	return result, nil
}
