package store

import (
	"context"
	"errors"
	"reflect"

	"github.com/coocood/freecache"
	"github.com/hatlonely/gox/kv/serializer"
	"github.com/hatlonely/gox/ref"
)

type FreeCacheStoreOptions struct {
	Size          int `cfg:"size"`
	KeySerializer *ref.TypeOptions
	ValSerializer *ref.TypeOptions
}

type FreeCacheStore[K, V any] struct {
	cache           *freecache.Cache
	keySerializer   serializer.Serializer[K, []byte]
	valueSerializer serializer.Serializer[V, []byte]
}

func NewFreeCacheStoreWithOptions[K, V any](options *FreeCacheStoreOptions) (*FreeCacheStore[K, V], error) {
	// 注册当前泛型类型的序列化器
	ref.RegisterT[*serializer.JSONSerializer[K]](func() *serializer.JSONSerializer[K] {
		return serializer.NewJSONSerializer[K]()
	})
	ref.RegisterT[*serializer.MsgPackSerializer[K]](func() *serializer.MsgPackSerializer[K] {
		return serializer.NewMsgPackSerializer[K]()
	})
	ref.RegisterT[*serializer.BSONSerializer[K]](func() *serializer.BSONSerializer[K] {
		return serializer.NewBSONSerializer[K]()
	})

	ref.RegisterT[*serializer.JSONSerializer[V]](func() *serializer.JSONSerializer[V] {
		return serializer.NewJSONSerializer[V]()
	})
	ref.RegisterT[*serializer.MsgPackSerializer[V]](func() *serializer.MsgPackSerializer[V] {
		return serializer.NewMsgPackSerializer[V]()
	})
	ref.RegisterT[*serializer.BSONSerializer[V]](func() *serializer.BSONSerializer[V] {
		return serializer.NewBSONSerializer[V]()
	})

	// 获取K和V的类型名，用于构造默认TypeOptions
	var k K
	var v V

	// 设置默认的序列化器配置
	keySerializerOptions := options.KeySerializer
	if keySerializerOptions == nil {
		keySerializerOptions = &ref.TypeOptions{
			Namespace: "github.com/hatlonely/gox/kv/serializer",
			Type:      "MsgPackSerializer[" + reflect.TypeOf(k).String() + "]",
		}
	}

	valSerializerOptions := options.ValSerializer
	if valSerializerOptions == nil {
		valSerializerOptions = &ref.TypeOptions{
			Namespace: "github.com/hatlonely/gox/kv/serializer",
			Type:      "MsgPackSerializer[" + reflect.TypeOf(v).String() + "]",
		}
	}

	// 构造 key 序列化器
	keySerializerInterface, err := ref.NewWithOptions(keySerializerOptions)
	if err != nil {
		return nil, err
	}
	keySerializer, ok := keySerializerInterface.(serializer.Serializer[K, []byte])
	if !ok {
		return nil, errors.New("invalid key serializer type")
	}

	// 构造 value 序列化器
	valSerializerInterface, err := ref.NewWithOptions(valSerializerOptions)
	if err != nil {
		return nil, err
	}
	valueSerializer, ok := valSerializerInterface.(serializer.Serializer[V, []byte])
	if !ok {
		return nil, errors.New("invalid value serializer type")
	}

	return &FreeCacheStore[K, V]{
		cache:           freecache.NewCache(options.Size),
		keySerializer:   keySerializer,
		valueSerializer: valueSerializer,
	}, nil
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
