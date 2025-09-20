package store

import (
	"context"
	"errors"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/hatlonely/gox/kv/serializer"
	"github.com/hatlonely/gox/ref"
	"github.com/redis/go-redis/v9"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewRedisStoreWithOptions(t *testing.T) {
	PatchConvey("NewRedisStoreWithOptions", t, func() {
		Convey("使用默认配置创建", func() {
			options := &RedisStoreOptions{
				Endpoint: "localhost:6379",
			}

			// Mock redis.NewClient
			mockClient := &redis.Client{}
			Mock(redis.NewClient).Return(mockClient).Build()

			// Mock Ping method
			statusCmd := redis.NewStatusCmd(context.Background())
			statusCmd.SetVal("PONG")
			Mock((*redis.Client).Ping).Return(statusCmd).Build()

			store, err := NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
		})

		Convey("指定JSON序列化器", func() {
			options := &RedisStoreOptions{
				Endpoint: "localhost:6379",
				KeySerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "JSONSerializer[string]",
				},
				ValSerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "JSONSerializer[string]",
				},
			}

			// Mock redis.NewClient
			mockClient := &redis.Client{}
			Mock(redis.NewClient).Return(mockClient).Build()

			// Mock Ping method
			statusCmd := redis.NewStatusCmd(context.Background())
			statusCmd.SetVal("PONG")
			Mock((*redis.Client).Ping).Return(statusCmd).Build()

			store, err := NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
		})

		Convey("指定BSON序列化器", func() {
			options := &RedisStoreOptions{
				Endpoint: "localhost:6379",
				KeySerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "BSONSerializer[string]",
				},
				ValSerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "BSONSerializer[string]",
				},
			}

			// Mock redis.NewClient
			mockClient := &redis.Client{}
			Mock(redis.NewClient).Return(mockClient).Build()

			// Mock Ping method
			statusCmd := redis.NewStatusCmd(context.Background())
			statusCmd.SetVal("PONG")
			Mock((*redis.Client).Ping).Return(statusCmd).Build()

			store, err := NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
		})

		Convey("支持不同数据类型", func() {
			Convey("int类型值", func() {
				options := &RedisStoreOptions{
					Endpoint: "localhost:6379",
				}

				// Mock redis.NewClient
				mockClient := &redis.Client{}
				Mock(redis.NewClient).Return(mockClient).Build()

				// Mock Ping method
				statusCmd := redis.NewStatusCmd(context.Background())
				statusCmd.SetVal("PONG")
				Mock((*redis.Client).Ping).Return(statusCmd).Build()

				store, err := NewRedisStoreWithOptions[string, int](options)
				So(err, ShouldBeNil)
				So(store, ShouldNotBeNil)
			})

			Convey("int类型键", func() {
				options := &RedisStoreOptions{
					Endpoint: "localhost:6379",
				}

				// Mock redis.NewClient
				mockClient := &redis.Client{}
				Mock(redis.NewClient).Return(mockClient).Build()

				// Mock Ping method
				statusCmd := redis.NewStatusCmd(context.Background())
				statusCmd.SetVal("PONG")
				Mock((*redis.Client).Ping).Return(statusCmd).Build()

				store, err := NewRedisStoreWithOptions[int, string](options)
				So(err, ShouldBeNil)
				So(store, ShouldNotBeNil)
			})
		})

		Convey("配置默认TTL", func() {
			options := &RedisStoreOptions{
				Endpoint:   "localhost:6379",
				DefaultTTL: 300,
			}

			// Mock redis.NewClient
			mockClient := &redis.Client{}
			Mock(redis.NewClient).Return(mockClient).Build()

			// Mock Ping method
			statusCmd := redis.NewStatusCmd(context.Background())
			statusCmd.SetVal("PONG")
			Mock((*redis.Client).Ping).Return(statusCmd).Build()

			store, err := NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(store.defaultTTL, ShouldEqual, 300)
		})

		Convey("配置数据库编号", func() {
			options := &RedisStoreOptions{
				Endpoint: "localhost:6379",
				DB:       5,
			}

			// Mock redis.NewClient to verify DB option
			var capturedOpt *redis.Options
			Mock(redis.NewClient).To(func(opt *redis.Options) *redis.Client {
				capturedOpt = opt
				return &redis.Client{}
			}).Build()

			// Mock Ping method
			statusCmd := redis.NewStatusCmd(context.Background())
			statusCmd.SetVal("PONG")
			Mock((*redis.Client).Ping).Return(statusCmd).Build()

			store, err := NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(capturedOpt.DB, ShouldEqual, 5)
		})

		Convey("Ping失败", func() {
			options := &RedisStoreOptions{
				Endpoint: "localhost:6379",
			}

			// Mock redis.NewClient
			mockClient := &redis.Client{}
			Mock(redis.NewClient).Return(mockClient).Build()

			// Mock Ping method to return error
			statusCmd := redis.NewStatusCmd(context.Background())
			statusCmd.SetErr(errors.New("connection refused"))
			Mock((*redis.Client).Ping).Return(statusCmd).Build()

			_, err := NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "redis.client.Ping failed")
		})
	})
}

