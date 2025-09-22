package store

import (
	"context"
	"sync"

	"github.com/hatlonely/gox/kv/loader"
	"github.com/hatlonely/gox/kv/parser"
	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/log/logger"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

func NewLoadableStoreWithOptions[K comparable, V any](options *LoadableStoreOptions) (*LoadableStore[K, V], error) {
	if options == nil {
		return nil, errors.New("options is nil")
	}

	if options.LoadStrategy != loader.LoadStrategyInPlace && options.LoadStrategy != loader.LoadStrategyReplace {
		return nil, errors.Errorf("invalid load strategy: %s", options.LoadStrategy)
	}

	// 创建底层 store
	store, err := NewStoreWithOptions[K, V](options.Store)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create store")
	}

	// 创建 loader
	loaderInstance, err := loader.NewLoaderWithOptions[K, V](options.Loader)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create loader")
	}

	// 创建 logger
	l, err := log.NewLoggerWithOptions(options.Logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logger")
	}
	l = l.WithGroup("loadableStore")

	loadableStore := &LoadableStore[K, V]{
		store:        store,
		loader:       loaderInstance,
		loadStrategy: options.LoadStrategy,
		storeOptions: options.Store, // 保存原始配置
		logger:       l,
		closed:       false,
	}

	// 注册数据变更监听器，首次加载数据
	err = loaderInstance.OnChange(loadableStore.handleDataChange)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to register data change listener")
	}

	return loadableStore, nil
}

type LoadableStoreOptions struct {
	Store        *ref.TypeOptions `cfg:"store" validate:"required"`
	Loader       *ref.TypeOptions `cfg:"loader" validate:"required"`
	LoadStrategy string           `cfg:"loadStrategy" def:"inplace"` // "inplace" 或 "replace"
	Logger       *ref.TypeOptions `cfg:"logger"`
}

type LoadableStore[K comparable, V any] struct {
	store         Store[K, V]
	loader        loader.Loader[K, V]
	loadStrategy  string
	storeOptions  *ref.TypeOptions // 保存原始 store 配置用于 replace 策略
	mutex         sync.RWMutex
	logger        logger.Logger
	closed        bool
}

// Store 接口实现
func (ls *LoadableStore[K, V]) Set(ctx context.Context, key K, value V, opts ...setOption) error {
	if ls.closed {
		return errors.New("store is closed")
	}
	
	if ls.loadStrategy == loader.LoadStrategyReplace {
		ls.mutex.RLock()
		defer ls.mutex.RUnlock()
	}
	
	return ls.store.Set(ctx, key, value, opts...)
}

func (ls *LoadableStore[K, V]) Get(ctx context.Context, key K) (V, error) {
	if ls.closed {
		var zero V
		return zero, errors.New("store is closed")
	}
	
	if ls.loadStrategy == loader.LoadStrategyReplace {
		ls.mutex.RLock()
		defer ls.mutex.RUnlock()
	}
	
	return ls.store.Get(ctx, key)
}

func (ls *LoadableStore[K, V]) Del(ctx context.Context, key K) error {
	if ls.closed {
		return errors.New("store is closed")
	}
	
	if ls.loadStrategy == loader.LoadStrategyReplace {
		ls.mutex.RLock()
		defer ls.mutex.RUnlock()
	}
	
	return ls.store.Del(ctx, key)
}

func (ls *LoadableStore[K, V]) BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error) {
	if ls.closed {
		return nil, errors.New("store is closed")
	}
	
	if ls.loadStrategy == loader.LoadStrategyReplace {
		ls.mutex.RLock()
		defer ls.mutex.RUnlock()
	}
	
	return ls.store.BatchSet(ctx, keys, vals, opts...)
}

func (ls *LoadableStore[K, V]) BatchGet(ctx context.Context, keys []K) ([]V, []error, error) {
	if ls.closed {
		return nil, nil, errors.New("store is closed")
	}
	
	if ls.loadStrategy == loader.LoadStrategyReplace {
		ls.mutex.RLock()
		defer ls.mutex.RUnlock()
	}
	
	return ls.store.BatchGet(ctx, keys)
}

