package intgen

import (
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

func init() {
	ref.RegisterT[TimestampSeqGenerator](NewTimestampSeqGenerator)
	ref.RegisterT[SnowflakeGenerator](NewSnowflakeGeneratorWithOptions)
	ref.RegisterT[RedisGenerator](NewRedisGeneratorWithOptions)
}

// IntGenerator 生成64位整数UID的接口
type IntGenerator interface {
	// Generate 生成一个64位整数UID
	Generate() int64
}

// NewIntGeneratorWithOptions 创建整数生成器
func NewIntGeneratorWithOptions(options *ref.TypeOptions) (IntGenerator, error) {
	generator, err := ref.New(options.Namespace, options.Type, options.Options)
	if err != nil {
		return nil, errors.WithMessage(err, "ref.New failed")
	}
	if generator == nil {
		return nil, errors.New("generator is nil")
	}
	if _, ok := generator.(IntGenerator); !ok {
		return nil, errors.New("generator is not an IntGenerator")
	}

	return generator.(IntGenerator), nil
}