func TestRedisStoreSet(t *testing.T) {
	PatchConvey("RedisStore.Set", t, func() {
		store := &RedisStore[string, string]{
			client:        &redis.Client{},
			defaultTTL:    300,
			keySerializer: serializer.NewMsgPackSerializer[string](),
			valSerializer: serializer.NewMsgPackSerializer[string](),
		}

		ctx := context.Background()

		Convey("基本设置操作", func() {
			key := "test_key"
			value := "test_value"

			// Mock Set command to return success
			statusCmd := redis.NewStatusCmd(ctx)
			statusCmd.SetVal("OK")
			Mock((*redis.Client).Set).Return(statusCmd).Build()

			err := store.Set(ctx, key, value)
			So(err, ShouldBeNil)
		})

		Convey("设置操作失败", func() {
			key := "test_key"
			value := "test_value"

			// Mock Set command to return error
			statusCmd := redis.NewStatusCmd(ctx)
			statusCmd.SetErr(errors.New("redis error"))
			Mock((*redis.Client).Set).Return(statusCmd).Build()

			err := store.Set(ctx, key, value)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "redis error")
		})

		Convey("覆盖已存在的键", func() {
			key := "test_key_overwrite"
			value1 := "test_value1"
			value2 := "test_value2"

			// Mock Set command to return success for both operations
			statusCmd := redis.NewStatusCmd(ctx)
			statusCmd.SetVal("OK")
			Mock((*redis.Client).Set).Return(statusCmd).Build()

			// 设置第一个值
			err := store.Set(ctx, key, value1)
			So(err, ShouldBeNil)

			// 覆盖为第二个值
			err = store.Set(ctx, key, value2)
			So(err, ShouldBeNil)
		})

		Convey("带过期时间设置", func() {
			key := "test_key_ttl"
			value := "test_value"

			// Mock Set command to return success
			statusCmd := redis.NewStatusCmd(ctx)
			statusCmd.SetVal("OK")
			Mock((*redis.Client).Set).Return(statusCmd).Build()

			err := store.Set(ctx, key, value, WithExpiration(600))
			So(err, ShouldBeNil)
		})
	})
}

func TestRedisStoreGet(t *testing.T) {
	PatchConvey("RedisStore.Get", t, func() {
		store := &RedisStore[string, string]{
			client:        &redis.Client{},
			defaultTTL:    300,
			keySerializer: serializer.NewMsgPackSerializer[string](),
			valSerializer: serializer.NewMsgPackSerializer[string](),
		}

		ctx := context.Background()

		Convey("获取存在的键", func() {
			key := "test_key"
			value := "test_value"

			// Prepare serialized value
			serializedValue, _ := store.valSerializer.Serialize(value)

			stringCmd := redis.NewStringCmd(ctx)
			stringCmd.SetVal(string(serializedValue))
			Mock((*redis.Client).Get).Return(stringCmd).Build()

			gotValue, err := store.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value)
		})

		Convey("获取不存在的键", func() {
			// Mock Get command to return redis.Nil error
			stringCmd := redis.NewStringCmd(ctx)
			stringCmd.SetErr(redis.Nil)
			Mock((*redis.Client).Get).Return(stringCmd).Build()

			_, err := store.Get(ctx, "non_exist_key")
			So(err, ShouldEqual, ErrKeyNotFound)
		})

		Convey("Redis连接错误", func() {
			// Mock Get command to return connection error
			stringCmd := redis.NewStringCmd(ctx)
			stringCmd.SetErr(errors.New("connection error"))
			Mock((*redis.Client).Get).Return(stringCmd).Build()

			_, err := store.Get(ctx, "test_key")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "connection error")
		})

		Convey("不同数据类型", func() {
			Convey("int类型值", func() {
				intStore := &RedisStore[string, int]{
					client:        &redis.Client{},
					defaultTTL:    300,
					keySerializer: serializer.NewMsgPackSerializer[string](),
					valSerializer: serializer.NewMsgPackSerializer[int](),
				}

				key := "int_key"
				value := 42

				// Prepare serialized value
				serializedValue, _ := intStore.valSerializer.Serialize(value)

				stringCmd := redis.NewStringCmd(ctx)
				stringCmd.SetVal(string(serializedValue))
				Mock((*redis.Client).Get).Return(stringCmd).Build()

				gotValue, err := intStore.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, value)
			})
		})
	})
}

