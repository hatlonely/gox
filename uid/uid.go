package uid

import (
	"github.com/hatlonely/gox/ref"
	"github.com/hatlonely/gox/uid/intgen"
	"github.com/hatlonely/gox/uid/strgen"
)

// NewIntGeneratorWithOptions 创建整数生成器
func NewIntGeneratorWithOptions(options *ref.TypeOptions) (intgen.IntGenerator, error) {
	return intgen.NewIntGeneratorWithOptions(options)
}

// NewStrGeneratorWithOptions 创建字符串生成器
func NewStrGeneratorWithOptions(options *ref.TypeOptions) (strgen.StrGenerator, error) {
	return strgen.NewStrGeneratorWithOptions(options)
}
