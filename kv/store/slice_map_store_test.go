package store

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewSliceMapStoreWithOptions(t *testing.T) {
	Convey("NewSliceMapStoreWithOptions", t, func() {
		Convey("使用默认配置创建", func() {
			store := NewSliceMapStoreWithOptions[string, string](nil)
			So(store, ShouldNotBeNil)
			So(len(store.s), ShouldEqual, 1024)
			defer store.Close()
		})

		Convey("指定容量创建", func() {
			options := &SliceMapStoreOptions{N: 100}
			store := NewSliceMapStoreWithOptions[string, string](options)
			So(store, ShouldNotBeNil)
			So(len(store.s), ShouldEqual, 100)
			defer store.Close()
		})

		Convey("支持不同数据类型", func() {
			Convey("int类型值", func() {
				store := NewSliceMapStoreWithOptions[string, int](nil)
				So(store, ShouldNotBeNil)
				defer store.Close()
			})

			Convey("int类型键", func() {
				store := NewSliceMapStoreWithOptions[int, string](nil)
				So(store, ShouldNotBeNil)
				defer store.Close()
			})
		})
	})
}

func TestSliceMapStoreSet(t *testing.T) {
	Convey("SliceMapStore.Set", t, func() {
		options := &SliceMapStoreOptions{N: 10}
		store := NewSliceMapStoreWithOptions[string, string](options)
		defer store.Close()

		ctx := context.Background()

		Convey("基本设置操作", func() {
			key := "test_key"
			value := "test_value"

			err := store.Set(ctx, key, value)
			So(err, ShouldBeNil)
		})

		Convey("条件设置 - IfNotExist", func() {
			key := "test_key_ifnotexist"
			value := "test_value"

			Convey("键不存在时设置成功", func() {
				err := store.Set(ctx, key, value, WithIfNotExist())
				So(err, ShouldBeNil)
			})

			Convey("键存在时返回条件失败错误", func() {
				// 先设置一个值
				err := store.Set(ctx, key, "original_value")
				So(err, ShouldBeNil)

				// 尝试条件设置应该失败
				err = store.Set(ctx, key, "new_value", WithIfNotExist())
				So(err, ShouldEqual, ErrConditionFailed)

				// 验证值没有被覆盖
				gotValue, err := store.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, "original_value")
			})
		})

		Convey("覆盖已存在的键", func() {
			key := "test_key_overwrite"
			value1 := "test_value1"
			value2 := "test_value2"

			// 设置初始值
			err := store.Set(ctx, key, value1)
			So(err, ShouldBeNil)

			// 覆盖值
			err = store.Set(ctx, key, value2)
			So(err, ShouldBeNil)

			// 验证值被覆盖
			gotValue, err := store.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value2)
		})

		Convey("容量限制测试", func() {
			// 填满存储
			for i := 0; i < 10; i++ {
				err := store.Set(ctx, string(rune('a'+i)), "value")
				So(err, ShouldBeNil)
			}

			// 尝试添加第11个元素应该失败
			err := store.Set(ctx, "overflow", "value")
			So(err, ShouldEqual, ErrConditionFailed)
		})
	})
}

func TestSliceMapStoreGet(t *testing.T) {
	Convey("SliceMapStore.Get", t, func() {
		store := NewSliceMapStoreWithOptions[string, string](nil)
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

		Convey("不同数据类型", func() {
			Convey("int类型值", func() {
				intStore := NewSliceMapStoreWithOptions[string, int](nil)
				defer intStore.Close()

				key := "int_key"
				value := 42

				err := intStore.Set(ctx, key, value)
				So(err, ShouldBeNil)

				gotValue, err := intStore.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, value)
			})
		})
	})
}

func TestSliceMapStoreDel(t *testing.T) {
	Convey("SliceMapStore.Del", t, func() {
		store := NewSliceMapStoreWithOptions[string, string](nil)
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

		Convey("测试索引复用", func() {
			options := &SliceMapStoreOptions{N: 5}
			testStore := NewSliceMapStoreWithOptions[string, string](options)
			defer testStore.Close()

			// 填满存储
			keys := []string{"key1", "key2", "key3", "key4", "key5"}
			for _, key := range keys {
				err := testStore.Set(ctx, key, "value")
				So(err, ShouldBeNil)
			}

			// 删除一个键
			err := testStore.Del(ctx, "key3")
			So(err, ShouldBeNil)

			// 应该能够添加新键（复用被删除的索引）
			err = testStore.Set(ctx, "new_key", "new_value")
			So(err, ShouldBeNil)

			// 验证新键能够正常获取
			gotValue, err := testStore.Get(ctx, "new_key")
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, "new_value")
		})
	})
}

func TestSliceMapStoreBatchSet(t *testing.T) {
	Convey("SliceMapStore.BatchSet", t, func() {
		store := NewSliceMapStoreWithOptions[string, string](nil)
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
			So(err, ShouldEqual, ErrConditionFailed)
		})

		Convey("空数组", func() {
			keys := []string{}
			vals := []string{}

			errs, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 0)
		})

		Convey("带IfNotExist选项的批量设置", func() {
			keys := []string{"key1", "key2", "key3"}
			vals := []string{"val1", "val2", "val3"}

			// 先设置key2
			err := store.Set(ctx, "key2", "existing_value")
			So(err, ShouldBeNil)

			errs, err := store.BatchSet(ctx, keys, vals, WithIfNotExist())
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 3)

			// key1 和 key3 应该设置成功
			So(errs[0], ShouldBeNil)
			So(errs[2], ShouldBeNil)
			// key2 应该失败
			So(errs[1], ShouldEqual, ErrConditionFailed)

			// 验证结果
			gotValue, err := store.Get(ctx, "key1")
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, "val1")

			gotValue, err = store.Get(ctx, "key2")
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, "existing_value") // 应该保持原值

			gotValue, err = store.Get(ctx, "key3")
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, "val3")
		})

		Convey("容量限制测试", func() {
			options := &SliceMapStoreOptions{N: 3}
			testStore := NewSliceMapStoreWithOptions[string, string](options)
			defer testStore.Close()

			keys := []string{"key1", "key2", "key3", "key4"}
			vals := []string{"val1", "val2", "val3", "val4"}

			errs, err := testStore.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 4)

			// 前三个应该成功
			So(errs[0], ShouldBeNil)
			So(errs[1], ShouldBeNil)
			So(errs[2], ShouldBeNil)
			// 第四个应该失败（容量不足）
			So(errs[3], ShouldEqual, ErrConditionFailed)
		})
	})
}

func TestSliceMapStoreBatchGet(t *testing.T) {
	Convey("SliceMapStore.BatchGet", t, func() {
		store := NewSliceMapStoreWithOptions[string, string](nil)
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

func TestSliceMapStoreBatchDel(t *testing.T) {
	Convey("SliceMapStore.BatchDel", t, func() {
		store := NewSliceMapStoreWithOptions[string, string](nil)
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

func TestSliceMapStoreClose(t *testing.T) {
	Convey("SliceMapStore.Close", t, func() {
		store := NewSliceMapStoreWithOptions[string, string](nil)

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