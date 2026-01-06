package tool

import "encoding/json"

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
	WithParameters(json.RawMessage) Tool[T]
	WithExecute(func(T) (string, error)) Tool[T]
}

type tool[T any] struct {
	name        string
	description string
	parameters  json.RawMessage
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

func (t *tool[T]) WithParameters(params json.RawMessage) Tool[T] {
	t.parameters = params
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
	return t.parameters
}

func (t *tool[T]) Call(input json.RawMessage) (string, error) {
	var parsed T
	if err := json.Unmarshal(input, &parsed); err != nil {
		return "", err
	}
	return t.execute(parsed)
}
