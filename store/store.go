// Package store defines the backing store interface and implementations.
package store

// Store is the interface that all backing stores must implement.
// It operates on named collections, where each collection contains
// documents keyed by a string identifier.
type Store interface {
	// GetAll returns every document in a collection as a map of key -> document.
	GetAll(collection string) (map[string]map[string]any, error)

	// Get returns a single document by key, or nil if not found.
	Get(collection, key string) (map[string]any, error)

	// Put inserts or replaces a document.
	Put(collection, key string, data map[string]any) error

	// Delete removes a document. Returns true if it existed.
	Delete(collection, key string) (bool, error)

	// ListCollections returns the names of all collections that contain data.
	ListCollections() ([]string, error)

	// GetSchema returns the JSON Schema for a collection, or nil.
	GetSchema(collection string) (map[string]any, error)

	// PutSchema stores a JSON Schema for a collection.
	PutSchema(collection string, schema map[string]any) error

	// DeleteSchema removes the schema for a collection. Returns true if it existed.
	DeleteSchema(collection string) (bool, error)

	// ListSchemas returns all schemas as collection_name -> schema.
	ListSchemas() (map[string]map[string]any, error)
}
