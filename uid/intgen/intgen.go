package intgen

// Int64Generator 生成64位整数UID的接口
type IntGenerator interface {
	// Generate 生成一个64位整数UID
	Generate() int64
}
