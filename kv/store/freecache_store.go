package store

import (
	"context"
	"errors"

	"github.com/coocood/freecache"
	"github.com/hatlonely/gox/kv/serializer"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

type FreeCacheStore[K, V any] struct {
	cache           *freecache.Cache
	keySerializer   serializer.Serializer[K, []byte]
	valueSerializer serializer.Serializer[V, []byte]
}

func NewFreeCacheStore[K, V any](
	size int,
	keySerializer serializer.Serializer[K, []byte],
	valueSerializer serializer.Serializer[V, []byte],
) *FreeCacheStore[K, V] {
	return &FreeCacheStore[K, V]{
		cache:           freecache.NewCache(size),
		keySerializer:   keySerializer,
		valueSerializer: valueSerializer,
	}
}

func (s *FreeCacheStore[K, V]) Set(ctx context.Context, key K, value V, opts ...setOption) error {
	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	keyBytes, err := s.keySerializer.Serialize(key)
	if err != nil {
		return err
	}

	valueBytes, err := s.valueSerializer.Serialize(value)
	if err != nil {
		return err
	}

	if options.IfNotExist {
		if _, err := s.cache.Get(keyBytes); err == nil {
			return nil
		}
	}

	expireSeconds := int(options.Expiration.Seconds())
	return s.cache.Set(keyBytes, valueBytes, expireSeconds)
}

func (s *FreeCacheStore[K, V]) Get(ctx context.Context, key K) (V, error) {
	var zero V

	keyBytes, err := s.keySerializer.Serialize(key)
	if err != nil {
		return zero, err
	}

	valueBytes, err := s.cache.Get(keyBytes)
	if err != nil {
		return zero, ErrKeyNotFound
	}

	return s.valueSerializer.Deserialize(valueBytes)
}

func (s *FreeCacheStore[K, V]) Del(ctx context.Context, key K) error {
	keyBytes, err := s.keySerializer.Serialize(key)
	if err != nil {
		return err
	}

	s.cache.Del(keyBytes)
	return nil
}

func (s *FreeCacheStore[K, V]) BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error) {
	if len(keys) != len(vals) {
		return nil, errors.New("keys and vals length mismatch")
	}

	errs := make([]error, len(keys))
	for i := range keys {
		errs[i] = s.Set(ctx, keys[i], vals[i], opts...)
	}
	return errs, nil
}

func (s *FreeCacheStore[K, V]) BatchGet(ctx context.Context, keys []K) ([]V, []error, error) {
	vals := make([]V, len(keys))
	errs := make([]error, len(keys))

	for i, key := range keys {
		val, err := s.Get(ctx, key)
		vals[i] = val
		errs[i] = err
	}
	return vals, errs, nil
}

func (s *FreeCacheStore[K, V]) BatchDel(ctx context.Context, keys []K) ([]error, error) {
	errs := make([]error, len(keys))
	for i, key := range keys {
		errs[i] = s.Del(ctx, key)
	}
	return errs, nil
}

func (s *FreeCacheStore[K, V]) Close() error {
	s.cache.Clear()
	return nil
}
