// Package store defines the backing store interface and implementations.
package store

import (
	"fmt"
	"strings"
	"time"
)

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

	// PutIfNewer atomically writes data only if its updatedAt is newer than the
	// existing document's updatedAt. Returns the stored document (either the
	// incoming data if written or the existing data if not) and whether a write
	// occurred.
	PutIfNewer(collection, key string, data map[string]any) (stored map[string]any, written bool, err error)

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

// ParseTimestamp parses an ISO 8601 timestamp string, trying RFC3339Nano first.
func ParseTimestamp(s string) (time.Time, error) {
	s = strings.Replace(s, "Z", "+00:00", 1)
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid timestamp: %s", s)
}

// IsNewer returns true if incoming's updatedAt is strictly after existing's updatedAt.
// If either document lacks a parseable updatedAt, incoming wins.
func IsNewer(incoming, existing map[string]any) bool {
	inTS, ok := incoming["updatedAt"].(string)
	if !ok {
		return true
	}
	exTS, ok := existing["updatedAt"].(string)
	if !ok {
		return true
	}
	inTime, err1 := ParseTimestamp(inTS)
	exTime, err2 := ParseTimestamp(exTS)
	if err1 != nil || err2 != nil {
		return true
	}
	return inTime.After(exTime)
}
