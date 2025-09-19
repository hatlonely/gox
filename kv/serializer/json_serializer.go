package serializer

import (
	"encoding/json"
)

type JSONSerializer[T any] struct{}

func NewJSONSerializer[T any]() *JSONSerializer[T] {
	return &JSONSerializer[T]{}
}

func (s *JSONSerializer[T]) Serialize(from T) ([]byte, error) {
	return json.Marshal(from)
}

func (s *JSONSerializer[T]) Deserialize(to []byte) (T, error) {
	var result T
	err := json.Unmarshal(to, &result)
	return result, err
}
