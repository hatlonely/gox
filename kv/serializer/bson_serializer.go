package serializer

import (
	"go.mongodb.org/mongo-driver/bson"
)

type BSONSerializer[T any] struct{}

func NewBSONSerializer[T any]() *BSONSerializer[T] {
	return &BSONSerializer[T]{}
}

func (s *BSONSerializer[T]) Serialize(from T) ([]byte, error) {
	return bson.Marshal(from)
}

func (s *BSONSerializer[T]) Deserialize(to []byte) (T, error) {
	var result T
	err := bson.Unmarshal(to, &result)
	return result, err
}