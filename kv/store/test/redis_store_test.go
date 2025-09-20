package test

import (
	"context"
	"testing"
	"time"

	"github.com/hatlonely/gox/kv/store"
	"github.com/hatlonely/gox/ref"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewRedisStoreWithOptions(t *testing.T) {
	Convey("NewRedisStoreWithOptions", t, func() {
		Convey("使用默认配置创建", func() {
			options := &store.RedisStoreOptions{
				Endpoint: "localhost:6379",
				DB:       15, // 使用数据库15进行测试
			}

			redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(redisStore, ShouldNotBeNil)
			defer redisStore.Close()
		})

		Convey("指定JSON序列化器", func() {
			options := &store.RedisStoreOptions{
				Endpoint: "localhost:6379",
				DB:       15,
				KeySerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "JSONSerializer[string]",
				},
				ValSerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "JSONSerializer[string]",
				},
			}

			redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(redisStore, ShouldNotBeNil)
			defer redisStore.Close()
		})

		Convey("指定BSON序列化器", func() {
			options := &store.RedisStoreOptions{
				Endpoint: "localhost:6379",
				DB:       15,
				KeySerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "BSONSerializer[string]",
				},
				ValSerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "BSONSerializer[string]",
				},
			}

			redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(redisStore, ShouldNotBeNil)
			defer redisStore.Close()
		})

		Convey("支持不同数据类型", func() {
			Convey("int类型值", func() {
				options := &store.RedisStoreOptions{
					Endpoint: "localhost:6379",
					DB:       15,
				}
				redisStore, err := store.NewRedisStoreWithOptions[string, int](options)
				So(err, ShouldBeNil)
				So(redisStore, ShouldNotBeNil)
				defer redisStore.Close()
			})

			Convey("int类型键", func() {
				options := &store.RedisStoreOptions{
					Endpoint: "localhost:6379",
					DB:       15,
				}
				redisStore, err := store.NewRedisStoreWithOptions[int, string](options)
				So(err, ShouldBeNil)
				So(redisStore, ShouldNotBeNil)
				defer redisStore.Close()
			})
		})

		Convey("设置默认TTL", func() {
			options := &store.RedisStoreOptions{
				Endpoint:   "localhost:6379",
				DB:         15,
				DefaultTTL: 10, // 10秒默认TTL
			}

			redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(redisStore, ShouldNotBeNil)
			defer redisStore.Close()
		})

		Convey("集群模式配置", func() {
			options := &store.RedisStoreOptions{
				Endpoints: []string{"localhost:7000", "localhost:7001", "localhost:7002"},
			}

			// 注意：这个测试可能会失败如果没有Redis集群环境
			// 这里仅测试配置是否正确解析，实际连接可能失败
			_, err := store.NewRedisStoreWithOptions[string, string](options)
			// 我们不检查err，因为集群可能不存在
			_ = err
		})
	})
}

