// Package schema provides JSON Schema validation for collection documents.
package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Validate checks a document against a JSON Schema (draft-07 subset).
// Returns nil if validation passes or the schema is nil.
//
// Supported JSON Schema keywords:
//   - type (string, number, integer, boolean, object, array, null)
//   - properties, required, additionalProperties
//   - items (for arrays)
//   - minimum, maximum, exclusiveMinimum, exclusiveMaximum
//   - minLength, maxLength
//   - minItems, maxItems
//   - enum
func Validate(schema map[string]any, doc map[string]any) error {
	if schema == nil {
		return nil
	}
	return validateValue(schema, doc, "")
}

func validateValue(schema map[string]any, value any, path string) error {
	if path == "" {
		path = "$"
	}

	// Check type constraint
	if t, ok := schema["type"]; ok {
		if ts, ok := t.(string); ok {
			if err := checkType(ts, value, path); err != nil {
				return err
			}
		}
	}

	// Check enum
	if enumRaw, ok := schema["enum"]; ok {
		if enumList, ok := enumRaw.([]any); ok {
			if err := checkEnum(enumList, value, path); err != nil {
				return err
			}
		}
	}

	switch v := value.(type) {
	case map[string]any:
		return validateObject(schema, v, path)
	case []any:
		return validateArray(schema, v, path)
	case string:
		return validateString(schema, v, path)
	case float64:
		return validateNumber(schema, v, path)
	case json.Number:
		f, _ := v.Float64()
		return validateNumber(schema, f, path)
	}

	return nil
}

func checkType(expected string, value any, path string) error {
	actual := jsonType(value)
	if expected == "integer" {
		// Accept float64 values that are whole numbers
		if f, ok := value.(float64); ok && f == float64(int64(f)) {
			return nil
		}
		if actual != "integer" {
			return fmt.Errorf("%s: expected type %q, got %q", path, expected, actual)
		}
		return nil
	}
	if actual != expected {
		// "number" should also accept integer
		if expected == "number" && actual == "integer" {
			return nil
		}
		return fmt.Errorf("%s: expected type %q, got %q", path, expected, actual)
	}
	return nil
}

func jsonType(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case json.Number:
		return "number"
	case int, int64:
		return "integer"
	default:
		return reflect.TypeOf(v).String()
	}
}

func checkEnum(allowed []any, value any, path string) error {
	for _, a := range allowed {
		if reflect.DeepEqual(a, value) {
			return nil
		}
	}
	return fmt.Errorf("%s: value not in enum %v", path, allowed)
}

func validateObject(schema map[string]any, obj map[string]any, path string) error {
	// Check required fields
	if req, ok := schema["required"]; ok {
		if reqList, ok := req.([]any); ok {
			for _, r := range reqList {
				if field, ok := r.(string); ok {
					if _, exists := obj[field]; !exists {
						return fmt.Errorf("%s: missing required field %q", path, field)
					}
				}
			}
		}
	}

	// Validate properties
	if props, ok := schema["properties"]; ok {
		if propsMap, ok := props.(map[string]any); ok {
			for field, propSchema := range propsMap {
				val, exists := obj[field]
				if !exists {
					continue
				}
				ps, ok := propSchema.(map[string]any)
				if !ok {
					continue
				}
				if err := validateValue(ps, val, path+"."+field); err != nil {
					return err
				}
			}
		}
	}

	// Check additionalProperties
	if ap, ok := schema["additionalProperties"]; ok {
		if apBool, ok := ap.(bool); ok && !apBool {
			propsMap := map[string]any{}
			if props, ok := schema["properties"]; ok {
				if pm, ok := props.(map[string]any); ok {
					propsMap = pm
				}
			}
			var extra []string
			for field := range obj {
				if _, defined := propsMap[field]; !defined {
					extra = append(extra, field)
				}
			}
			if len(extra) > 0 {
				return fmt.Errorf("%s: additional properties not allowed: %s", path, strings.Join(extra, ", "))
			}
		}
	}

	return nil
}

func validateArray(schema map[string]any, arr []any, path string) error {
	// minItems
	if v, ok := toFloat(schema["minItems"]); ok {
		if float64(len(arr)) < v {
			return fmt.Errorf("%s: array length %d is less than minItems %v", path, len(arr), v)
		}
	}
	// maxItems
	if v, ok := toFloat(schema["maxItems"]); ok {
		if float64(len(arr)) > v {
			return fmt.Errorf("%s: array length %d is greater than maxItems %v", path, len(arr), v)
		}
	}
	// Validate items
	if items, ok := schema["items"]; ok {
		if itemSchema, ok := items.(map[string]any); ok {
			for i, elem := range arr {
				if err := validateValue(itemSchema, elem, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateString(schema map[string]any, s string, path string) error {
	if v, ok := toFloat(schema["minLength"]); ok {
		if float64(len(s)) < v {
			return fmt.Errorf("%s: string length %d is less than minLength %v", path, len(s), v)
		}
	}
	if v, ok := toFloat(schema["maxLength"]); ok {
		if float64(len(s)) > v {
			return fmt.Errorf("%s: string length %d is greater than maxLength %v", path, len(s), v)
		}
	}
	return nil
}

func validateNumber(schema map[string]any, n float64, path string) error {
	if v, ok := toFloat(schema["minimum"]); ok {
		if n < v {
			return fmt.Errorf("%s: %v is less than minimum %v", path, n, v)
		}
	}
	if v, ok := toFloat(schema["maximum"]); ok {
		if n > v {
			return fmt.Errorf("%s: %v is greater than maximum %v", path, n, v)
		}
	}
	if v, ok := toFloat(schema["exclusiveMinimum"]); ok {
		if n <= v {
			return fmt.Errorf("%s: %v is not greater than exclusiveMinimum %v", path, n, v)
		}
	}
	if v, ok := toFloat(schema["exclusiveMaximum"]); ok {
		if n >= v {
			return fmt.Errorf("%s: %v is not less than exclusiveMaximum %v", path, n, v)
		}
	}
	return nil
}

func toFloat(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}
