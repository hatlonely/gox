package store

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hatlonely/gox/kv/loader"
	"github.com/hatlonely/gox/kv/parser"
	"github.com/hatlonely/gox/ref"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewLoadableStoreWithOptions(t *testing.T) {
	Convey("NewLoadableStoreWithOptions", t, func() {
		Convey("创建基本LoadableStore", func() {
			tmpDir := os.TempDir()
			testFile := filepath.Join(tmpDir, "test_loadable_store.txt")
			defer os.RemoveAll(testFile)

			// 创建测试文件
			content := "key1\tvalue1\nkey2\tvalue2\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			options := &LoadableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Loader: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/loader",
					Type:      "KVFileLoader[string,string]",
					Options: &loader.KVFileLoaderOptions{
						FilePath: testFile,
						Parser: &ref.TypeOptions{
							Namespace: "github.com/hatlonely/gox/kv/parser",
							Type:      "LineParser[string,string]",
							Options: &parser.LineParserOptions{
								Separator: "\t",
							},
						},
					},
				},
				LoadStrategy: loader.LoadStrategyInPlace,
				CloseDelay:   time.Second,
			}

			loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(loadableStore, ShouldNotBeNil)
			defer loadableStore.Close()

			// 验证初始数据已加载
			ctx := context.Background()
			value1, err := loadableStore.Get(ctx, "key1")
			So(err, ShouldBeNil)
			So(value1, ShouldEqual, "value1")

			value2, err := loadableStore.Get(ctx, "key2")
			So(err, ShouldBeNil)
			So(value2, ShouldEqual, "value2")
		})

		Convey("空配置返回错误", func() {
			loadableStore, err := NewLoadableStoreWithOptions[string, string](nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "options is nil")
			So(loadableStore, ShouldBeNil)
		})

		Convey("无效加载策略返回错误", func() {
			options := &LoadableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Loader: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/loader",
					Type:      "KVFileLoader[string,string]",
					Options: &loader.KVFileLoaderOptions{
						FilePath: "/tmp/test.txt",
					},
				},
				LoadStrategy: "invalid_strategy",
			}

			loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid load strategy")
			So(loadableStore, ShouldBeNil)
		})

		Convey("Store创建失败", func() {
			options := &LoadableStoreOptions{
				Store: &ref.TypeOptions{
					Type: "InvalidStore",
				},
				Loader: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/loader",
					Type:      "KVFileLoader[string,string]",
				},
				LoadStrategy: loader.LoadStrategyInPlace,
			}

			loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "failed to create store")
			So(loadableStore, ShouldBeNil)
		})

		Convey("Loader创建失败", func() {
			options := &LoadableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Loader: &ref.TypeOptions{
					Type: "InvalidLoader",
				},
				LoadStrategy: loader.LoadStrategyInPlace,
			}

			loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "failed to create loader")
			So(loadableStore, ShouldBeNil)
		})

		Convey("Replace策略", func() {
			tmpDir := os.TempDir()
			testFile := filepath.Join(tmpDir, "test_replace.txt")
			defer os.RemoveAll(testFile)

			content := "key1\tvalue1\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			options := &LoadableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Loader: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/loader",
					Type:      "KVFileLoader[string,string]",
					Options: &loader.KVFileLoaderOptions{
						FilePath: testFile,
						Parser: &ref.TypeOptions{
							Namespace: "github.com/hatlonely/gox/kv/parser",
							Type:      "LineParser[string,string]",
							Options: &parser.LineParserOptions{
								Separator: "\t",
							},
						},
					},
				},
				LoadStrategy: loader.LoadStrategyReplace,
				CloseDelay:   100 * time.Millisecond,
			}

			loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(loadableStore, ShouldNotBeNil)
			defer loadableStore.Close()
		})
	})
}

