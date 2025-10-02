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

// NewIntGenerator 创建默认的整数生成器（Snowflake算法）
func NewIntGenerator() intgen.IntGenerator {
	return intgen.NewSnowflakeGeneratorWithOptions(nil)
}

// NewStrGenerator 创建默认的字符串生成器（UUID v7）
func NewStrGenerator() strgen.StrGenerator {
	return strgen.NewUUIDGeneratorWithOptions(&strgen.UUIDOptions{Version: "v7"})
}
