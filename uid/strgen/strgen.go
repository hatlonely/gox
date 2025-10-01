package strgen

import (
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

func init() {
	ref.RegisterT[UUIDGenerator](NewUUIDGeneratorWithOptions)
}

// StringGenerator 生成字符串UID的接口
type StrGenerator interface {
	// Generate 生成一个字符串UID
	Generate() string
}

// NewStrGeneratorWithOptions 创建字符串生成器
func NewStrGeneratorWithOptions(options *ref.TypeOptions) (StrGenerator, error) {
	generator, err := ref.New(options.Namespace, options.Type, options.Options)
	if err != nil {
		return nil, errors.WithMessage(err, "ref.New failed")
	}
	if generator == nil {
		return nil, errors.New("generator is nil")
	}
	if _, ok := generator.(StrGenerator); !ok {
		return nil, errors.New("generator is not a StrGenerator")
	}

	return generator.(StrGenerator), nil
}
