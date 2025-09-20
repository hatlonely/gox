package store

import "context"

type MapStore[K comparable, V any] struct {
	m map[K]V
}

func NewMapStoreWithOptions[K comparable, V any]() *MapStore[K, V] {
	return &MapStore[K, V]{
		m: make(map[K]V),
	}
}

func (s *MapStore[K, V]) Set(ctx context.Context, key K, value V, opts ...setOption) error {
	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.IfNotExist {
		if _, exists := s.m[key]; exists {
			return ErrConditionFailed
		}
	}

	s.m[key] = value
	return nil
}

func (s *MapStore[K, V]) Get(ctx context.Context, key K) (V, error) {
	value, exists := s.m[key]
	if !exists {
		var zero V
		return zero, ErrKeyNotFound
	}
	return value, nil
}

func (s *MapStore[K, V]) Del(ctx context.Context, key K) error {
	delete(s.m, key)
	return nil
}

func (s *MapStore[K, V]) BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error) {
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
			if _, exists := s.m[key]; exists {
				errors[i] = ErrConditionFailed
				continue
			}
		}
		s.m[key] = vals[i]
	}

	return errors, nil
}

func (s *MapStore[K, V]) BatchGet(ctx context.Context, keys []K) ([]V, []error, error) {
	values := make([]V, len(keys))
	errors := make([]error, len(keys))

	for i, key := range keys {
		value, exists := s.m[key]
		if !exists {
			var zero V
			values[i] = zero
			errors[i] = ErrKeyNotFound
		} else {
			values[i] = value
		}
	}

	return values, errors, nil
}

func (s *MapStore[K, V]) BatchDel(ctx context.Context, keys []K) ([]error, error) {
	errors := make([]error, len(keys))
	for _, key := range keys {
		delete(s.m, key)
	}

	return errors, nil
}

func (s *MapStore[K, V]) Close() error {
	s.m = nil
	return nil
}
