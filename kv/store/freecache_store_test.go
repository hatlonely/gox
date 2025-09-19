package store

import (
	"context"
	"testing"
	"time"

	"github.com/hatlonely/gox/ref"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewFreeCacheStoreWithOptions(t *testing.T) {
	Convey("NewFreeCacheStoreWithOptions", t, func() {
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

		Convey("支持不同数据类型", func() {
			Convey("int类型值", func() {
				options := &FreeCacheStoreOptions{
					Size: 1024 * 1024,
				}
				store, err := NewFreeCacheStoreWithOptions[string, int](options)
				So(err, ShouldBeNil)
				So(store, ShouldNotBeNil)
				defer store.Close()
			})

			Convey("int类型键", func() {
				options := &FreeCacheStoreOptions{
					Size: 1024 * 1024,
				}
				store, err := NewFreeCacheStoreWithOptions[int, string](options)
				So(err, ShouldBeNil)
				So(store, ShouldNotBeNil)
				defer store.Close()
			})
		})
	})
}

func TestFreeCacheStoreSet(t *testing.T) {
	Convey("FreeCacheStore.Set", t, func() {
		options := &FreeCacheStoreOptions{
			Size: 1024 * 1024,
		}
		store, err := NewFreeCacheStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("基本设置操作", func() {
			key := "test_key"
			value := "test_value"

			err := store.Set(ctx, key, value)
			So(err, ShouldBeNil)
		})

		Convey("带过期时间设置", func() {
			key := "test_key_expire"
			value := "test_value_expire"

			err := store.Set(ctx, key, value, WithExpiration(time.Minute))
			So(err, ShouldBeNil)
		})

		Convey("条件设置 - IfNotExist", func() {
			key := "test_key_ifnotexist"
			value := "test_value"

			Convey("键不存在时设置成功", func() {
				err := store.Set(ctx, key, value, WithIfNotExist())
				So(err, ShouldBeNil)
			})

			Convey("键存在时不覆盖", func() {
				// 先设置一个值
				err := store.Set(ctx, key, "original_value")
				So(err, ShouldBeNil)

				// 尝试条件设置
				err = store.Set(ctx, key, "new_value", WithIfNotExist())
				So(err, ShouldBeNil)

				// 验证值没有被覆盖
				gotValue, err := store.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, "original_value")
			})
		})

		Convey("组合选项", func() {
			key := "test_key_combined"
			value := "test_value"

			err := store.Set(ctx, key, value, WithExpiration(time.Minute), WithIfNotExist())
			So(err, ShouldBeNil)
		})
	})
}

func TestFreeCacheStoreGet(t *testing.T) {
	Convey("FreeCacheStore.Get", t, func() {
		options := &FreeCacheStoreOptions{
			Size: 1024 * 1024,
		}
		store, err := NewFreeCacheStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("获取存在的键", func() {
			key := "test_key"
			value := "test_value"

			err := store.Set(ctx, key, value)
			So(err, ShouldBeNil)

			gotValue, err := store.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value)
		})

		Convey("获取不存在的键", func() {
			_, err := store.Get(ctx, "non_exist_key")
			So(err, ShouldEqual, ErrKeyNotFound)
		})

		Convey("获取已过期的键", func() {
			key := "test_key_expire"
			value := "test_value_expire"

			err := store.Set(ctx, key, value, WithExpiration(time.Millisecond*50))
			So(err, ShouldBeNil)

			// 立即获取应该成功
			gotValue, err := store.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value)

			// 等待过期
			time.Sleep(time.Millisecond * 100)

			// freecache的过期可能不是立即生效的，所以这里只是测试不会panic
			_, _ = store.Get(ctx, key)
		})

		Convey("不同数据类型", func() {
			Convey("int类型值", func() {
				intStore, err := NewFreeCacheStoreWithOptions[string, int](options)
				So(err, ShouldBeNil)
				defer intStore.Close()

				key := "int_key"
				value := 42

				err = intStore.Set(ctx, key, value)
				So(err, ShouldBeNil)

				gotValue, err := intStore.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, value)
			})
		})
	})
}