func TestRedisStoreSet(t *testing.T) {
	Convey("RedisStore.Set", t, func() {
		options := &store.RedisStoreOptions{
			Endpoint: "localhost:6379",
			DB:       15,
		}
		redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer redisStore.Close()

		ctx := context.Background()

		Convey("基本设置操作", func() {
			key := "test:set:basic"
			value := "test_value"

			// 清理可能存在的键
			_ = redisStore.Del(ctx, key)

			err := redisStore.Set(ctx, key, value)
			So(err, ShouldBeNil)

			// 验证设置成功
			gotValue, err := redisStore.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value)

			// 清理
			_ = redisStore.Del(ctx, key)
		})

		Convey("条件设置 - IfNotExist", func() {
			key := "test:set:ifnotexist"
			value := "test_value"

			// 清理
			_ = redisStore.Del(ctx, key)

			Convey("键不存在时设置成功", func() {
				err := redisStore.Set(ctx, key, value, store.WithIfNotExist())
				So(err, ShouldBeNil)

				// 验证设置成功
				gotValue, err := redisStore.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, value)

				// 清理
				_ = redisStore.Del(ctx, key)
			})

			Convey("键存在时返回条件失败错误", func() {
				// 先设置一个值
				err := redisStore.Set(ctx, key, "original_value")
				So(err, ShouldBeNil)

				// 尝试条件设置应该失败
				err = redisStore.Set(ctx, key, "new_value", store.WithIfNotExist())
				So(err, ShouldEqual, store.ErrConditionFailed)

				// 验证值没有被覆盖
				gotValue, err := redisStore.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, "original_value")

				// 清理
				_ = redisStore.Del(ctx, key)
			})
		})

		Convey("覆盖已存在的键", func() {
			key := "test:set:overwrite"
			value1 := "test_value1"
			value2 := "test_value2"

			// 清理
			_ = redisStore.Del(ctx, key)

			// 设置第一个值
			err := redisStore.Set(ctx, key, value1)
			So(err, ShouldBeNil)

			// 覆盖为第二个值
			err = redisStore.Set(ctx, key, value2)
			So(err, ShouldBeNil)

			// 验证值已被覆盖
			gotValue, err := redisStore.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value2)

			// 清理
			_ = redisStore.Del(ctx, key)
		})

		Convey("设置过期时间", func() {
			key := "test:set:expire"
			value := "expire_value"

			// 清理
			_ = redisStore.Del(ctx, key)

			// 设置1秒过期
			err := redisStore.Set(ctx, key, value, store.WithExpiration(1*time.Second))
			So(err, ShouldBeNil)

			// 立即获取应该成功
			gotValue, err := redisStore.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value)

			// 等待过期
			time.Sleep(1200 * time.Millisecond)

			// 过期后获取应该失败
			_, err = redisStore.Get(ctx, key)
			So(err, ShouldEqual, store.ErrKeyNotFound)
		})
	})
}

func TestRedisStoreGet(t *testing.T) {
	Convey("RedisStore.Get", t, func() {
		options := &store.RedisStoreOptions{
			Endpoint: "localhost:6379",
			DB:       15,
		}
		redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer redisStore.Close()

		ctx := context.Background()

		Convey("获取存在的键", func() {
			key := "test:get:exist"
			value := "test_value"

			// 先设置值
			err := redisStore.Set(ctx, key, value)
			So(err, ShouldBeNil)

			gotValue, err := redisStore.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value)

			// 清理
			_ = redisStore.Del(ctx, key)
		})

		Convey("获取不存在的键", func() {
			key := "test:get:nonexist"
			
			// 确保键不存在
			_ = redisStore.Del(ctx, key)

			_, err := redisStore.Get(ctx, key)
			So(err, ShouldEqual, store.ErrKeyNotFound)
		})

		Convey("不同数据类型", func() {
			Convey("int类型值", func() {
				intStore, err := store.NewRedisStoreWithOptions[string, int](&store.RedisStoreOptions{
					Endpoint: "localhost:6379",
					DB:       15,
				})
				So(err, ShouldBeNil)
				defer intStore.Close()

				key := "test:get:int"
				value := 42

				err = intStore.Set(ctx, key, value)
				So(err, ShouldBeNil)

				gotValue, err := intStore.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, value)

				// 清理
				_ = intStore.Del(ctx, key)
			})
		})

		Convey("获取已删除的键", func() {
			key := "test:get:deleted"
			value := "test_value"

			// 设置值
			err := redisStore.Set(ctx, key, value)
			So(err, ShouldBeNil)

			// 删除键
			err = redisStore.Del(ctx, key)
			So(err, ShouldBeNil)

			// 尝试获取应该失败
			_, err = redisStore.Get(ctx, key)
			So(err, ShouldEqual, store.ErrKeyNotFound)
		})
	})
}

