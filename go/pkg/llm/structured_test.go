package llm

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateSchema(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		_, err := GenerateSchema(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("non-struct type", func(t *testing.T) {
		_, err := GenerateSchema("string")
		require.Error(t, err)
		require.Contains(t, err.Error(), "must be a struct")
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		val := 42
		_, err := GenerateSchema(&val)
		require.Error(t, err)
		require.Contains(t, err.Error(), "must be a struct")
	})

	t.Run("simple struct", func(t *testing.T) {
		type Simple struct {
			Name  string `json:"name"`
			Age   int    `json:"age"`
			Email string `json:"email,omitempty"`
		}

		schema, err := GenerateSchema(Simple{})
		require.NoError(t, err)

		require.Equal(t, "object", schema["type"])
		props := schema["properties"].(map[string]interface{})
		require.Len(t, props, 3)

		nameSchema := props["name"].(map[string]interface{})
		require.Equal(t, "string", nameSchema["type"])

		ageSchema := props["age"].(map[string]interface{})
		require.Equal(t, "integer", ageSchema["type"])

		required := schema["required"].([]string)
		require.Contains(t, required, "name")
		require.Contains(t, required, "age")
		require.NotContains(t, required, "email")
	})

	t.Run("struct with description tags", func(t *testing.T) {
		type Described struct {
			Name string `json:"name" description:"The user's name"`
			Age  int    `json:"age" description:"The user's age in years"`
		}

		schema, err := GenerateSchema(Described{})
		require.NoError(t, err)

		props := schema["properties"].(map[string]interface{})
		nameSchema := props["name"].(map[string]interface{})
		require.Equal(t, "The user's name", nameSchema["description"])

		ageSchema := props["age"].(map[string]interface{})
		require.Equal(t, "The user's age in years", ageSchema["description"])
	})

	t.Run("struct with unexported fields", func(t *testing.T) {
		type WithPrivate struct {
			Public  string `json:"public"`
			private string
		}

		schema, err := GenerateSchema(WithPrivate{})
		require.NoError(t, err)

		props := schema["properties"].(map[string]interface{})
		require.Len(t, props, 1)
		require.Contains(t, props, "public")
	})

	t.Run("struct with json ignore", func(t *testing.T) {
		type WithIgnore struct {
			Included string `json:"included"`
			Ignored  string `json:"-"`
		}

		schema, err := GenerateSchema(WithIgnore{})
		require.NoError(t, err)

		props := schema["properties"].(map[string]interface{})
		require.Len(t, props, 1)
		require.Contains(t, props, "included")
		require.NotContains(t, props, "Ignored")
	})

	t.Run("struct with no json tag uses field name", func(t *testing.T) {
		type NoTag struct {
			FieldName string
		}

		schema, err := GenerateSchema(NoTag{})
		require.NoError(t, err)

		props := schema["properties"].(map[string]interface{})
		require.Contains(t, props, "FieldName")
	})
}

func TestParseStructured(t *testing.T) {
	t.Run("nil target", func(t *testing.T) {
		err := ParseStructured(`{"key":"value"}`, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("non-pointer target", func(t *testing.T) {
		type Result struct {
			Key string `json:"key"`
		}
		var result Result
		err := ParseStructured(`{"key":"value"}`, result)
		require.Error(t, err)
		require.Contains(t, err.Error(), "must be a pointer")
	})

	t.Run("valid json", func(t *testing.T) {
		type Result struct {
			Key   string `json:"key"`
			Value int    `json:"value"`
		}
		var result Result
		err := ParseStructured(`{"key":"test","value":42}`, &result)
		require.NoError(t, err)
		require.Equal(t, "test", result.Key)
		require.Equal(t, 42, result.Value)
	})

	t.Run("invalid json", func(t *testing.T) {
		type Result struct {
			Key string `json:"key"`
		}
		var result Result
		err := ParseStructured(`{invalid json}`, &result)
		require.Error(t, err)
		require.Contains(t, err.Error(), "decode structured response")
	})
}

func TestParseJSONTag(t *testing.T) {
	tests := []struct {
		name              string
		tag               string
		expectedName      string
		expectedOmitEmpty bool
	}{
		{
			name:              "simple name",
			tag:               "field_name",
			expectedName:      "field_name",
			expectedOmitEmpty: false,
		},
		{
			name:              "name with omitempty",
			tag:               "field_name,omitempty",
			expectedName:      "field_name",
			expectedOmitEmpty: true,
		},
		{
			name:              "omitempty with other options",
			tag:               "field_name,omitempty,string",
			expectedName:      "field_name",
			expectedOmitEmpty: true,
		},
		{
			name:              "empty tag",
			tag:               "",
			expectedName:      "",
			expectedOmitEmpty: false,
		},
		{
			name:              "dash only",
			tag:               "-",
			expectedName:      "-",
			expectedOmitEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Name: "TestField",
				Tag:  reflect.StructTag(`json:"` + tt.tag + `"`),
			}
			name, omitEmpty := parseJSONTag(field)
			require.Equal(t, tt.expectedName, name)
			require.Equal(t, tt.expectedOmitEmpty, omitEmpty)
		})
	}
}

func TestBuildSchemaForType(t *testing.T) {
	t.Run("boolean type", func(t *testing.T) {
		schema := buildSchemaForType(reflect.TypeOf(true))
		require.Equal(t, "boolean", schema["type"])
	})

	t.Run("string type", func(t *testing.T) {
		schema := buildSchemaForType(reflect.TypeOf(""))
		require.Equal(t, "string", schema["type"])
	})

	t.Run("integer types", func(t *testing.T) {
		intTypes := []interface{}{
			int(0), int8(0), int16(0), int32(0), int64(0),
			uint(0), uint8(0), uint16(0), uint32(0), uint64(0),
		}
		for _, val := range intTypes {
			schema := buildSchemaForType(reflect.TypeOf(val))
			require.Equal(t, "integer", schema["type"], "failed for type %T", val)
		}
	})

	t.Run("float types", func(t *testing.T) {
		floatTypes := []interface{}{float32(0.0), float64(0.0)}
		for _, val := range floatTypes {
			schema := buildSchemaForType(reflect.TypeOf(val))
			require.Equal(t, "number", schema["type"], "failed for type %T", val)
		}
	})

	t.Run("slice type", func(t *testing.T) {
		schema := buildSchemaForType(reflect.TypeOf([]string{}))
		require.Equal(t, "array", schema["type"])
		items := schema["items"].(map[string]interface{})
		require.Equal(t, "string", items["type"])
	})

	t.Run("array type", func(t *testing.T) {
		schema := buildSchemaForType(reflect.TypeOf([3]int{}))
		require.Equal(t, "array", schema["type"])
		items := schema["items"].(map[string]interface{})
		require.Equal(t, "integer", items["type"])
	})

	t.Run("map type", func(t *testing.T) {
		schema := buildSchemaForType(reflect.TypeOf(map[string]int{}))
		require.Equal(t, "object", schema["type"])
		additionalProps := schema["additionalProperties"].(map[string]interface{})
		require.Equal(t, "integer", additionalProps["type"])
	})

	t.Run("map with complex values", func(t *testing.T) {
		schema := buildSchemaForType(reflect.TypeOf(map[string][]string{}))
		require.Equal(t, "object", schema["type"])
		additionalProps := schema["additionalProperties"].(map[string]interface{})
		require.Equal(t, "array", additionalProps["type"])
	})

	t.Run("nested struct", func(t *testing.T) {
		type Inner struct {
			Value string `json:"value"`
		}
		type Outer struct {
			Inner Inner `json:"inner"`
		}

		schema := buildSchemaForType(reflect.TypeOf(Outer{}))
		require.Equal(t, "object", schema["type"])
		props := schema["properties"].(map[string]interface{})
		innerSchema := props["inner"].(map[string]interface{})
		require.Equal(t, "object", innerSchema["type"])

		innerProps := innerSchema["properties"].(map[string]interface{})
		valueSchema := innerProps["value"].(map[string]interface{})
		require.Equal(t, "string", valueSchema["type"])
	})

	t.Run("pointer to struct", func(t *testing.T) {
		type Simple struct {
			Name string `json:"name"`
		}

		schema := buildSchemaForType(reflect.TypeOf(&Simple{}))
		require.Equal(t, "object", schema["type"])
		props := schema["properties"].(map[string]interface{})
		nameSchema := props["name"].(map[string]interface{})
		require.Equal(t, "string", nameSchema["type"])
	})

	t.Run("pointer to primitive", func(t *testing.T) {
		val := 42
		schema := buildSchemaForType(reflect.TypeOf(&val))
		require.Equal(t, "integer", schema["type"])
	})

	t.Run("struct with omitempty", func(t *testing.T) {
		type WithOmit struct {
			Required string `json:"required"`
			Optional string `json:"optional,omitempty"`
		}

		schema := buildSchemaForType(reflect.TypeOf(WithOmit{}))
		require.Equal(t, "object", schema["type"])
		required := schema["required"].([]string)
		require.Contains(t, required, "required")
		require.NotContains(t, required, "optional")
	})

	t.Run("unknown type defaults to string", func(t *testing.T) {
		// Using a channel as an unknown type
		schema := buildSchemaForType(reflect.TypeOf(make(chan int)))
		require.Equal(t, "string", schema["type"])
	})

	t.Run("slice of structs", func(t *testing.T) {
		type Item struct {
			Name string `json:"name"`
		}

		schema := buildSchemaForType(reflect.TypeOf([]Item{}))
		require.Equal(t, "array", schema["type"])
		items := schema["items"].(map[string]interface{})
		require.Equal(t, "object", items["type"])
		props := items["properties"].(map[string]interface{})
		nameSchema := props["name"].(map[string]interface{})
		require.Equal(t, "string", nameSchema["type"])
	})

	t.Run("deeply nested structure", func(t *testing.T) {
		type Level3 struct {
			Value string `json:"value"`
		}
		type Level2 struct {
			Items []Level3 `json:"items"`
		}
		type Level1 struct {
			Data map[string]Level2 `json:"data"`
		}

		schema := buildSchemaForType(reflect.TypeOf(Level1{}))
		require.Equal(t, "object", schema["type"])

		props := schema["properties"].(map[string]interface{})
		dataSchema := props["data"].(map[string]interface{})
		require.Equal(t, "object", dataSchema["type"])

		additionalProps := dataSchema["additionalProperties"].(map[string]interface{})
		require.Equal(t, "object", additionalProps["type"])

		level2Props := additionalProps["properties"].(map[string]interface{})
		itemsSchema := level2Props["items"].(map[string]interface{})
		require.Equal(t, "array", itemsSchema["type"])
	})
}
