package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hatlonely/gox/ref"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewLevelDBStoreWithOptions(t *testing.T) {
	Convey("NewLevelDBStoreWithOptions", t, func() {
		Convey("使用默认配置创建", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			options := &LevelDBStoreOptions{
				DBPath: tempDir,
			}

			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()
		})

		Convey("指定JSON序列化器", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			options := &LevelDBStoreOptions{
				DBPath: tempDir,
				KeySerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "JSONSerializer[string]",
				},
				ValSerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "JSONSerializer[string]",
				},
			}

			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()
		})

		Convey("指定BSON序列化器", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			options := &LevelDBStoreOptions{
				DBPath: tempDir,
				KeySerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "BSONSerializer[string]",
				},
				ValSerializer: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/serializer",
					Type:      "BSONSerializer[string]",
				},
			}

			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()
		})

		Convey("支持不同数据类型", func() {
			Convey("int类型值", func() {
				tempDir, err := os.MkdirTemp("", "leveldb_test_*")
				So(err, ShouldBeNil)
				defer os.RemoveAll(tempDir)

				options := &LevelDBStoreOptions{
					DBPath: tempDir,
				}
				store, err := NewLevelDBStoreWithOptions[string, int](options)
				So(err, ShouldBeNil)
				So(store, ShouldNotBeNil)
				defer store.Close()
			})

			Convey("int类型键", func() {
				tempDir, err := os.MkdirTemp("", "leveldb_test_*")
				So(err, ShouldBeNil)
				defer os.RemoveAll(tempDir)

				options := &LevelDBStoreOptions{
					DBPath: tempDir,
				}
				store, err := NewLevelDBStoreWithOptions[int, string](options)
				So(err, ShouldBeNil)
				So(store, ShouldNotBeNil)
				defer store.Close()
			})
		})

		Convey("生成数据库路径后缀", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			options := &LevelDBStoreOptions{
				DBPath:               filepath.Join(tempDir, "test_db"),
				GenerateDBPathSuffix: true,
			}

			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()
		})

		Convey("设置快照类型", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			options := &LevelDBStoreOptions{
				DBPath:       tempDir,
				SnapshotType: "zip",
			}

			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()
		})

		Convey("LevelDB特定配置选项", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			options := &LevelDBStoreOptions{
				DBPath:                     tempDir,
				BlockCacher:                "lru",
				BlockCacheCapacity:         8 * 1024 * 1024, // 8MB
				Compression:                "snappy",
				WriteBuffer:                4 * 1024 * 1024, // 4MB
				CompactionL0Trigger:        4,
				CompactionTableSize:        2 * 1024 * 1024, // 2MB
				OpenFilesCacheCapacity:     500,
				DisableBlockCache:          false,
				NoSync:                     false,
			}

			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			defer store.Close()
		})
	})
}

func TestLevelDBStoreSet(t *testing.T) {
	Convey("LevelDBStore.Set", t, func() {
		tempDir, err := os.MkdirTemp("", "leveldb_test_*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)

		options := &LevelDBStoreOptions{
			DBPath: tempDir,
		}
		store, err := NewLevelDBStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer store.Close()

		ctx := context.Background()

		Convey("基本设置操作", func() {
			key := "test_key"
			value := "test_value"

			err := store.Set(ctx, key, value)
			So(err, ShouldBeNil)

			// 验证设置成功
			gotValue, err := store.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value)
		})

		Convey("条件设置 - IfNotExist", func() {
			key := "test_key_ifnotexist"
			value := "test_value"

			Convey("键不存在时设置成功", func() {
				err := store.Set(ctx, key, value, WithIfNotExist())
				So(err, ShouldBeNil)

				// 验证设置成功
				gotValue, err := store.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, value)
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

			// 设置第一个值
			err := store.Set(ctx, key, value1)
			So(err, ShouldBeNil)

			// 覆盖为第二个值
			err = store.Set(ctx, key, value2)
			So(err, ShouldBeNil)

			// 验证值已被覆盖
			gotValue, err := store.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value2)
		})
	})
}

