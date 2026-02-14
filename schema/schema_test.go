package schema_test

import (
	"testing"

	"github.com/stevemurr/simple-sync-server/schema"
)

func TestValidateNilSchema(t *testing.T) {
	err := schema.Validate(nil, map[string]any{"anything": "goes"})
	if err != nil {
		t.Fatalf("nil schema should pass: %v", err)
	}
}

func TestValidateType(t *testing.T) {
	s := map[string]any{"type": "object"}

	if err := schema.Validate(s, map[string]any{}); err != nil {
		t.Fatalf("expected pass: %v", err)
	}
}

func TestValidateRequired(t *testing.T) {
	s := map[string]any{
		"type":     "object",
		"required": []any{"name", "age"},
	}

	// Missing required field
	err := schema.Validate(s, map[string]any{"name": "Alice"})
	if err == nil {
		t.Fatal("expected error for missing 'age'")
	}

	// All present
	err = schema.Validate(s, map[string]any{"name": "Alice", "age": float64(30)})
	if err != nil {
		t.Fatalf("expected pass: %v", err)
	}
}

func TestValidateProperties(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
	}

	// Correct types
	err := schema.Validate(s, map[string]any{"name": "Bob", "age": float64(25)})
	if err != nil {
		t.Fatalf("expected pass: %v", err)
	}

	// Wrong type
	err = schema.Validate(s, map[string]any{"name": float64(123)})
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestValidateAdditionalProperties(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"additionalProperties": false,
	}

	err := schema.Validate(s, map[string]any{"name": "ok", "extra": "bad"})
	if err == nil {
		t.Fatal("expected error for additional properties")
	}

	err = schema.Validate(s, map[string]any{"name": "ok"})
	if err != nil {
		t.Fatalf("expected pass: %v", err)
	}
}

func TestValidateStringConstraints(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":      "string",
				"minLength": float64(2),
				"maxLength": float64(5),
			},
		},
	}

	// Too short
	err := schema.Validate(s, map[string]any{"code": "A"})
	if err == nil {
		t.Fatal("expected error for too-short string")
	}

	// Too long
	err = schema.Validate(s, map[string]any{"code": "ABCDEF"})
	if err == nil {
		t.Fatal("expected error for too-long string")
	}

	// Just right
	err = schema.Validate(s, map[string]any{"code": "ABC"})
	if err != nil {
		t.Fatalf("expected pass: %v", err)
	}
}

func TestValidateNumberConstraints(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"score": map[string]any{
				"type":    "number",
				"minimum": float64(0),
				"maximum": float64(100),
			},
		},
	}

	err := schema.Validate(s, map[string]any{"score": float64(-1)})
	if err == nil {
		t.Fatal("expected error for below minimum")
	}

	err = schema.Validate(s, map[string]any{"score": float64(101)})
	if err == nil {
		t.Fatal("expected error for above maximum")
	}

	err = schema.Validate(s, map[string]any{"score": float64(50)})
	if err != nil {
		t.Fatalf("expected pass: %v", err)
	}
}

func TestValidateEnum(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"role": map[string]any{
				"type": "string",
				"enum": []any{"admin", "user", "guest"},
			},
		},
	}

	err := schema.Validate(s, map[string]any{"role": "admin"})
	if err != nil {
		t.Fatalf("expected pass: %v", err)
	}

	err = schema.Validate(s, map[string]any{"role": "superadmin"})
	if err == nil {
		t.Fatal("expected error for invalid enum value")
	}
}

func TestValidateArray(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tags": map[string]any{
				"type":     "array",
				"items":    map[string]any{"type": "string"},
				"minItems": float64(1),
				"maxItems": float64(3),
			},
		},
	}

	// Empty array
	err := schema.Validate(s, map[string]any{"tags": []any{}})
	if err == nil {
		t.Fatal("expected error for empty array (minItems=1)")
	}

	// Too many
	err = schema.Validate(s, map[string]any{"tags": []any{"a", "b", "c", "d"}})
	if err == nil {
		t.Fatal("expected error for too many items")
	}

	// Wrong item type
	err = schema.Validate(s, map[string]any{"tags": []any{"a", float64(1)}})
	if err == nil {
		t.Fatal("expected error for wrong item type")
	}

	// OK
	err = schema.Validate(s, map[string]any{"tags": []any{"go", "rust"}})
	if err != nil {
		t.Fatalf("expected pass: %v", err)
	}
}

func TestValidateNestedObject(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"address": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{"type": "string"},
					"zip":  map[string]any{"type": "string"},
				},
				"required": []any{"city"},
			},
		},
	}

	// Missing required nested field
	err := schema.Validate(s, map[string]any{
		"address": map[string]any{"zip": "12345"},
	})
	if err == nil {
		t.Fatal("expected error for missing nested required field")
	}

	// Valid
	err = schema.Validate(s, map[string]any{
		"address": map[string]any{"city": "NY", "zip": "10001"},
	})
	if err != nil {
		t.Fatalf("expected pass: %v", err)
	}
}

func TestValidateIntegerType(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"count": map[string]any{"type": "integer"},
		},
	}

	// float64 that is a whole number should pass as integer
	err := schema.Validate(s, map[string]any{"count": float64(5)})
	if err != nil {
		t.Fatalf("expected pass for whole float64: %v", err)
	}

	// float64 with fractional part should fail
	err = schema.Validate(s, map[string]any{"count": float64(5.5)})
	if err == nil {
		t.Fatal("expected error for fractional number as integer")
	}
}