func TestRedisStoreDel(t *testing.T) {
	Convey("RedisStore.Del", t, func() {
		options := &store.RedisStoreOptions{
			Endpoint: "localhost:6379",
			DB:       15,
		}
		redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer redisStore.Close()

		ctx := context.Background()

		Convey("删除存在的键", func() {
			key := "test:del:exist"
			value := "test_value"

			err := redisStore.Set(ctx, key, value)
			So(err, ShouldBeNil)

			err = redisStore.Del(ctx, key)
			So(err, ShouldBeNil)

			_, err = redisStore.Get(ctx, key)
			So(err, ShouldEqual, store.ErrKeyNotFound)
		})

		Convey("删除不存在的键", func() {
			key := "test:del:nonexist"
			
			// 确保键不存在
			_ = redisStore.Del(ctx, key)

			err := redisStore.Del(ctx, key)
			So(err, ShouldBeNil)
		})

		Convey("删除后重新设置", func() {
			key := "test:del:reset"
			value1 := "test_value1"
			value2 := "test_value2"

			// 设置值
			err := redisStore.Set(ctx, key, value1)
			So(err, ShouldBeNil)

			// 删除
			err = redisStore.Del(ctx, key)
			So(err, ShouldBeNil)

			// 重新设置
			err = redisStore.Set(ctx, key, value2)
			So(err, ShouldBeNil)

			// 验证新值
			gotValue, err := redisStore.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value2)

			// 清理
			_ = redisStore.Del(ctx, key)
		})
	})
}

func TestRedisStoreBatchSet(t *testing.T) {
	Convey("RedisStore.BatchSet", t, func() {
		options := &store.RedisStoreOptions{
			Endpoint: "localhost:6379",
			DB:       15,
		}
		redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer redisStore.Close()

		ctx := context.Background()

		Convey("正常批量设置", func() {
			keys := []string{"test:batch_set:key1", "test:batch_set:key2", "test:batch_set:key3"}
			vals := []string{"val1", "val2", "val3"}

			// 清理
			for _, key := range keys {
				_ = redisStore.Del(ctx, key)
			}

			errs, err := redisStore.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 3)
			for _, e := range errs {
				So(e, ShouldBeNil)
			}

			// 验证设置成功
			for i, key := range keys {
				gotValue, err := redisStore.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, vals[i])
			}

			// 清理
			for _, key := range keys {
				_ = redisStore.Del(ctx, key)
			}
		})

		Convey("键值数量不匹配", func() {
			keys := []string{"key1", "key2"}
			vals := []string{"val1"}

			_, err := redisStore.BatchSet(ctx, keys, vals)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mismatch")
		})

		Convey("空数组", func() {
			keys := []string{}
			vals := []string{}

			errs, err := redisStore.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 0)
		})

		Convey("带条件的批量设置", func() {
			keys := []string{"test:batch_set_cond:key1", "test:batch_set_cond:key2"}
			vals := []string{"val1", "val2"}

			// 清理
			for _, key := range keys {
				_ = redisStore.Del(ctx, key)
			}

			// 先设置一个键
			err := redisStore.Set(ctx, keys[0], "existing_value")
			So(err, ShouldBeNil)

			errs, err := redisStore.BatchSet(ctx, keys, vals, store.WithIfNotExist())
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 2)
			So(errs[0], ShouldEqual, store.ErrConditionFailed) // key1 已存在
			So(errs[1], ShouldBeNil)                           // key2 不存在，设置成功

			// 验证 key1 没有被覆盖
			gotValue, err := redisStore.Get(ctx, keys[0])
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, "existing_value")

			// 验证 key2 设置成功
			gotValue, err = redisStore.Get(ctx, keys[1])
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, "val2")

			// 清理
			for _, key := range keys {
				_ = redisStore.Del(ctx, key)
			}
		})
	})
}

