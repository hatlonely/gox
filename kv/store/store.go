package store

import (
	"context"
	"time"

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
