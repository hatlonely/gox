package strgen

// StringGenerator 生成字符串UID的接口
type StrGenerator interface {
	// Generate 生成一个字符串UID
	Generate() string
}
