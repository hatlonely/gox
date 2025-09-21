package serializer

import (
	"reflect"
	
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

type Serializer[F, T any] interface {
	Serialize(from F) (T, error)
	Deserialize(to T) (F, error)
}

func NewByteSerializerWithOptions[T any](options *ref.TypeOptions) (Serializer[T, []byte], error) {
	// 注册 serializer 类型
	ref.RegisterT[*JSONSerializer[T]](NewJSONSerializer[T])
	ref.RegisterT[*BSONSerializer[T]](NewBSONSerializer[T])
	ref.RegisterT[*MsgPackSerializer[T]](NewMsgPackSerializer[T])

	ref.RegisterT[JSONSerializer[T]](NewJSONSerializer[T])
	ref.RegisterT[BSONSerializer[T]](NewBSONSerializer[T])
	ref.RegisterT[MsgPackSerializer[T]](NewMsgPackSerializer[T])
	// 注意：ProtobufSerializer 有特殊的类型约束，需要单独处理

	// 处理默认配置
	actualOptions := options
	if actualOptions == nil {
		var t T
		actualOptions = &ref.TypeOptions{
			Namespace: "github.com/hatlonely/gox/kv/serializer",
			Type:      "MsgPackSerializer[" + reflect.TypeOf(t).String() + "]",
		}
	}

	serializer, err := ref.New(actualOptions.Namespace, actualOptions.Type, actualOptions.Options)
	if err != nil {
		return nil, errors.WithMessage(err, "refx.NewT failed")
	}
	if serializer == nil {
		return nil, errors.New("serializer is nil")
	}
	if _, ok := serializer.(Serializer[T, []byte]); !ok {
		return nil, errors.New("serializer is not a Serializer")
	}

	return serializer.(Serializer[T, []byte]), nil
}