func TestRedisStoreDel(t *testing.T) {
	PatchConvey("RedisStore.Del", t, func() {
		store := &RedisStore[string, string]{
			client:        &redis.Client{},
			defaultTTL:    300,
			keySerializer: serializer.NewMsgPackSerializer[string](),
			valSerializer: serializer.NewMsgPackSerializer[string](),
		}

		ctx := context.Background()

		Convey("删除存在的键", func() {
			key := "test_key_del"

			// Mock Del command to return 1 (key deleted)
			intCmd := redis.NewIntCmd(ctx)
			intCmd.SetVal(1)
			Mock((*redis.Client).Del).Return(intCmd).Build()

			err := store.Del(ctx, key)
			So(err, ShouldBeNil)
		})

		Convey("删除不存在的键", func() {
			// Mock Del command to return 0 (no key deleted)
			intCmd := redis.NewIntCmd(ctx)
			intCmd.SetVal(0)
			Mock((*redis.Client).Del).Return(intCmd).Build()

			err := store.Del(ctx, "non_exist_key")
			So(err, ShouldBeNil)
		})

		Convey("删除操作失败", func() {
			// Mock Del command to return error
			intCmd := redis.NewIntCmd(ctx)
			intCmd.SetErr(errors.New("redis error"))
			Mock((*redis.Client).Del).Return(intCmd).Build()

			err := store.Del(ctx, "test_key")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "redis error")
		})
	})
}

func TestRedisStoreBatchSet(t *testing.T) {
	PatchConvey("RedisStore.BatchSet", t, func() {
		store := &RedisStore[string, string]{
			client:        &redis.Client{},
			defaultTTL:    300,
			keySerializer: serializer.NewMsgPackSerializer[string](),
			valSerializer: serializer.NewMsgPackSerializer[string](),
		}

		ctx := context.Background()

		Convey("键值数量不匹配", func() {
			keys := []string{"key1", "key2"}
			vals := []string{"val1"}

			_, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "length mismatch")
		})

		Convey("空数组", func() {
			keys := []string{}
			vals := []string{}

			errs, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 0)
		})
	})
}

func TestRedisStoreBatchGet(t *testing.T) {
	PatchConvey("RedisStore.BatchGet", t, func() {
		store := &RedisStore[string, string]{
			client:        &redis.Client{},
			defaultTTL:    300,
			keySerializer: serializer.NewMsgPackSerializer[string](),
			valSerializer: serializer.NewMsgPackSerializer[string](),
		}

		ctx := context.Background()

		Convey("批量获取存在的键", func() {
			keys := []string{"key1", "key2", "key3"}
			expectedVals := []string{"val1", "val2", "val3"}

			// Prepare serialized values
			val1, _ := store.valSerializer.Serialize("val1")
			val2, _ := store.valSerializer.Serialize("val2")
			val3, _ := store.valSerializer.Serialize("val3")

			// Mock MGet command
			sliceCmd := redis.NewSliceCmd(ctx)
			sliceCmd.SetVal([]any{string(val1), string(val2), string(val3)})
			Mock((*redis.Client).MGet).Return(sliceCmd).Build()

			gotVals, errs, err := store.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotVals), ShouldEqual, 3)
			So(len(errs), ShouldEqual, 3)

			for i := range keys {
				So(errs[i], ShouldBeNil)
				So(gotVals[i], ShouldEqual, expectedVals[i])
			}
		})

		Convey("批量获取不存在的键", func() {
			keys := []string{"non_exist1", "non_exist2"}

			// Mock MGet command with nil values
			sliceCmd := redis.NewSliceCmd(ctx)
			sliceCmd.SetVal([]any{nil, nil})
			Mock((*redis.Client).MGet).Return(sliceCmd).Build()

			gotVals, errs, err := store.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotVals), ShouldEqual, 2)
			So(len(errs), ShouldEqual, 2)

			for _, e := range errs {
				So(e, ShouldEqual, ErrKeyNotFound)
			}
		})

		Convey("混合存在和不存在的键", func() {
			keys := []string{"exist_key", "non_exist_key"}

			// Prepare one serialized value
			val1, _ := store.valSerializer.Serialize("exist_value")

			// Mock MGet command with mixed results
			sliceCmd := redis.NewSliceCmd(ctx)
			sliceCmd.SetVal([]any{string(val1), nil})
			Mock((*redis.Client).MGet).Return(sliceCmd).Build()

			gotVals, errs, err := store.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotVals), ShouldEqual, 2)
			So(len(errs), ShouldEqual, 2)

			So(errs[0], ShouldBeNil)
			So(gotVals[0], ShouldEqual, "exist_value")
			So(errs[1], ShouldEqual, ErrKeyNotFound)
		})

		Convey("空数组", func() {
			keys := []string{}

			gotVals, errs, err := store.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotVals), ShouldEqual, 0)
			So(len(errs), ShouldEqual, 0)
		})

	})
}

func TestRedisStoreBatchDel(t *testing.T) {
	PatchConvey("RedisStore.BatchDel", t, func() {
		store := &RedisStore[string, string]{
			client:        &redis.Client{},
			defaultTTL:    300,
			keySerializer: serializer.NewMsgPackSerializer[string](),
			valSerializer: serializer.NewMsgPackSerializer[string](),
		}

		ctx := context.Background()

		Convey("空数组", func() {
			keys := []string{}

			errs, err := store.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 0)
		})
	})
}

