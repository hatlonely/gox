package store

import (
	"context"

	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

type TieredStoreOptions struct {
	// Tiers 多级存储层配置，按优先级从高到低排列
	// 第一层应该是最快的缓存（如内存），最后一层是持久化存储
	// 至少需要配置一层存储
	Tiers []*ref.TypeOptions `cfg:"tiers" validate:"required,min=1,dive,required"`

	// WritePolicy 写入策略，支持以下两种模式：
	// - "writeThrough": 写穿模式，同步写入所有层，保证数据一致性但性能较低
	// - "writeBack": 写回模式，只写入第一层，异步写入其他层，性能较高但可能存在数据丢失风险
	WritePolicy string `cfg:"writePolicy" def:"writeThrough" validate:"oneof=writeThrough writeBack"`

	// Promote 数据提升策略，控制是否将从下层读取的数据提升到上层缓存
	// - true: 启用提升，从下层读取数据后会异步写入到上层缓存，提高后续访问性能
	// - false: 禁用提升，数据只在原层访问，适用于访问模式相对固定的场景
	Promote bool `cfg:"promote" def:"true"`
}

// TieredStore 多级缓存存储实现
// 支持多层级的缓存架构，提供数据提升和不同的写入策略
type TieredStore[K comparable, V any] struct {
	tiers       []Store[K, V]
	writePolicy string
	promote     bool
}

func NewTieredStoreWithOptions[K comparable, V any](options *TieredStoreOptions) (*TieredStore[K, V], error) {
	if options == nil {
		return nil, errors.New("options is nil")
	}

	if len(options.Tiers) == 0 {
		return nil, errors.New("at least one tier is required")
	}

	if options.WritePolicy != "writeThrough" && options.WritePolicy != "writeBack" {
		return nil, errors.Errorf("invalid write policy: %s", options.WritePolicy)
	}

	tiers := make([]Store[K, V], 0, len(options.Tiers))
	for i, tierOptions := range options.Tiers {
		tier, err := NewStoreWithOptions[K, V](tierOptions)
		if err != nil {
			// 清理已创建的 tiers
			for _, createdTier := range tiers {
				createdTier.Close()
			}
			return nil, errors.WithMessagef(err, "failed to create tier %d", i)
		}
		tiers = append(tiers, tier)
	}

	return &TieredStore[K, V]{
		tiers:       tiers,
		writePolicy: options.WritePolicy,
		promote:     options.Promote,
	}, nil
}

func (ts *TieredStore[K, V]) Set(ctx context.Context, key K, value V, opts ...setOption) error {
	if len(ts.tiers) == 0 {
		return errors.New("no tiers available")
	}

	switch ts.writePolicy {
	case "writeThrough":
		return ts.writeThrough(ctx, key, value, opts...)
	case "writeBack":
		return ts.writeBack(ctx, key, value, opts...)
	default:
		return errors.Errorf("unknown write policy: %s", ts.writePolicy)
	}
}

func (ts *TieredStore[K, V]) Get(ctx context.Context, key K) (V, error) {
	var zero V

	for i, tier := range ts.tiers {
		value, err := tier.Get(ctx, key)
		if err == nil {
			// 找到数据，如果启用提升且不是第一层，则异步提升到上层缓存
			if ts.promote && i > 0 {
				go ts.promoteToUpperTiers(ctx, key, value, i)
			}
			return value, nil
		}
		if err != ErrKeyNotFound {
			// 记录错误但继续尝试下一层
			continue
		}
	}

	return zero, ErrKeyNotFound
}

func (ts *TieredStore[K, V]) Del(ctx context.Context, key K) error {
	var lastErr error
	for _, tier := range ts.tiers {
		if err := tier.Del(ctx, key); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (ts *TieredStore[K, V]) BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error) {
	if len(keys) != len(vals) {
		return nil, errors.New("keys and vals length mismatch")
	}

	errs := make([]error, len(keys))
	for i := range keys {
		errs[i] = ts.Set(ctx, keys[i], vals[i], opts...)
	}
	return errs, nil
}

func (ts *TieredStore[K, V]) BatchGet(ctx context.Context, keys []K) ([]V, []error, error) {
	vals := make([]V, len(keys))
	errs := make([]error, len(keys))

	for i, key := range keys {
		val, err := ts.Get(ctx, key)
		vals[i] = val
		errs[i] = err
	}

	return vals, errs, nil
}

func (ts *TieredStore[K, V]) BatchDel(ctx context.Context, keys []K) ([]error, error) {
	errs := make([]error, len(keys))
	for i, key := range keys {
		errs[i] = ts.Del(ctx, key)
	}
	return errs, nil
}

func (ts *TieredStore[K, V]) Close() error {
	var errs []error
	for i, tier := range ts.tiers {
		if err := tier.Close(); err != nil {
			errs = append(errs, errors.WithMessagef(err, "failed to close tier %d", i))
		}
	}

	if len(errs) > 0 {
		return errors.Errorf("close errors: %v", errs)
	}
	return nil
}

// writeThrough 写穿策略：同步写入所有层
func (ts *TieredStore[K, V]) writeThrough(ctx context.Context, key K, value V, opts ...setOption) error {
	var lastErr error
	success := false

	for _, tier := range ts.tiers {
		if err := tier.Set(ctx, key, value, opts...); err != nil {
			lastErr = err
			// 如果是条件失败，直接返回，不继续尝试其他层
			if err == ErrConditionFailed {
				return err
			}
		} else {
			success = true
		}
	}

	if !success {
		return lastErr
	}
	return nil
}

// writeBack 写回策略：只写入第一层，后续异步写入下层
func (ts *TieredStore[K, V]) writeBack(ctx context.Context, key K, value V, opts ...setOption) error {
	// 先写入第一层
	if err := ts.tiers[0].Set(ctx, key, value, opts...); err != nil {
		return err
	}

	// 异步写入其他层
	if len(ts.tiers) > 1 {
		go ts.writeToLowerTiers(context.Background(), key, value, opts...)
	}

	return nil
}

// promoteToUpperTiers 异步提升数据到上层缓存
func (ts *TieredStore[K, V]) promoteToUpperTiers(ctx context.Context, key K, value V, fromTier int) {
	// 从找到数据的层开始，往上层写入
	for i := fromTier - 1; i >= 0; i-- {
		if tier := ts.tiers[i]; tier != nil {
			tier.Set(ctx, key, value)
		}
	}
}

// writeToLowerTiers 异步写入下层存储
func (ts *TieredStore[K, V]) writeToLowerTiers(ctx context.Context, key K, value V, opts ...setOption) {
	// 从第二层开始写入
	for i := 1; i < len(ts.tiers); i++ {
		if tier := ts.tiers[i]; tier != nil {
			tier.Set(ctx, key, value, opts...)
		}
	}
}

// GetTierCount 返回当前层数
func (ts *TieredStore[K, V]) GetTierCount() int {
	return len(ts.tiers)
}

// GetFromTier 从指定层获取数据 (用于测试和监控)
func (ts *TieredStore[K, V]) GetFromTier(ctx context.Context, tier int, key K) (V, error) {
	var zero V
	if tier < 0 || tier >= len(ts.tiers) {
		return zero, errors.Errorf("invalid tier index: %d", tier)
	}

	return ts.tiers[tier].Get(ctx, key)
}

