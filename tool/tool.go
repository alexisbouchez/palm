package tool

import (
	"encoding/json"
	"reflect"
	"strings"
)

type Callable interface {
	GetName() string
	GetDescription() string
	GetParameters() json.RawMessage
	Call(input json.RawMessage) (string, error)
}

type Tool[T any] interface {
	Callable
	WithName(string) Tool[T]
	WithDescription(string) Tool[T]
	WithExecute(func(T) (string, error)) Tool[T]
}

type tool[T any] struct {
	name        string
	description string
	execute     func(input T) (string, error)
}

func New[T any]() Tool[T] {
	return &tool[T]{}
}

func (t *tool[T]) WithName(name string) Tool[T] {
	t.name = name
	return t
}

func (t *tool[T]) WithDescription(desc string) Tool[T] {
	t.description = desc
	return t
}

func (t *tool[T]) WithExecute(fn func(input T) (string, error)) Tool[T] {
	t.execute = fn
	return t
}

func (t *tool[T]) GetName() string {
	return t.name
}

func (t *tool[T]) GetDescription() string {
	return t.description
}

func (t *tool[T]) GetParameters() json.RawMessage {
	var input T
	schema := schemaFromType(reflect.TypeOf(input))
	b, _ := json.Marshal(schema)
	return b
}

func (t *tool[T]) Call(input json.RawMessage) (string, error) {
	var parsed T
	if err := json.Unmarshal(input, &parsed); err != nil {
		return "", err
	}
	return t.execute(parsed)
}

func schemaFromType(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := map[string]any{"type": "object"}
	props := map[string]any{}
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		name := field.Tag.Get("json")
		if idx := strings.Index(name, ","); idx != -1 {
			name = name[:idx]
		}
		if name == "" || name == "-" {
			name = strings.ToLower(field.Name)
		}

		prop := map[string]any{}

		switch field.Type.Kind() {
		case reflect.String:
			prop["type"] = "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			prop["type"] = "integer"
		case reflect.Float32, reflect.Float64:
			prop["type"] = "number"
		case reflect.Bool:
			prop["type"] = "boolean"
		case reflect.Slice:
			prop["type"] = "array"
			if elem := field.Type.Elem(); elem.Kind() == reflect.Struct {
				prop["items"] = schemaFromType(elem)
			} else {
				prop["items"] = schemaFromType(elem)
			}
		case reflect.Struct:
			prop = schemaFromType(field.Type)
		}

		if desc := field.Tag.Get("description"); desc != "" {
			prop["description"] = desc
		}

		if field.Tag.Get("required") == "true" {
			required = append(required, name)
		}

		props[name] = prop
	}

	schema["properties"] = props
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
