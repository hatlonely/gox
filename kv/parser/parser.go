package parser

import (
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
	parser, err := ref.New(options.Namespace, options.Type, options.Options)
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
