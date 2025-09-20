package store

import (
	"context"
	"sync"
)

type SyncMapStore[K comparable, V any] struct {
	m sync.Map
}

func NewSyncMapStoreWithOptions[K comparable, V any]() *SyncMapStore[K, V] {
	return &SyncMapStore[K, V]{}
}

func (s *SyncMapStore[K, V]) Set(ctx context.Context, key K, value V, opts ...setOption) error {
	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.IfNotExist {
		_, loaded := s.m.LoadOrStore(key, value)
		if loaded {
			return ErrConditionFailed
		}
		return nil
	}

	s.m.Store(key, value)
	return nil
}

func (s *SyncMapStore[K, V]) Get(ctx context.Context, key K) (V, error) {
	value, ok := s.m.Load(key)
	if !ok {
		var zero V
		return zero, ErrKeyNotFound
	}
	return value.(V), nil
}

func (s *SyncMapStore[K, V]) Del(ctx context.Context, key K) error {
	s.m.Delete(key)
	return nil
}

func (s *SyncMapStore[K, V]) BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error) {
	if len(keys) != len(vals) {
		return nil, ErrConditionFailed
	}

	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	errors := make([]error, len(keys))
	for i, key := range keys {
		if options.IfNotExist {
			_, loaded := s.m.LoadOrStore(key, vals[i])
			if loaded {
				errors[i] = ErrConditionFailed
				continue
			}
		} else {
			s.m.Store(key, vals[i])
		}
	}

	return errors, nil
}

func (s *SyncMapStore[K, V]) BatchGet(ctx context.Context, keys []K) ([]V, []error, error) {
	values := make([]V, len(keys))
	errors := make([]error, len(keys))

	for i, key := range keys {
		value, ok := s.m.Load(key)
		if !ok {
			var zero V
			values[i] = zero
			errors[i] = ErrKeyNotFound
		} else {
			values[i] = value.(V)
		}
	}

	return values, errors, nil
}

func (s *SyncMapStore[K, V]) BatchDel(ctx context.Context, keys []K) ([]error, error) {
	errors := make([]error, len(keys))
	for _, key := range keys {
		s.m.Delete(key)
	}

	return errors, nil
}

func (s *SyncMapStore[K, V]) Close() error {
	s.m.Range(func(key, value any) bool {
		s.m.Delete(key)
		return true
	})
	return nil
}