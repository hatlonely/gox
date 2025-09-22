package store

import (
	"context"
	"testing"

	"github.com/hatlonely/gox/ref"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewObservableStoreWithOptions(t *testing.T) {
	Convey("NewObservableStoreWithOptions", t, func() {
		Convey("创建基本ObservableStore", func() {
			options := &ObservableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Name:          "test_store",
				EnableMetrics: true,
				EnableLogging: false,
				EnableTracing: false,
			}
			store, err := NewObservableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()
		})

		Convey("创建带Logger的ObservableStore", func() {
			options := &ObservableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Logger: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/log",
					Type:      "GetLogger",
					Options:   "test",
				},
				Name:          "test_store_with_logger",
				EnableMetrics: true,
				EnableLogging: true,
				EnableTracing: false,
			}
			store, err := NewObservableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()
		})

		Convey("options为nil时返回错误", func() {
			store, err := NewObservableStoreWithOptions[string, string](nil)
			So(err, ShouldNotBeNil)
			So(store, ShouldBeNil)
		})

		Convey("底层store创建失败时返回错误", func() {
			options := &ObservableStoreOptions{
				Store: &ref.TypeOptions{
					Type: "NonExistentStore",
				},
			}
			store, err := NewObservableStoreWithOptions[string, string](options)
			So(err, ShouldNotBeNil)
			So(store, ShouldBeNil)
		})
	})
}

func TestObservableStoreSet(t *testing.T) {
	Convey("ObservableStore.Set", t, func() {
		Convey("基本设置操作", func() {
			options := &ObservableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Logger: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/log",
					Type:      "GetLogger",
					Options:   "test",
				},
				Name:          "test_store_set_basic",
				EnableMetrics: true,
				EnableLogging: true,
			}
			store, err := NewObservableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			defer store.Close()

			ctx := context.Background()
			err = store.Set(ctx, "test_key", "test_value")
			So(err, ShouldBeNil)
		})

		Convey("条件设置操作", func() {
			options := &ObservableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Logger: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/log",
					Type:      "GetLogger",
					Options:   "test",
				},
				Name:          "test_store_set_condition",
				EnableMetrics: true,
				EnableLogging: true,
			}
			store, err := NewObservableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			defer store.Close()

			ctx := context.Background()
			key := "test_key_ifnotexist"
			value := "test_value"

			err = store.Set(ctx, key, value, WithIfNotExist())
			So(err, ShouldBeNil)

			err = store.Set(ctx, key, "new_value", WithIfNotExist())
			So(err, ShouldEqual, ErrConditionFailed)
		})
	})
}

func TestObservableStoreGet(t *testing.T) {
	Convey("ObservableStore.Get", t, func() {
		options := &ObservableStoreOptions{
			Store: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/store",
				Type:      "MapStore[string,string]",
			},
			Name:          "test_store_get",
			EnableMetrics: true,
			EnableLogging: false,
		}
		store, err := NewObservableStoreWithOptions[string, string](options)
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
	})
}

func TestObservableStoreDel(t *testing.T) {
	Convey("ObservableStore.Del", t, func() {
		options := &ObservableStoreOptions{
			Store: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/store",
				Type:      "MapStore[string,string]",
			},
			Name:          "test_store_del",
			EnableMetrics: true,
			EnableLogging: false,
		}
		store, err := NewObservableStoreWithOptions[string, string](options)
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
	})
}

func TestObservableStoreBatchSet(t *testing.T) {
	Convey("ObservableStore.BatchSet", t, func() {
		options := &ObservableStoreOptions{
			Store: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/store",
				Type:      "MapStore[string,string]",
			},
			Name:          "test_store_batch_set",
			EnableMetrics: true,
			EnableLogging: false,
		}
		store, err := NewObservableStoreWithOptions[string, string](options)
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
			So(err, ShouldEqual, ErrConditionFailed)
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

func TestObservableStoreBatchGet(t *testing.T) {
	Convey("ObservableStore.BatchGet", t, func() {
		options := &ObservableStoreOptions{
			Store: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/store",
				Type:      "MapStore[string,string]",
			},
			Name:          "test_store_batch_get",
			EnableMetrics: true,
			EnableLogging: false,
		}
		store, err := NewObservableStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("批量获取存在的键", func() {
			keys := []string{"key1", "key2", "key3"}
			vals := []string{"val1", "val2", "val3"}

			_, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)

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

		Convey("空数组", func() {
			keys := []string{}

			gotVals, errs, err := store.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotVals), ShouldEqual, 0)
			So(len(errs), ShouldEqual, 0)
		})
	})
}

func TestObservableStoreBatchDel(t *testing.T) {
	Convey("ObservableStore.BatchDel", t, func() {
		options := &ObservableStoreOptions{
			Store: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/store",
				Type:      "MapStore[string,string]",
			},
			Name:          "test_store_batch_del",
			EnableMetrics: true,
			EnableLogging: false,
		}
		store, err := NewObservableStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("批量删除存在的键", func() {
			keys := []string{"key1", "key2", "key3"}
			vals := []string{"val1", "val2", "val3"}

			_, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)

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

func TestObservableStoreClose(t *testing.T) {
	Convey("ObservableStore.Close", t, func() {
		options := &ObservableStoreOptions{
			Store: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/store",
				Type:      "MapStore[string,string]",
			},
			Name:          "test_store_close",
			EnableMetrics: true,
			EnableLogging: false,
		}
		store, err := NewObservableStoreWithOptions[string, string](options)
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

func TestObservableStoreObservation(t *testing.T) {
	Convey("ObservableStore观测功能", t, func() {
		Convey("禁用所有观测功能", func() {
			options := &ObservableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Name:          "test_store_no_observe",
				EnableMetrics: false,
				EnableLogging: false,
				EnableTracing: false,
			}
			store, err := NewObservableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			defer store.Close()

			ctx := context.Background()
			err = store.Set(ctx, "key", "value")
			So(err, ShouldBeNil)

			value, err := store.Get(ctx, "key")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "value")
		})

		Convey("只启用指标收集", func() {
			options := &ObservableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Name:          "metrics_only_store",
				EnableMetrics: true,
				EnableLogging: false,
				EnableTracing: false,
			}
			store, err := NewObservableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			defer store.Close()

			So(store.metrics, ShouldNotBeNil)
			So(store.logger, ShouldBeNil)

			ctx := context.Background()
			err = store.Set(ctx, "key", "value")
			So(err, ShouldBeNil)
		})

		Convey("同时启用多种观测功能", func() {
			options := &ObservableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Logger: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/log",
					Type:      "GetLogger",
					Options:   "test",
				},
				Name:          "full_observable_store",
				EnableMetrics: true,
				EnableLogging: true,
				EnableTracing: true,
			}
			store, err := NewObservableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			defer store.Close()

			So(store.metrics, ShouldNotBeNil)
			So(store.logger, ShouldNotBeNil)
			So(store.enableTracing, ShouldBeTrue)

			ctx := context.Background()
			err = store.Set(ctx, "key", "value")
			So(err, ShouldBeNil)

			value, err := store.Get(ctx, "key")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "value")
		})
	})
}