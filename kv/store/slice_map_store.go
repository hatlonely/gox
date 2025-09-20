package store

import (
	"context"
	"sync"
)

type SliceMapStoreOptions struct {
	N int `cfg:"n" def:"1024"`
}

type SliceMapStore[K comparable, V any] struct {
	s    []V
	m    map[K]int
	free []int
	mu   sync.RWMutex
}

func NewSliceMapStoreWithOptions[K comparable, V any](options *SliceMapStoreOptions) *SliceMapStore[K, V] {
	if options == nil {
		options = &SliceMapStoreOptions{N: 1024}
	}
	return &SliceMapStore[K, V]{
		s:    make([]V, options.N),
		m:    make(map[K]int),
		free: make([]int, 0, options.N),
	}
}

func (s *SliceMapStore[K, V]) Set(ctx context.Context, key K, value V, opts ...setOption) error {
	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if idx, exists := s.m[key]; exists {
		if options.IfNotExist {
			return ErrConditionFailed
		}
		s.s[idx] = value
		return nil
	}

	// 获取可用索引
	var idx int
	if len(s.free) > 0 {
		idx = s.free[len(s.free)-1]
		s.free = s.free[:len(s.free)-1]
	} else {
		if len(s.m) >= len(s.s) {
			return ErrConditionFailed // 容量已满
		}
		idx = len(s.m)
	}

	s.s[idx] = value
	s.m[key] = idx
	return nil
}

func (s *SliceMapStore[K, V]) Get(ctx context.Context, key K) (V, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, exists := s.m[key]
	if !exists {
		var zero V
		return zero, ErrKeyNotFound
	}
	return s.s[idx], nil
}

func (s *SliceMapStore[K, V]) Del(ctx context.Context, key K) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, exists := s.m[key]
	if !exists {
		return nil
	}

	delete(s.m, key)
	s.free = append(s.free, idx)
	
	// 清零删除的位置
	var zero V
	s.s[idx] = zero
	
	return nil
}

func (s *SliceMapStore[K, V]) BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error) {
	if len(keys) != len(vals) {
		return nil, ErrConditionFailed
	}

	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	errors := make([]error, len(keys))
	for i, key := range keys {
		if idx, exists := s.m[key]; exists {
			if options.IfNotExist {
				errors[i] = ErrConditionFailed
				continue
			}
			s.s[idx] = vals[i]
			continue
		}

		// 获取可用索引
		var idx int
		if len(s.free) > 0 {
			idx = s.free[len(s.free)-1]
			s.free = s.free[:len(s.free)-1]
		} else {
			if len(s.m) >= len(s.s) {
				errors[i] = ErrConditionFailed // 容量已满
				continue
			}
			idx = len(s.m)
		}

		s.s[idx] = vals[i]
		s.m[key] = idx
	}

	return errors, nil
}

func (s *SliceMapStore[K, V]) BatchGet(ctx context.Context, keys []K) ([]V, []error, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	values := make([]V, len(keys))
	errors := make([]error, len(keys))

	for i, key := range keys {
		idx, exists := s.m[key]
		if !exists {
			var zero V
			values[i] = zero
			errors[i] = ErrKeyNotFound
		} else {
			values[i] = s.s[idx]
		}
	}

	return values, errors, nil
}

func (s *SliceMapStore[K, V]) BatchDel(ctx context.Context, keys []K) ([]error, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	errors := make([]error, len(keys))
	for _, key := range keys {
		idx, exists := s.m[key]
		if exists {
			delete(s.m, key)
			s.free = append(s.free, idx)
			
			// 清零删除的位置
			var zero V
			s.s[idx] = zero
		}
	}

	return errors, nil
}

func (s *SliceMapStore[K, V]) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.s = nil
	s.m = nil
	s.free = nil
	return nil
}