package strgen

import "github.com/hatlonely/gox/ref"

func init() {
	ref.RegisterT[UUIDGenerator](NewUUIDGeneratorWithOptions)
}

// StringGenerator 生成字符串UID的接口
type StrGenerator interface {
	// Generate 生成一个字符串UID
	Generate() string
}
