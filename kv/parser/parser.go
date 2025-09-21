package parser

import (
	"reflect"

	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

// ChangeType 数据加载时数据的变更类型
type ChangeType int

const (
	ChangeTypeUnknown ChangeType = 0    // 未知
	ChangeTypeAdd     ChangeType = iota // 新增
	ChangeTypeUpdate                    // 更新
	ChangeTypeDelete                    // 删除
)

type Parser[K, V any] interface {
	Parse(buf []byte) (ChangeType, K, V, error)
}

func NewParserWithOptions[K, V any](options *ref.TypeOptions) (Parser[K, V], error) {
	// 注册 parser 类型
	ref.RegisterT[LineParser[K, V]](NewLineParserWithOptions[K, V])
	ref.RegisterT[JsonParser[K, V]](NewJsonParserWithOptions[K, V])
	ref.RegisterT[BsonParser[K, V]](NewBsonParserWithOptions[K, V])

	// 处理默认配置
	actualOptions := options
	if actualOptions == nil {
		var k K
		var v V
		actualOptions = &ref.TypeOptions{
			Namespace: "github.com/hatlonely/gox/kv/parser",
			Type:      "LineParser[" + reflect.TypeOf(k).String() + "," + reflect.TypeOf(v).String() + "]",
		}
	}

	parser, err := ref.New(actualOptions.Namespace, actualOptions.Type, actualOptions.Options)
	if err != nil {
		return nil, errors.WithMessage(err, "refx.NewT failed")
	}
	if parser == nil {
		return nil, errors.New("parser is nil")
	}
	if _, ok := parser.(Parser[K, V]); !ok {
		return nil, errors.New("parser is not a Parser")
	}

	return parser.(Parser[K, V]), nil
}
