package serializer

import (
	"google.golang.org/protobuf/proto"
)

type ProtobufSerializer[T proto.Message] struct{}

func NewProtobufSerializer[T proto.Message]() *ProtobufSerializer[T] {
	return &ProtobufSerializer[T]{}
}

func (s *ProtobufSerializer[T]) Serialize(from T) ([]byte, error) {
	return proto.Marshal(from)
}

func (s *ProtobufSerializer[T]) Deserialize(to []byte) (T, error) {
	var result T
	if err := proto.Unmarshal(to, result); err != nil {
		return result, err
	}
	return result, nil
}