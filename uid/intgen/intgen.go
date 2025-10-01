package intgen

import "github.com/hatlonely/gox/ref"

func init() {
	ref.RegisterT[TimestampSeqGenerator](NewTimestampSeqGenerator)
	ref.RegisterT[SnowflakeGenerator](NewSnowflakeGeneratorWithOptions)
}

// IntGenerator 生成64位整数UID的接口
type IntGenerator interface {
	// Generate 生成一个64位整数UID
	Generate() int64
}
