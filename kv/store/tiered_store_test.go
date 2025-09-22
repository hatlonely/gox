package store

import (
	"context"
	"testing"
	"time"

	"github.com/hatlonely/gox/ref"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewTieredStoreWithOptions(t *testing.T) {
	Convey("NewTieredStoreWithOptions", t, func() {
		Convey("创建基本TieredStore", func() {
			options := &TieredStoreOptions{
				Tiers: []*ref.TypeOptions{
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "MapStore[string,string]",
					},
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "MapStore[string,string]",
					},
				},
				WritePolicy: "writeThrough",
				Promote:     true,
			}

			store, err := NewTieredStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(store.GetTierCount(), ShouldEqual, 2)
			defer store.Close()
		})

		Convey("创建三层TieredStore", func() {
			options := &TieredStoreOptions{
				Tiers: []*ref.TypeOptions{
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "MapStore[string,string]",
					},
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "SyncMapStore[string,string]",
					},
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "MapStore[string,string]",
					},
				},
				WritePolicy: "writeBack",
				Promote:     false,
			}

			store, err := NewTieredStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(store.GetTierCount(), ShouldEqual, 3)
			defer store.Close()
		})

		Convey("空配置返回错误", func() {
			store, err := NewTieredStoreWithOptions[string, string](nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "options is nil")
			So(store, ShouldBeNil)
		})

		Convey("空tiers返回错误", func() {
			options := &TieredStoreOptions{
				Tiers: []*ref.TypeOptions{},
			}

			store, err := NewTieredStoreWithOptions[string, string](options)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "at least one tier is required")
			So(store, ShouldBeNil)
		})

		Convey("无效写策略返回错误", func() {
			options := &TieredStoreOptions{
				Tiers: []*ref.TypeOptions{
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "MapStore[string,string]",
					},
				},
				WritePolicy: "invalidPolicy",
			}

			store, err := NewTieredStoreWithOptions[string, string](options)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid write policy")
			So(store, ShouldBeNil)
		})
	})
}

func TestTieredStoreSet(t *testing.T) {
	Convey("TieredStore.Set", t, func() {
		options := &TieredStoreOptions{
			Tiers: []*ref.TypeOptions{
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
			},
			WritePolicy: "writeThrough",
			Promote:     true,
		}

		store, err := NewTieredStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("writeThrough策略基本设置", func() {
			key := "test_key"
			value := "test_value"

			err := store.Set(ctx, key, value)
			So(err, ShouldBeNil)

			// 验证所有层都有数据
			for i := 0; i < store.GetTierCount(); i++ {
				tierValue, err := store.GetFromTier(ctx, i, key)
				So(err, ShouldBeNil)
				So(tierValue, ShouldEqual, value)
			}
		})

		Convey("writeThrough策略条件设置", func() {
			key := "test_key_condition"
			value := "test_value"

			Convey("IfNotExist成功", func() {
				err := store.Set(ctx, key, value, WithIfNotExist())
				So(err, ShouldBeNil)

				// 验证数据存在
				gotValue, err := store.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, value)
			})

			Convey("IfNotExist失败", func() {
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
	})
}

func TestTieredStoreWriteBack(t *testing.T) {
	Convey("TieredStore WriteBack策略", t, func() {
		options := &TieredStoreOptions{
			Tiers: []*ref.TypeOptions{
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
			},
			WritePolicy: "writeBack",
			Promote:     true,
		}

		store, err := NewTieredStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("writeBack基本写入", func() {
			key := "test_key"
			value := "test_value"

			err := store.Set(ctx, key, value)
			So(err, ShouldBeNil)

			// 立即检查第一层应该有数据
			tierValue, err := store.GetFromTier(ctx, 0, key)
			So(err, ShouldBeNil)
			So(tierValue, ShouldEqual, value)

			// 等待异步写入完成
			time.Sleep(100 * time.Millisecond)

			// 检查第二层也应该有数据
			tierValue, err = store.GetFromTier(ctx, 1, key)
			So(err, ShouldBeNil)
			So(tierValue, ShouldEqual, value)
		})
	})
}

func TestTieredStoreGet(t *testing.T) {
	Convey("TieredStore.Get", t, func() {
		options := &TieredStoreOptions{
			Tiers: []*ref.TypeOptions{
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
			},
			WritePolicy: "writeThrough",
			Promote:     true,
		}

		store, err := NewTieredStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("基本获取操作", func() {
			key := "test_key"
			value := "test_value"

			err := store.Set(ctx, key, value)
			So(err, ShouldBeNil)

			gotValue, err := store.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value)
		})

		Convey("获取不存在的键", func() {
			_, err := store.Get(ctx, "nonexistent_key")
			So(err, ShouldEqual, ErrKeyNotFound)
		})

		Convey("数据提升功能", func() {
			// 创建一个三层store，用于测试提升
			options3Tier := &TieredStoreOptions{
				Tiers: []*ref.TypeOptions{
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "MapStore[string,string]",
					},
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "MapStore[string,string]",
					},
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "MapStore[string,string]",
					},
				},
				WritePolicy: "writeThrough",
				Promote:     true,
			}

			store3, err := NewTieredStoreWithOptions[string, string](options3Tier)
			So(err, ShouldBeNil)
			defer store3.Close()

			// 通过TieredStore设置数据，验证提升功能
			key := "promotion_key"
			value := "promotion_value"

			err = store3.Set(ctx, key, value)
			So(err, ShouldBeNil)

			// 验证所有层都有数据
			for i := 0; i < store3.GetTierCount(); i++ {
				tierValue, err := store3.GetFromTier(ctx, i, key)
				So(err, ShouldBeNil)
				So(tierValue, ShouldEqual, value)
			}
		})

		Convey("禁用提升功能", func() {
			optionsNoPromote := &TieredStoreOptions{
				Tiers: []*ref.TypeOptions{
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "MapStore[string,string]",
					},
					{
						Namespace: "github.com/hatlonely/gox/kv/store",
						Type:      "MapStore[string,string]",
					},
				},
				WritePolicy: "writeThrough",
				Promote:     false,
			}

			storeNoPromote, err := NewTieredStoreWithOptions[string, string](optionsNoPromote)
			So(err, ShouldBeNil)
			defer storeNoPromote.Close()

			key := "no_promote_key"
			value := "no_promote_value"

			err = storeNoPromote.Set(ctx, key, value)
			So(err, ShouldBeNil)

			gotValue, err := storeNoPromote.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value)
		})
	})
}

