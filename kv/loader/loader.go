package loader

import "github.com/hatlonely/gox/kv/parser"

const (
	LoadStrategyReplace = "replace"
	LoadStrategyInPlace = "inplace"
)

// KVStream 用于遍历 KV 数据流
type KVStream[K, V any] interface {
	Each(func(changeType parser.ChangeType, key K, val V) error) error
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