func TestLoadableStoreBasicOperations(t *testing.T) {
	Convey("LoadableStore基本Store接口操作", t, func() {
		tmpDir := os.TempDir()
		testFile := filepath.Join(tmpDir, "test_basic_ops.txt")
		defer os.RemoveAll(testFile)

		// 创建初始文件
		content := "initial_key\tinitial_value\n"
		err := os.WriteFile(testFile, []byte(content), 0644)
		So(err, ShouldBeNil)

		options := &LoadableStoreOptions{
			Store: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/store",
				Type:      "MapStore[string,string]",
			},
			Loader: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/loader",
				Type:      "KVFileLoader[string,string]",
				Options: &loader.KVFileLoaderOptions{
					FilePath: testFile,
					Parser: &ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/kv/parser",
						Type:      "LineParser[string,string]",
						Options: &parser.LineParserOptions{
							Separator: "\t",
						},
					},
				},
			},
			LoadStrategy: loader.LoadStrategyInPlace,
		}

		loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer loadableStore.Close()

		ctx := context.Background()

		Convey("Set和Get操作", func() {
			key := "test_key"
			value := "test_value"

			err := loadableStore.Set(ctx, key, value)
			So(err, ShouldBeNil)

			gotValue, err := loadableStore.Get(ctx, key)
			So(err, ShouldBeNil)
			So(gotValue, ShouldEqual, value)
		})

		Convey("Del操作", func() {
			key := "del_key"
			value := "del_value"

			// 先设置
			err := loadableStore.Set(ctx, key, value)
			So(err, ShouldBeNil)

			// 验证存在
			_, err = loadableStore.Get(ctx, key)
			So(err, ShouldBeNil)

			// 删除
			err = loadableStore.Del(ctx, key)
			So(err, ShouldBeNil)

			// 验证不存在
			_, err = loadableStore.Get(ctx, key)
			So(err, ShouldEqual, ErrKeyNotFound)
		})

		Convey("BatchSet操作", func() {
			keys := []string{"batch1", "batch2", "batch3"}
			values := []string{"value1", "value2", "value3"}

			errs, err := loadableStore.BatchSet(ctx, keys, values)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 3)
			for _, e := range errs {
				So(e, ShouldBeNil)
			}

			// 验证设置成功
			for i, key := range keys {
				gotValue, err := loadableStore.Get(ctx, key)
				So(err, ShouldBeNil)
				So(gotValue, ShouldEqual, values[i])
			}
		})

		Convey("BatchGet操作", func() {
			keys := []string{"batch1", "batch2", "non_exist"}
			values := []string{"value1", "value2"}

			// 先设置部分键
			_, err := loadableStore.BatchSet(ctx, keys[:2], values)
			So(err, ShouldBeNil)

			// 批量获取
			gotValues, errs, err := loadableStore.BatchGet(ctx, keys)
			So(err, ShouldBeNil)
			So(len(gotValues), ShouldEqual, 3)
			So(len(errs), ShouldEqual, 3)

			So(errs[0], ShouldBeNil)
			So(gotValues[0], ShouldEqual, "value1")
			So(errs[1], ShouldBeNil)
			So(gotValues[1], ShouldEqual, "value2")
			So(errs[2], ShouldEqual, ErrKeyNotFound)
		})

		Convey("BatchDel操作", func() {
			keys := []string{"del1", "del2", "del3"}
			values := []string{"value1", "value2", "value3"}

			// 先设置
			_, err := loadableStore.BatchSet(ctx, keys, values)
			So(err, ShouldBeNil)

			// 批量删除
			errs, err := loadableStore.BatchDel(ctx, keys)
			So(err, ShouldBeNil)
			So(len(errs), ShouldEqual, 3)
			for _, e := range errs {
				So(e, ShouldBeNil)
			}

			// 验证删除成功
			for _, key := range keys {
				_, err := loadableStore.Get(ctx, key)
				So(err, ShouldEqual, ErrKeyNotFound)
			}
		})

		Convey("关闭后操作返回错误", func() {
			key := "test_key"
			value := "test_value"

			// 关闭store
			err := loadableStore.Close()
			So(err, ShouldBeNil)

			// 所有操作都应该返回错误
			err = loadableStore.Set(ctx, key, value)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "store is closed")

			_, err = loadableStore.Get(ctx, key)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "store is closed")

			err = loadableStore.Del(ctx, key)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "store is closed")

			_, err = loadableStore.BatchSet(ctx, []string{key}, []string{value})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "store is closed")

			_, _, err = loadableStore.BatchGet(ctx, []string{key})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "store is closed")

			_, err = loadableStore.BatchDel(ctx, []string{key})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "store is closed")
		})
	})
}

