package loader

const (
	LoadStrategyReplace = "replace"
	LoadStrategyInPlace = "inplace"
)

// ChangeType 数据加载时数据的变更类型
type ChangeType int

const (
	ChangeTypeUnknown ChangeType = 0    // 未知
	ChangeTypeAdd     ChangeType = iota // 新增
	ChangeTypeUpdate                    // 更新
	ChangeTypeDelete                    // 删除
)

// KVStream 用于遍历 KV 数据流
type KVStream[K, V any] interface {
	Each(func(changeType ChangeType, key K, val V) error) error
}

// Listener 用于监听 KV 数据变更
type Listener[K, V any] func(stream KVStream[K, V]) error

// Loader 用于加载 KV 数据
type Loader[K, V any] interface {
	// OnChange 注册数据变更监听
	OnChange(listener Listener[K, V]) error
	// Close 关闭 Loader
	Close() error
}
