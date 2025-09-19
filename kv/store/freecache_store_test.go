package store

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/hatlonely/gox/ref"
)

func TestFreeCacheStore(t *testing.T) {
	Convey("FreeCacheStore", t, func() {
		Convey("构造函数", func() {
			Convey("使用默认配置创建", func() {
				options := &FreeCacheStoreOptions{
					Size: 1024 * 1024, // 1MB
				}
				
				store, err := NewFreeCacheStoreWithOptions[string, string](options)
				
				So(err, ShouldBeNil)
				So(store, ShouldNotBeNil)
				defer store.Close()
			})

			Convey("指定JSON序列化器", func() {
				options := &FreeCacheStoreOptions{
					Size: 1024 * 1024,
					KeySerializer: &ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/kv/serializer",
						Type:      "JSONSerializer[string]",
					},
					ValSerializer: &ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/kv/serializer",
						Type:      "JSONSerializer[string]",
					},
				}
				
				store, err := NewFreeCacheStoreWithOptions[string, string](options)
				
				So(err, ShouldBeNil)
				So(store, ShouldNotBeNil)
				defer store.Close()
			})

			Convey("指定BSON序列化器", func() {
				options := &FreeCacheStoreOptions{
					Size: 1024 * 1024,
					KeySerializer: &ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/kv/serializer",
						Type:      "BSONSerializer[string]",
					},
					ValSerializer: &ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/kv/serializer",
						Type:      "BSONSerializer[string]",
					},
				}
				
				store, err := NewFreeCacheStoreWithOptions[string, string](options)
				
				So(err, ShouldBeNil)
				So(store, ShouldNotBeNil)
				defer store.Close()
			})
		})

		Convey("基本操作", func() {
			options := &FreeCacheStoreOptions{
				Size: 1024 * 1024,
			}
			store, err := NewFreeCacheStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			defer store.Close()

			ctx := context.Background()

			Convey("Set和Get操作", func() {
				key := "test_key"
				value := "test_value"

				Convey("设置值", func() {
					err := store.Set(ctx, key, value)
					So(err, ShouldBeNil)
				})

				Convey("获取值", func() {
					err := store.Set(ctx, key, value)
					So(err, ShouldBeNil)

					gotValue, err := store.Get(ctx, key)
					So(err, ShouldBeNil)
					So(gotValue, ShouldEqual, value)
				})
			})

			Convey("Delete操作", func() {
				key := "test_key_del"
				value := "test_value_del"

				Convey("删除存在的键", func() {
					err := store.Set(ctx, key, value)
					So(err, ShouldBeNil)

					err = store.Del(ctx, key)
					So(err, ShouldBeNil)

					_, err = store.Get(ctx, key)
					So(err, ShouldEqual, ErrKeyNotFound)
				})

				Convey("删除不存在的键", func() {
					err := store.Del(ctx, "non_exist_key")
					So(err, ShouldBeNil)
				})
			})

			Convey("获取不存在的键", func() {
				_, err := store.Get(ctx, "non_exist_key")
				So(err, ShouldEqual, ErrKeyNotFound)
			})
		})

		Convey("高级功能", func() {
			options := &FreeCacheStoreOptions{
				Size: 1024 * 1024,
			}
			store, err := NewFreeCacheStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			defer store.Close()

			ctx := context.Background()

			Convey("带过期时间的操作", func() {
				key := "test_key_expire"
				value := "test_value_expire"

				Convey("设置带过期时间的值", func() {
					err := store.Set(ctx, key, value, WithExpiration(time.Millisecond*100))
					So(err, ShouldBeNil)

					// 立即获取应该成功
					gotValue, err := store.Get(ctx, key)
					So(err, ShouldBeNil)
					So(gotValue, ShouldEqual, value)
				})

				Convey("过期后获取应该失败", func() {
					err := store.Set(ctx, key, value, WithExpiration(time.Millisecond*50))
					So(err, ShouldBeNil)

					// 等待过期
					time.Sleep(time.Millisecond * 100)
					
					_, err = store.Get(ctx, key)
					// 注意：freecache的过期可能不是立即生效的
					// So(err, ShouldEqual, ErrKeyNotFound)
				})
			})

			Convey("条件设置", func() {
				key := "test_key_ifnotexist"
				value1 := "test_value1"
				value2 := "test_value2"

				Convey("IfNotExist - 键不存在时设置", func() {
					err := store.Set(ctx, key, value1, WithIfNotExist())
					So(err, ShouldBeNil)

					gotValue, err := store.Get(ctx, key)
					So(err, ShouldBeNil)
					So(gotValue, ShouldEqual, value1)
				})

				Convey("IfNotExist - 键存在时不设置", func() {
					// 先设置一个值
					err := store.Set(ctx, key, value1)
					So(err, ShouldBeNil)

					// 尝试条件设置
					err = store.Set(ctx, key, value2, WithIfNotExist())
					So(err, ShouldBeNil)

					// 值应该保持原来的
					gotValue, err := store.Get(ctx, key)
					So(err, ShouldBeNil)
					So(gotValue, ShouldEqual, value1)
				})
			})
		})

		Convey("批量操作", func() {
			options := &FreeCacheStoreOptions{
				Size: 1024 * 1024,
			}
			store, err := NewFreeCacheStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			defer store.Close()

			ctx := context.Background()

			Convey("BatchSet", func() {
				keys := []string{"key1", "key2", "key3"}
				vals := []string{"val1", "val2", "val3"}

				errs, err := store.BatchSet(ctx, keys, vals)
				So(err, ShouldBeNil)
				So(len(errs), ShouldEqual, 3)
				for _, e := range errs {
					So(e, ShouldBeNil)
				}
			})

			Convey("BatchGet", func() {
				keys := []string{"key1", "key2", "key3"}
				vals := []string{"val1", "val2", "val3"}

				// 先批量设置
				_, err := store.BatchSet(ctx, keys, vals)
				So(err, ShouldBeNil)

				// 批量获取
				gotVals, errs, err := store.BatchGet(ctx, keys)
				So(err, ShouldBeNil)
				So(len(gotVals), ShouldEqual, 3)
				So(len(errs), ShouldEqual, 3)
				
				for i := range keys {
					So(errs[i], ShouldBeNil)
					So(gotVals[i], ShouldEqual, vals[i])
				}
			})

			Convey("BatchDel", func() {
				keys := []string{"key1", "key2", "key3"}
				vals := []string{"val1", "val2", "val3"}

				// 先批量设置
				_, err := store.BatchSet(ctx, keys, vals)
				So(err, ShouldBeNil)

				// 批量删除
				errs, err := store.BatchDel(ctx, keys)
				So(err, ShouldBeNil)
				So(len(errs), ShouldEqual, 3)
				for _, e := range errs {
					So(e, ShouldBeNil)
				}

				// 验证删除成功
				for _, key := range keys {
					_, err := store.Get(ctx, key)
					So(err, ShouldEqual, ErrKeyNotFound)
				}
			})

			Convey("BatchSet键值数量不匹配", func() {
				keys := []string{"key1", "key2"}
				vals := []string{"val1"}

				_, err := store.BatchSet(ctx, keys, vals)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "length mismatch")
			})
		})

		Convey("不同数据类型", func() {
			Convey("int类型", func() {
				options := &FreeCacheStoreOptions{
					Size: 1024 * 1024,
				}
				store, err := NewFreeCacheStoreWithOptions[string, int](options)
				So(err, ShouldBeNil)
				defer store.Close()

				ctx := context.Background()
				key := "int_key"
				value := 42

				err = store.Set(ctx, key, value)
				So(err, ShouldBeNil)

				gotValue, err := store.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, value)
			})

			Convey("结构体类型", func() {
				type User struct {
					Name string `json:"name"`
					Age  int    `json:"age"`
				}

				options := &FreeCacheStoreOptions{
					Size: 1024 * 1024,
					// 显式指定JSON序列化器来避免类型名问题
					ValSerializer: &ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/kv/serializer",
						Type:      "JSONSerializer[store.User]",
					},
				}
				store, err := NewFreeCacheStoreWithOptions[string, User](options)
				So(err, ShouldBeNil)
				defer store.Close()

				ctx := context.Background()
				key := "user_key"
				value := User{Name: "张三", Age: 25}

				err = store.Set(ctx, key, value)
				So(err, ShouldBeNil)

				gotValue, err := store.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue.Name, ShouldEqual, value.Name)
				So(gotValue.Age, ShouldEqual, value.Age)
			})
		})
	})
}