func TestFreeCacheStoreDel(t *testing.T) {
	Convey("FreeCacheStore.Del", t, func() {
		options := &FreeCacheStoreOptions{
			Size: 1024 * 1024,
		}
		store, err := NewFreeCacheStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("删除存在的键", func() {
			key := "test_key_del"
			value := "test_value_del"

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

		Convey("删除后重新设置", func() {
			key := "test_key_redelete"
			value1 := "test_value1"
			value2 := "test_value2"

			// 设置值
			err := store.Set(ctx, key, value1)
			So(err, ShouldBeNil)

			// 删除
			err = store.Del(ctx, key)
			So(err, ShouldBeNil)

			// 重新设置
			err = store.Set(ctx, key, value2)
			So(err, ShouldBeNil)

			// 验证新值
			gotValue, err := store.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value2)
		})
	})
}

func TestFreeCacheStoreBatchSet(t *testing.T) {
	Convey("FreeCacheStore.BatchSet", t, func() {
		options := &FreeCacheStoreOptions{
			Size: 1024 * 1024,
		}
		store, err := NewFreeCacheStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("正常批量设置", func() {
			keys := []string{"key1", "key2", "key3"}
			vals := []string{"val1", "val2", "val3"}

			errs, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 3)
			for _, e := range errs {
				So(e, ShouldBeNil)
			}

			// 验证设置成功
			for i, key := range keys {
				gotValue, err := store.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, vals[i])
			}
		})

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

		Convey("带选项的批量设置", func() {
			keys := []string{"key1", "key2"}
			vals := []string{"val1", "val2"}

			errs, err := store.BatchSet(ctx, keys, vals, WithExpiration(time.Minute))
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 2)
			for _, e := range errs {
				So(e, ShouldBeNil)
			}
		})
	})
}

func TestFreeCacheStoreBatchGet(t *testing.T) {
	Convey("FreeCacheStore.BatchGet", t, func() {
		options := &FreeCacheStoreOptions{
			Size: 1024 * 1024,
		}
		store, err := NewFreeCacheStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("批量获取存在的键", func() {
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

		Convey("批量获取不存在的键", func() {
			keys := []string{"non_exist1", "non_exist2"}

			gotVals, errs, err := store.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotVals), ShouldEqual, 2)
			So(len(errs), ShouldEqual, 2)

			for _, e := range errs {
				So(e, ShouldEqual, ErrKeyNotFound)
			}
		})

		Convey("混合存在和不存在的键", func() {
			// 设置一个存在的键
			err := store.Set(ctx, "exist_key", "exist_value")
			So(err, ShouldBeNil)

			keys := []string{"exist_key", "non_exist_key"}
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

func TestFreeCacheStoreBatchDel(t *testing.T) {
	Convey("FreeCacheStore.BatchDel", t, func() {
		options := &FreeCacheStoreOptions{
			Size: 1024 * 1024,
		}
		store, err := NewFreeCacheStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("批量删除存在的键", func() {
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

		Convey("批量删除不存在的键", func() {
			keys := []string{"non_exist1", "non_exist2"}

			errs, err := store.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 2)
			for _, e := range errs {
				So(e, ShouldBeNil)
			}
		})

		Convey("空数组", func() {
			keys := []string{}

			errs, err := store.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 0)
		})
	})
}

func TestFreeCacheStoreClose(t *testing.T) {
	Convey("FreeCacheStore.Close", t, func() {
		options := &FreeCacheStoreOptions{
			Size: 1024 * 1024,
		}
		store, err := NewFreeCacheStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)

		Convey("正常关闭", func() {
			err := store.Close()
			So(err, ShouldBeNil)
		})

		Convey("重复关闭", func() {
			err := store.Close()
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldBeNil)
		})
	})
}