func TestLoadableStoreInPlaceStrategy(t *testing.T) {
	Convey("LoadableStore InPlace加载策略", t, func() {
		tmpDir := os.TempDir()
		testFile := filepath.Join(tmpDir, "test_inplace.txt")
		defer os.RemoveAll(testFile)

		// 创建初始文件
		content := "key1\tvalue1\nkey2\tvalue2\n"
		err := os.WriteFile(testFile, []byte(content), 0644)
		So(err, ShouldBeNil)

		options := &LoadableStoreOptions{
			Store: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/store",
				Type:      "MapStore[string,string]",
			},
			Loader: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/loader",
				Type:      "KVFileLoader[string,string]",
				Options: &loader.KVFileLoaderOptions{
					FilePath: testFile,
					Parser: &ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/kv/parser",
						Type:      "LineParser[string,string]",
						Options: &parser.LineParserOptions{
							Separator: "\t",
						},
					},
				},
			},
			LoadStrategy: loader.LoadStrategyInPlace,
		}

		loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer loadableStore.Close()

		ctx := context.Background()

		Convey("初始数据加载", func() {
			value1, err := loadableStore.Get(ctx, "key1")
			So(err, ShouldBeNil)
			So(value1, ShouldEqual, "value1")

			value2, err := loadableStore.Get(ctx, "key2")
			So(err, ShouldBeNil)
			So(value2, ShouldEqual, "value2")
		})

		Convey("文件更新触发增量加载", func() {
			// 更新文件内容
			newContent := "key1\tvalue1_updated\nkey2\tvalue2\nkey3\tvalue3\n"
			err := os.WriteFile(testFile, []byte(newContent), 0644)
			So(err, ShouldBeNil)

			// 等待文件变化被检测到
			time.Sleep(200 * time.Millisecond)

			// 验证数据更新
			value1, err := loadableStore.Get(ctx, "key1")
			So(err, ShouldBeNil)
			So(value1, ShouldEqual, "value1_updated")

			value3, err := loadableStore.Get(ctx, "key3")
			So(err, ShouldBeNil)
			So(value3, ShouldEqual, "value3")
		})
	})
}

func TestLoadableStoreReplaceStrategy(t *testing.T) {
	Convey("LoadableStore Replace加载策略", t, func() {
		tmpDir := os.TempDir()
		testFile := filepath.Join(tmpDir, "test_replace.txt")
		defer os.RemoveAll(testFile)

		// 创建初始文件
		content := "key1\tvalue1\nkey2\tvalue2\n"
		err := os.WriteFile(testFile, []byte(content), 0644)
		So(err, ShouldBeNil)

		options := &LoadableStoreOptions{
			Store: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/store",
				Type:      "MapStore[string,string]",
			},
			Loader: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/loader",
				Type:      "KVFileLoader[string,string]",
				Options: &loader.KVFileLoaderOptions{
					FilePath: testFile,
					Parser: &ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/kv/parser",
						Type:      "LineParser[string,string]",
						Options: &parser.LineParserOptions{
							Separator: "\t",
						},
					},
				},
			},
			LoadStrategy: loader.LoadStrategyReplace,
			CloseDelay:   50 * time.Millisecond,
		}

		loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer loadableStore.Close()

		ctx := context.Background()

		Convey("初始数据加载", func() {
			value1, err := loadableStore.Get(ctx, "key1")
			So(err, ShouldBeNil)
			So(value1, ShouldEqual, "value1")

			value2, err := loadableStore.Get(ctx, "key2")
			So(err, ShouldBeNil)
			So(value2, ShouldEqual, "value2")
		})

		Convey("文件更新触发替换加载", func() {
			// 添加一些数据到当前store（模拟业务写入）
			err := loadableStore.Set(ctx, "runtime_key", "runtime_value")
			So(err, ShouldBeNil)

			// 更新文件内容（完全替换）
			newContent := "key1\tvalue1_new\nkey3\tvalue3\n"
			err = os.WriteFile(testFile, []byte(newContent), 0644)
			So(err, ShouldBeNil)

			// 等待文件变化被检测到和替换完成
			time.Sleep(200 * time.Millisecond)

			// 验证新数据
			value1, err := loadableStore.Get(ctx, "key1")
			So(err, ShouldBeNil)
			So(value1, ShouldEqual, "value1_new")

			value3, err := loadableStore.Get(ctx, "key3")
			So(err, ShouldBeNil)
			So(value3, ShouldEqual, "value3")

			// 验证旧数据和运行时数据被清除
			_, err = loadableStore.Get(ctx, "key2")
			So(err, ShouldEqual, ErrKeyNotFound)

			_, err = loadableStore.Get(ctx, "runtime_key")
			So(err, ShouldEqual, ErrKeyNotFound)
		})
	})
}

