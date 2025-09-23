package loader

import (
	"github.com/hatlonely/gox/kv/parser"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

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

func NewLoaderWithOptions[K, V any](options *ref.TypeOptions) (Loader[K, V], error) {
	// 注册 loader 类型
	ref.RegisterT[KVFileLoader[K, V]](NewKVFileLoaderWithOptions[K, V])
	ref.RegisterT[FileTrigger[K, V]](NewFileTriggerWithOptions[K, V])

	loader, err := ref.New(options.Namespace, options.Type, options.Options)
	if err != nil {
		return nil, errors.WithMessage(err, "refx.NewT failed")
	}
	if loader == nil {
		return nil, errors.New("loader is nil")
	}
	if _, ok := loader.(Loader[K, V]); !ok {
		return nil, errors.New("loader is not a Loader")
	}

	return loader.(Loader[K, V]), nil
}
