package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// GenerateSchema builds a lightweight JSON schema from a struct definition.
func GenerateSchema(v interface{}) (map[string]interface{}, error) {
	if v == nil {
		return nil, errors.New("schema value cannot be nil")
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("schema must be a struct, got %s", t.Kind())
	}

	properties := make(map[string]interface{})
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() || field.Tag.Get("json") == "-" {
			continue
		}

		name, omitEmpty := parseJSONTag(field)
		if name == "" {
			name = field.Name
		}

		prop := buildSchemaForType(field.Type)
		if desc := field.Tag.Get("description"); desc != "" {
			prop["description"] = desc
		}
		properties[name] = prop
		if !omitEmpty {
			required = append(required, name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema, nil
}

// ParseStructured decodes a JSON string into the provided target value.
func ParseStructured(jsonStr string, target interface{}) error {
	if target == nil {
		return errors.New("target cannot be nil")
	}
	if reflect.ValueOf(target).Kind() != reflect.Ptr {
		return errors.New("target must be a pointer")
	}
	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("decode structured response: %w", err)
	}
	return nil
}

func parseJSONTag(field reflect.StructField) (name string, omitEmpty bool) {
	tag := field.Tag.Get("json")
	if tag == "" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return "", false
	}
	name = parts[0]
	for _, part := range parts[1:] {
		if part == "omitempty" {
			omitEmpty = true
		}
	}
	return name, omitEmpty
}

func buildSchemaForType(t reflect.Type) map[string]interface{} {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.Slice, reflect.Array:
		itemSchema := buildSchemaForType(t.Elem())
		return map[string]interface{}{
			"type":  "array",
			"items": itemSchema,
		}
	case reflect.Map:
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": buildSchemaForType(t.Elem()),
		}
	case reflect.Struct:
		props := make(map[string]interface{})
		var required []string
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() || field.Tag.Get("json") == "-" {
				continue
			}
			name, omitEmpty := parseJSONTag(field)
			if name == "" {
				name = field.Name
			}
			props[name] = buildSchemaForType(field.Type)
			if !omitEmpty {
				required = append(required, name)
			}
		}
		schema := map[string]interface{}{
			"type":       "object",
			"properties": props,
		}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	default:
		return map[string]interface{}{"type": "string"}
	}
}