func TestLevelDBStoreGet(t *testing.T) {
	Convey("LevelDBStore.Get", t, func() {
		tempDir, err := os.MkdirTemp("", "leveldb_test_*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)

		options := &LevelDBStoreOptions{
			DBPath: tempDir,
		}
		store, err := NewLevelDBStoreWithOptions[string, string](options)
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

		Convey("不同数据类型", func() {
			Convey("int类型值", func() {
				tempDir2, err := os.MkdirTemp("", "leveldb_test_*")
				So(err, ShouldBeNil)
				defer os.RemoveAll(tempDir2)

				options := &LevelDBStoreOptions{
					DBPath: tempDir2,
				}
				intStore, err := NewLevelDBStoreWithOptions[string, int](options)
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

		Convey("获取已删除的键", func() {
			key := "test_key_deleted"
			value := "test_value"

			// 设置值
			err := store.Set(ctx, key, value)
			So(err, ShouldBeNil)

			// 删除键
			err = store.Del(ctx, key)
			So(err, ShouldBeNil)

			// 尝试获取应该失败
			_, err = store.Get(ctx, key)
			So(err, ShouldEqual, ErrKeyNotFound)
		})
	})
}

func TestLevelDBStoreDel(t *testing.T) {
	Convey("LevelDBStore.Del", t, func() {
		tempDir, err := os.MkdirTemp("", "leveldb_test_*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)

		options := &LevelDBStoreOptions{
			DBPath: tempDir,
		}
		store, err := NewLevelDBStoreWithOptions[string, string](options)
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

func TestLevelDBStoreBatchSet(t *testing.T) {
	Convey("LevelDBStore.BatchSet", t, func() {
		tempDir, err := os.MkdirTemp("", "leveldb_test_*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)

		options := &LevelDBStoreOptions{
			DBPath: tempDir,
		}
		store, err := NewLevelDBStoreWithOptions[string, string](options)
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

		Convey("带条件的批量设置", func() {
			keys := []string{"key1", "key2"}
			vals := []string{"val1", "val2"}

			// 先设置一个键
			err := store.Set(ctx, "key1", "existing_value")
			So(err, ShouldBeNil)

			errs, err := store.BatchSet(ctx, keys, vals, WithIfNotExist())
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 2)
			So(errs[0], ShouldEqual, ErrConditionFailed) // key1 已存在
			So(errs[1], ShouldBeNil)                      // key2 不存在，设置成功

			// 验证 key1 没有被覆盖
			gotValue, err := store.Get(ctx, "key1")
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, "existing_value")

			// 验证 key2 设置成功
			gotValue, err = store.Get(ctx, "key2")
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, "val2")
		})
	})
}

func TestLevelDBStoreBatchGet(t *testing.T) {
	Convey("LevelDBStore.BatchGet", t, func() {
		tempDir, err := os.MkdirTemp("", "leveldb_test_*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)

		options := &LevelDBStoreOptions{
			DBPath: tempDir,
		}
		store, err := NewLevelDBStoreWithOptions[string, string](options)
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

func TestLevelDBStoreBatchDel(t *testing.T) {
	Convey("LevelDBStore.BatchDel", t, func() {
		tempDir, err := os.MkdirTemp("", "leveldb_test_*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)

		options := &LevelDBStoreOptions{
			DBPath: tempDir,
		}
		store, err := NewLevelDBStoreWithOptions[string, string](options)
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

		Convey("混合存在和不存在的键", func() {
			// 设置一个存在的键
			err := store.Set(ctx, "exist_key", "exist_value")
			So(err, ShouldBeNil)

			keys := []string{"exist_key", "non_exist_key"}
			errs, err := store.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 2)
			for _, e := range errs {
				So(e, ShouldBeNil) // 删除不存在的键也返回成功
			}

			// 验证存在的键被删除
			_, err = store.Get(ctx, "exist_key")
			So(err, ShouldEqual, ErrKeyNotFound)
		})
	})
}

func TestLevelDBStoreClose(t *testing.T) {
	Convey("LevelDBStore.Close", t, func() {
		Convey("正常关闭", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			options := &LevelDBStoreOptions{
				DBPath: tempDir,
			}
			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldBeNil)
		})

		Convey("重复关闭", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			options := &LevelDBStoreOptions{
				DBPath: tempDir,
			}
			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldBeNil)
		})

		Convey("关闭时创建快照", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			dbPath := filepath.Join(tempDir, "test_db")
			options := &LevelDBStoreOptions{
				DBPath:       dbPath,
				SnapshotType: "zip",
			}
			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)

			ctx := context.Background()
			// 写入一些数据
			err = store.Set(ctx, "key1", "value1")
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldBeNil)

			// 检查是否创建了快照文件
			files, err := filepath.Glob(dbPath + ".*.zip")
			So(err, ShouldBeNil)
			So(len(files), ShouldBeGreaterThan, 0)
		})

		Convey("关闭时创建tar.gz快照", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			dbPath := filepath.Join(tempDir, "test_db")
			options := &LevelDBStoreOptions{
				DBPath:       dbPath,
				SnapshotType: "tar.gz",
			}
			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)

			ctx := context.Background()
			// 写入一些数据
			err = store.Set(ctx, "key1", "value1")
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldBeNil)

			// 检查是否创建了快照文件
			files, err := filepath.Glob(dbPath + ".*.tar.gz")
			So(err, ShouldBeNil)
			So(len(files), ShouldBeGreaterThan, 0)
		})

		Convey("不支持的快照类型", func() {
			tempDir, err := os.MkdirTemp("", "leveldb_test_*")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tempDir)

			options := &LevelDBStoreOptions{
				DBPath:       tempDir,
				SnapshotType: "unsupported",
			}
			store, err := NewLevelDBStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)

			err = store.Close()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "unsupported snapshot type")
		})
	})
}