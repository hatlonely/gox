package store

import (
	"context"
	"time"

	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

var (
	ErrKeyNotFound     = errors.New("key not found")
	ErrConditionFailed = errors.New("condition failed")
)

// Store KV 存储相关接口

// setOptions 用于设置 KV 数据时的选项
type setOptions struct {
	Expiration time.Duration
	IfNotExist bool
}

// setOption 用于设置 KV 数据时的选项
type setOption func(*setOptions)

func WithExpiration(expiration time.Duration) setOption {
	return func(options *setOptions) {
		options.Expiration = expiration
	}
}

func WithIfNotExist() setOption {
	return func(options *setOptions) {
		options.IfNotExist = true
	}
}

type Store[K, V any] interface {
	// Set 设置键值对，WithIfNotExist 时键存在则返回 ErrConditionFailed
	Set(ctx context.Context, key K, value V, opts ...setOption) error
	// Get 获取键对应的值，键不存在时返回 ErrKeyNotFound
	Get(ctx context.Context, key K) (V, error)
	// Del 删除键，键不存在时也返回成功
	Del(ctx context.Context, key K) error
	// BatchSet 批量设置，返回每个键的操作结果
	BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error)
	// BatchGet 批量获取，返回每个键的值和错误
	BatchGet(ctx context.Context, keys []K) ([]V, []error, error)
	// BatchDel 批量删除，返回每个键的操作结果
	BatchDel(ctx context.Context, keys []K) ([]error, error)
	Close() error
}

func NewStoreWithOptions[K comparable, V any](options *ref.TypeOptions) (Store[K, V], error) {
	// 注册 store 类型
	ref.RegisterT[*MapStore[K, V]](NewMapStoreWithOptions[K, V])
	ref.RegisterT[*SyncMapStore[K, V]](NewSyncMapStoreWithOptions[K, V])
	ref.RegisterT[*SliceMapStore[K, V]](NewSliceMapStoreWithOptions[K, V])
	ref.RegisterT[*FreeCacheStore[K, V]](NewFreeCacheStoreWithOptions[K, V])
	ref.RegisterT[*BoltDBStore[K, V]](NewBoltDBStoreWithOptions[K, V])
	ref.RegisterT[*RedisStore[K, V]](NewRedisStoreWithOptions[K, V])
	ref.RegisterT[*PebbleStore[K, V]](NewPebbleStoreWithOptions[K, V])
	ref.RegisterT[*LevelDBStore[K, V]](NewLevelDBStoreWithOptions[K, V])
	ref.RegisterT[*LoadableStore[K, V]](NewLoadableStoreWithOptions[K, V])
	ref.RegisterT[*TieredStore[K, V]](NewTieredStoreWithOptions[K, V])
	ref.RegisterT[*ObservableStore[K, V]](NewObservableStoreWithOptions[K, V])

	store, err := ref.New(options.Namespace, options.Type, options.Options)
	if err != nil {
		return nil, errors.WithMessage(err, "refx.NewT failed")
	}
	if store == nil {
		return nil, errors.New("store is nil")
	}
	if _, ok := store.(Store[K, V]); !ok {
		return nil, errors.New("store is not a Store")
	}

	return store.(Store[K, V]), nil
}