func (ls *LoadableStore[K, V]) BatchDel(ctx context.Context, keys []K) ([]error, error) {
	if ls.closed {
		return nil, errors.New("store is closed")
	}
	
	if ls.loadStrategy == loader.LoadStrategyReplace {
		ls.mutex.RLock()
		defer ls.mutex.RUnlock()
	}
	
	return ls.store.BatchDel(ctx, keys)
}

func (ls *LoadableStore[K, V]) Close() error {
	if ls.closed {
		return nil
	}
	
	ls.closed = true
	
	var errs []error
	if err := ls.loader.Close(); err != nil {
		errs = append(errs, errors.WithMessage(err, "failed to close loader"))
	}
	
	if err := ls.store.Close(); err != nil {
		errs = append(errs, errors.WithMessage(err, "failed to close store"))
	}
	
	if len(errs) > 0 {
		return errors.Errorf("close errors: %v", errs)
	}
	
	return nil
}

// handleDataChange 处理数据变更事件
func (ls *LoadableStore[K, V]) handleDataChange(stream loader.KVStream[K, V]) error {
	if ls.closed {
		return errors.New("store is closed")
	}

	switch ls.loadStrategy {
	case loader.LoadStrategyInPlace:
		return ls.handleInPlaceLoad(stream)
	case loader.LoadStrategyReplace:
		return ls.handleReplaceLoad(stream)
	default:
		return errors.Errorf("unknown load strategy: %s", ls.loadStrategy)
	}
}

// handleInPlaceLoad 处理增量加载策略
func (ls *LoadableStore[K, V]) handleInPlaceLoad(stream loader.KVStream[K, V]) error {
	ctx := context.Background()
	
	return stream.Each(func(changeType parser.ChangeType, key K, val V) error {
		switch changeType {
		case parser.ChangeTypeAdd, parser.ChangeTypeUpdate:
			return ls.store.Set(ctx, key, val)
		case parser.ChangeTypeDelete:
			return ls.store.Del(ctx, key)
		case parser.ChangeTypeUnknown:
			// 对于未知类型，默认当做更新处理
			ls.logger.Warn("unknown change type, treating as update", "key", key)
			return ls.store.Set(ctx, key, val)
		default:
			return errors.Errorf("unsupported change type: %d", changeType)
		}
	})
}

// handleReplaceLoad 处理替换加载策略
func (ls *LoadableStore[K, V]) handleReplaceLoad(stream loader.KVStream[K, V]) error {
	ctx := context.Background()
	
	// 创建新的 store 实例
	newStore, err := NewStoreWithOptions[K, V](ls.getStoreOptions())
	if err != nil {
		return errors.WithMessage(err, "failed to create new store")
	}
	
	// 加载所有数据到新 store
	err = stream.Each(func(changeType parser.ChangeType, key K, val V) error {
		switch changeType {
		case parser.ChangeTypeAdd, parser.ChangeTypeUpdate:
			return newStore.Set(ctx, key, val)
		case parser.ChangeTypeDelete:
			// Replace 策略中删除操作意味着这个键不会出现在新 store 中
			// 所以这里什么都不做
			return nil
		case parser.ChangeTypeUnknown:
			// 对于未知类型，默认当做更新处理
			ls.logger.Warn("unknown change type, treating as update", "key", key)
			return newStore.Set(ctx, key, val)
		default:
			return errors.Errorf("unsupported change type: %d", changeType)
		}
	})
	
	if err != nil {
		newStore.Close()
		return errors.WithMessage(err, "failed to load data to new store")
	}
	
	// 原子性替换 store
	ls.mutex.Lock()
	oldStore := ls.store
	ls.store = newStore
	ls.mutex.Unlock()
	
	// 关闭旧的 store
	if err := oldStore.Close(); err != nil {
		ls.logger.Warn("failed to close old store", "error", err)
	}
	
	ls.logger.Info("store replaced successfully")
	return nil
}

// getStoreOptions 获取当前 store 的配置选项
func (ls *LoadableStore[K, V]) getStoreOptions() *ref.TypeOptions {
	return ls.storeOptions
}