func TestLoadableStoreConcurrency(t *testing.T) {
	Convey("LoadableStore并发安全性", t, func() {
		tmpDir := os.TempDir()
		testFile := filepath.Join(tmpDir, "test_concurrency.txt")
		defer os.RemoveAll(testFile)

		// 创建初始文件
		content := "key1\tvalue1\nkey2\tvalue2\n"
		err := os.WriteFile(testFile, []byte(content), 0644)
		So(err, ShouldBeNil)

		options := &LoadableStoreOptions{
			Store: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/store",
				Type:      "SyncMapStore[string,string]",
			},
			Loader: &ref.TypeOptions{
				Namespace: "github.com/hatlonely/gox/kv/loader",
				Type:      "KVFileLoader[string,string]",
				Options: &loader.KVFileLoaderOptions{
					FilePath: testFile,
					Parser: &ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/kv/parser",
						Type:      "LineParser[string,string]",
						Options: &parser.LineParserOptions{
							Separator: "\t",
						},
					},
				},
			},
			LoadStrategy: loader.LoadStrategyReplace,
			CloseDelay:   10 * time.Millisecond,
		}

		loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
		So(err, ShouldBeNil)
		defer loadableStore.Close()

		Convey("并发读写操作", func() {
			ctx := context.Background()
			var wg sync.WaitGroup
			errChan := make(chan error, 100)

			// 启动多个读goroutine
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 100; j++ {
						_, err := loadableStore.Get(ctx, "key1")
						if err != nil && err != ErrKeyNotFound {
							errChan <- err
							return
						}
						time.Sleep(time.Microsecond)
					}
				}(i)
			}

			// 启动多个写goroutine
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 50; j++ {
						key := "concurrent_key"
						value := "concurrent_value"
						err := loadableStore.Set(ctx, key, value)
						if err != nil {
							errChan <- err
							return
						}
						time.Sleep(time.Microsecond)
					}
				}(i)
			}

			// 启动文件更新goroutine
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 5; i++ {
					newContent := "key1\tvalue1_updated\nkey2\tvalue2_updated\n"
					err := os.WriteFile(testFile, []byte(newContent), 0644)
					if err != nil {
						errChan <- err
						return
					}
					time.Sleep(50 * time.Millisecond)
				}
			}()

			wg.Wait()
			close(errChan)

			// 检查是否有错误
			for err := range errChan {
				So(err, ShouldBeNil)
			}
		})
	})
}

func TestLoadableStoreClose(t *testing.T) {
	Convey("LoadableStore关闭和资源清理", t, func() {
		tmpDir := os.TempDir()
		testFile := filepath.Join(tmpDir, "test_close.txt")
		defer os.RemoveAll(testFile)

		content := "key1\tvalue1\n"
		err := os.WriteFile(testFile, []byte(content), 0644)
		So(err, ShouldBeNil)

		Convey("正常关闭", func() {
			options := &LoadableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Loader: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/loader",
					Type:      "KVFileLoader[string,string]",
					Options: &loader.KVFileLoaderOptions{
						FilePath: testFile,
						Parser: &ref.TypeOptions{
							Namespace: "github.com/hatlonely/gox/kv/parser",
							Type:      "LineParser[string,string]",
							Options: &parser.LineParserOptions{
								Separator: "\t",
							},
						},
					},
				},
				LoadStrategy: loader.LoadStrategyReplace,
				CloseDelay:   10 * time.Millisecond,
			}

			loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)

			err = loadableStore.Close()
			So(err, ShouldBeNil)
		})

		Convey("重复关闭", func() {
			options := &LoadableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Loader: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/loader",
					Type:      "KVFileLoader[string,string]",
					Options: &loader.KVFileLoaderOptions{
						FilePath: testFile,
						Parser: &ref.TypeOptions{
							Namespace: "github.com/hatlonely/gox/kv/parser",
							Type:      "LineParser[string,string]",
							Options: &parser.LineParserOptions{
								Separator: "\t",
							},
						},
					},
				},
				LoadStrategy: loader.LoadStrategyInPlace,
			}

			loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)

			// 第一次关闭
			err = loadableStore.Close()
			So(err, ShouldBeNil)

			// 第二次关闭应该也成功
			err = loadableStore.Close()
			So(err, ShouldBeNil)
		})

		Convey("延迟关闭验证", func() {
			options := &LoadableStoreOptions{
				Store: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/store",
					Type:      "MapStore[string,string]",
				},
				Loader: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/loader",
					Type:      "KVFileLoader[string,string]",
					Options: &loader.KVFileLoaderOptions{
						FilePath: testFile,
						Parser: &ref.TypeOptions{
							Namespace: "github.com/hatlonely/gox/kv/parser",
							Type:      "LineParser[string,string]",
							Options: &parser.LineParserOptions{
								Separator: "\t",
							},
						},
					},
				},
				LoadStrategy: loader.LoadStrategyReplace,
				CloseDelay:   200 * time.Millisecond,
			}

			loadableStore, err := NewLoadableStoreWithOptions[string, string](options)
			So(err, ShouldBeNil)

			// 触发一次替换，产生旧store
			newContent := "key1\tvalue1_new\n"
			err = os.WriteFile(testFile, []byte(newContent), 0644)
			So(err, ShouldBeNil)
			time.Sleep(100 * time.Millisecond) // 等待替换完成

			// 关闭应该等待所有旧store被清理
			start := time.Now()
			err = loadableStore.Close()
			duration := time.Since(start)

			So(err, ShouldBeNil)
			// 关闭时间应该至少包含延迟时间（考虑到其他处理时间，设置一个较小的下限）
			So(duration, ShouldBeGreaterThan, 50*time.Millisecond)
		})
	})
}