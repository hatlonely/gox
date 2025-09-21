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

type LineParser[K, V any] interface {
	Parse(line string) (ChangeType, K, V, error)
}

func NewLineParserWithOptions[K, V any](options *ref.TypeOptions) (LineParser[K, V], error) {
	parser, err := ref.NewT[LineParser[K, V]](options)
	if err != nil {
		return nil, errors.WithMessage(err, "refx.NewT failed")
	}

	return parser, nil
}
