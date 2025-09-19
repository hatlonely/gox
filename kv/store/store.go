package store

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

var (
	ErrKeyNotFound = errors.New("key not found")
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
	Set(ctx context.Context, key K, value V, opts ...setOption) error
	Get(ctx context.Context, key K) (V, error)
	Del(ctx context.Context, key K) error
	BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error)
	BatchGet(ctx context.Context, keys []K) ([]V, []error, error)
	BatchDel(ctx context.Context, keys []K) ([]error, error)
	Close() error
}