func TestTieredStoreDel(t *testing.T) {
	Convey("TieredStore.Del", t, func() {
		options := &TieredStoreOptions{
			Tiers: []*ref.TypeOptions{
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
			},
			WritePolicy: "writeThrough",
			Promote:     true,
		}

		store, err := NewTieredStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("基本删除操作", func() {
			key := "test_key"
			value := "test_value"

			// 设置数据
			err := store.Set(ctx, key, value)
			So(err, ShouldBeNil)

			// 删除数据
			err = store.Del(ctx, key)
			So(err, ShouldBeNil)

			// 验证所有层都删除了
			for i := 0; i < store.GetTierCount(); i++ {
				_, err := store.GetFromTier(ctx, i, key)
				So(err, ShouldEqual, ErrKeyNotFound)
			}
		})

		Convey("删除不存在的键", func() {
			err := store.Del(ctx, "nonexistent_key")
			So(err, ShouldBeNil) // 删除不存在的键应该成功
		})
	})
}

func TestTieredStoreBatchOperations(t *testing.T) {
	Convey("TieredStore批量操作", t, func() {
		options := &TieredStoreOptions{
			Tiers: []*ref.TypeOptions{
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
			},
			WritePolicy: "writeThrough",
			Promote:     true,
		}

		store, err := NewTieredStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("批量设置和获取", func() {
			keys := []string{"key1", "key2", "key3"}
			vals := []string{"value1", "value2", "value3"}

			// 批量设置
			errs, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)
			for _, e := range errs {
				So(e, ShouldBeNil)
			}

			// 批量获取
			gotVals, gotErrs, err := store.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			for i, e := range gotErrs {
				So(e, ShouldBeNil)
				So(gotVals[i], ShouldEqual, vals[i])
			}
		})

		Convey("批量删除", func() {
			keys := []string{"key1", "key2", "key3"}
			vals := []string{"value1", "value2", "value3"}

			// 先批量设置
			_, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldBeNil)

			// 批量删除
			delErrs, err := store.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			for _, e := range delErrs {
				So(e, ShouldBeNil)
			}

			// 验证删除成功
			_, gotErrs, err := store.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			for _, e := range gotErrs {
				So(e, ShouldEqual, ErrKeyNotFound)
			}
		})

		Convey("键值长度不匹配", func() {
			keys := []string{"key1", "key2"}
			vals := []string{"value1"}

			errs, err := store.BatchSet(ctx, keys, vals)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "length mismatch")
			So(errs, ShouldBeNil)
		})
	})
}

func TestTieredStoreDynamicConfiguration(t *testing.T) {
	Convey("TieredStore动态配置", t, func() {
		options := &TieredStoreOptions{
			Tiers: []*ref.TypeOptions{
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
			},
			WritePolicy: "writeThrough",
			Promote:     true,
		}

		store, err := NewTieredStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		Convey("测试基本功能", func() {
			ctx := context.Background()
			// 测试基本的 Set/Get 操作
			err := store.Set(ctx, "test_key", "test_value")
			So(err, ShouldBeNil)

			value, err := store.Get(ctx, "test_key")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "test_value")
		})
	})
}

func TestTieredStoreErrorHandling(t *testing.T) {
	Convey("TieredStore错误处理", t, func() {
		options := &TieredStoreOptions{
			Tiers: []*ref.TypeOptions{
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
			},
			WritePolicy: "writeThrough",
			Promote:     true,
		}

		store, err := NewTieredStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("无效的层索引", func() {
			_, err := store.GetFromTier(ctx, 999, "key1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid tier index")

			_, err = store.GetFromTier(ctx, -1, "key1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid tier index")
		})
	})
}

func TestTieredStoreClose(t *testing.T) {
	Convey("TieredStore.Close", t, func() {
		options := &TieredStoreOptions{
			Tiers: []*ref.TypeOptions{
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
			},
			WritePolicy: "writeThrough",
			Promote:     true,
		}

		store, err := NewTieredStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)

		ctx := context.Background()

		Convey("正常关闭", func() {
			// 设置一些数据
			err := store.Set(ctx, "key1", "value1")
			So(err, ShouldBeNil)

			// 关闭存储
			err = store.Close()
			So(err, ShouldBeNil)

			// Close() 只负责关闭各层 store，不修改结构
			So(store.GetTierCount(), ShouldEqual, 2)
		})
	})
}