func TestRedisStoreBatchGet(t *testing.T) {
	Convey("RedisStore.BatchGet", t, func() {
		options := &store.RedisStoreOptions{
			Endpoint: "localhost:6379",
			DB:       15,
		}
		redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer redisStore.Close()

		ctx := context.Background()

		Convey("批量获取存在的键", func() {
			keys := []string{"test:batch_get:key1", "test:batch_get:key2", "test:batch_get:key3"}
			vals := []string{"val1", "val2", "val3"}

			// 先批量设置
			_, err := redisStore.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)

			// 批量获取
			gotVals, errs, err := redisStore.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotVals), ShouldEqual, 3)
			So(len(errs), ShouldEqual, 3)

			for i := range keys {
				So(errs[i], ShouldBeNil)
				So(gotVals[i], ShouldEqual, vals[i])
			}

			// 清理
			for _, key := range keys {
				_ = redisStore.Del(ctx, key)
			}
		})

		Convey("批量获取不存在的键", func() {
			keys := []string{"test:batch_get:non1", "test:batch_get:non2"}

			// 确保键不存在
			for _, key := range keys {
				_ = redisStore.Del(ctx, key)
			}

			gotVals, errs, err := redisStore.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotVals), ShouldEqual, 2)
			So(len(errs), ShouldEqual, 2)

			for _, e := range errs {
				So(e, ShouldEqual, store.ErrKeyNotFound)
			}
		})

		Convey("混合存在和不存在的键", func() {
			existKey := "test:batch_get:exist"
			nonExistKey := "test:batch_get:nonexist"

			// 设置一个存在的键
			err := redisStore.Set(ctx, existKey, "exist_value")
			So(err, ShouldBeNil)

			// 确保另一个键不存在
			_ = redisStore.Del(ctx, nonExistKey)

			keys := []string{existKey, nonExistKey}
			gotVals, errs, err := redisStore.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotVals), ShouldEqual, 2)
			So(len(errs), ShouldEqual, 2)

			So(errs[0], ShouldBeNil)
			So(gotVals[0], ShouldEqual, "exist_value")
			So(errs[1], ShouldEqual, store.ErrKeyNotFound)

			// 清理
			_ = redisStore.Del(ctx, existKey)
		})

		Convey("空数组", func() {
			keys := []string{}

			gotVals, errs, err := redisStore.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotVals), ShouldEqual, 0)
			So(len(errs), ShouldEqual, 0)
		})
	})
}

func TestRedisStoreBatchDel(t *testing.T) {
	Convey("RedisStore.BatchDel", t, func() {
		options := &store.RedisStoreOptions{
			Endpoint: "localhost:6379",
			DB:       15,
		}
		redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer redisStore.Close()

		ctx := context.Background()

		Convey("批量删除存在的键", func() {
			keys := []string{"test:batch_del:key1", "test:batch_del:key2", "test:batch_del:key3"}
			vals := []string{"val1", "val2", "val3"}

			// 先批量设置
			_, err := redisStore.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)

			// 批量删除
			errs, err := redisStore.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 3)
			for _, e := range errs {
				So(e, ShouldBeNil)
			}

			// 验证删除成功
			for _, key := range keys {
				_, err := redisStore.Get(ctx, key)
				So(err, ShouldEqual, store.ErrKeyNotFound)
			}
		})

		Convey("批量删除不存在的键", func() {
			keys := []string{"test:batch_del:non1", "test:batch_del:non2"}

			// 确保键不存在
			for _, key := range keys {
				_ = redisStore.Del(ctx, key)
			}

			errs, err := redisStore.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 2)
			for _, e := range errs {
				So(e, ShouldBeNil)
			}
		})

		Convey("空数组", func() {
			keys := []string{}

			errs, err := redisStore.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 0)
		})

		Convey("混合存在和不存在的键", func() {
			existKey := "test:batch_del:exist"
			nonExistKey := "test:batch_del:nonexist"

			// 设置一个存在的键
			err := redisStore.Set(ctx, existKey, "exist_value")
			So(err, ShouldBeNil)

			// 确保另一个键不存在
			_ = redisStore.Del(ctx, nonExistKey)

			keys := []string{existKey, nonExistKey}
			errs, err := redisStore.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 2)
			for _, e := range errs {
				So(e, ShouldBeNil) // 删除不存在的键也返回成功
			}

			// 验证存在的键被删除
			_, err = redisStore.Get(ctx, existKey)
			So(err, ShouldEqual, store.ErrKeyNotFound)
		})
	})
}

func TestRedisStoreClose(t *testing.T) {
	Convey("RedisStore.Close", t, func() {
		Convey("正常关闭", func() {
			options := &store.RedisStoreOptions{
				Endpoint: "localhost:6379",
				DB:       15,
			}
			redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)

			err = redisStore.Close()
			So(err, ShouldBeNil)
		})

		Convey("重复关闭", func() {
			options := &store.RedisStoreOptions{
				Endpoint: "localhost:6379",
				DB:       15,
			}
			redisStore, err := store.NewRedisStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)

			err = redisStore.Close()
			So(err, ShouldBeNil)

			// Redis 客户端重复关闭会返回错误，这是正常行为
			err = redisStore.Close()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "client is closed")
		})
	})
}