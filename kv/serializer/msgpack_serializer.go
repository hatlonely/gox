package serializer

import (
	"github.com/vmihailenco/msgpack/v5"
)

type MsgPackSerializer[T any] struct{}

func NewMsgPackSerializer[T any]() *MsgPackSerializer[T] {
	return &MsgPackSerializer[T]{}
}

func (s *MsgPackSerializer[T]) Serialize(from T) ([]byte, error) {
	return msgpack.Marshal(from)
}

func (s *MsgPackSerializer[T]) Deserialize(to []byte) (T, error) {
	var result T
	err := msgpack.Unmarshal(to, &result)
	return result, err
}