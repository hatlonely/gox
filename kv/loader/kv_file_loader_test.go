package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hatlonely/gox/kv/parser"
	"github.com/hatlonely/gox/log/logger"
	"github.com/hatlonely/gox/ref"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewKVFileLoaderWithOptions(t *testing.T) {
	Convey("NewKVFileLoaderWithOptions", t, func() {
		Convey("创建基本KVFileLoader", func() {
			options := &KVFileLoaderOptions{
				FilePath: "/tmp/test.txt",
				Parser: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/parser",
					Type:      "LineParser[string,string]",
					Options: &parser.LineParserOptions{
						Separator: "\t",
					},
				},
			}

			loader, err := NewKVFileLoaderWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(loader, ShouldNotBeNil)
			So(loader.filePath, ShouldEqual, "/tmp/test.txt")
			So(loader.skipDirtyRows, ShouldBeFalse)
			So(loader.scannerBufferMinSize, ShouldEqual, 4*1024*1024)
			So(loader.scannerBufferMaxSize, ShouldEqual, 4*1024*1024)
			So(loader.logger, ShouldNotBeNil)
		})

		Convey("空配置返回错误", func() {
			loader, err := NewKVFileLoaderWithOptions[string, string](nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "options is nil")
			So(loader, ShouldBeNil)
		})

		Convey("自定义缓冲区大小", func() {
			options := &KVFileLoaderOptions{
				FilePath: "/tmp/test.txt",
				Parser: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/parser",
					Type:      "LineParser[string,string]",
					Options: &parser.LineParserOptions{
						Separator: "\t",
					},
				},
				ScannerBufferMinSize: 1024,
				ScannerBufferMaxSize: 2048,
			}

			loader, err := NewKVFileLoaderWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(loader.scannerBufferMinSize, ShouldEqual, 2048)
			So(loader.scannerBufferMaxSize, ShouldEqual, 2048)
		})

		Convey("启用跳过脏数据", func() {
			options := &KVFileLoaderOptions{
				FilePath: "/tmp/test.txt",
				Parser: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/parser",
					Type:      "LineParser[string,string]",
					Options: &parser.LineParserOptions{
						Separator: "\t",
					},
				},
				SkipDirtyRows: true,
			}

			loader, err := NewKVFileLoaderWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(loader.skipDirtyRows, ShouldBeTrue)
		})

		Convey("配置自定义Logger", func() {
			options := &KVFileLoaderOptions{
				FilePath: "/tmp/test.txt",
				Parser: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/parser",
					Type:      "LineParser[string,string]",
					Options: &parser.LineParserOptions{
						Separator: "\t",
					},
				},
				Logger: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/log",
					Type:      "GetLogger",
					Options:   "test",
				},
			}

			loader, err := NewKVFileLoaderWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(loader.logger, ShouldNotBeNil)
		})

		Convey("Parser创建失败", func() {
			options := &KVFileLoaderOptions{
				FilePath: "/tmp/test.txt",
				Parser: &ref.TypeOptions{
					Type: "InvalidParser",
				},
			}

			loader, err := NewKVFileLoaderWithOptions[string, string](options)
			So(err, ShouldNotBeNil)
			So(loader, ShouldBeNil)
		})
	})
}

func TestKVFileLoaderOnChange(t *testing.T) {
	Convey("KVFileLoader.OnChange", t, func() {
		// 创建临时测试文件
		tmpDir := os.TempDir()
		testFile := filepath.Join(tmpDir, "test_kv_file.txt")

		// 清理函数
		Reset(func() {
			os.RemoveAll(testFile)
		})

		Convey("监听文件变化并触发回调", func() {
			// 创建初始文件
			content := "key1\tvalue1\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			options := &KVFileLoaderOptions{
				FilePath: testFile,
				Parser: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/parser",
					Type:      "LineParser[string,string]",
					Options: &parser.LineParserOptions{
						Separator: "\t",
					},
				},
			}

			loader, err := NewKVFileLoaderWithOptions[string, string](options)
			So(err, ShouldBeNil)

			callCount := 0
			var receivedData [][]string

			listener := func(stream KVStream[string, string]) error {
				callCount++
				var data []string

				err := stream.Each(func(changeType parser.ChangeType, key, value string) error {
					data = append(data, key+":"+value)
					return nil
				})
				receivedData = append(receivedData, data)
				return err
			}

			err = loader.OnChange(listener)
			So(err, ShouldBeNil)

			// 验证初始加载
			So(callCount, ShouldEqual, 1)
			So(len(receivedData), ShouldEqual, 1)
			So(receivedData[0], ShouldContain, "key1:value1")

			// 修改文件触发变化
			newContent := "key1\tvalue1\nkey2\tvalue2\n"
			err = os.WriteFile(testFile, []byte(newContent), 0644)
			So(err, ShouldBeNil)

			// 等待文件变化被检测到
			time.Sleep(100 * time.Millisecond)

			// 清理
			loader.Close()
		})

		Convey("文件不存在时返回错误", func() {
			options := &KVFileLoaderOptions{
				FilePath: "/nonexistent/file.txt",
				Parser: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/parser",
					Type:      "LineParser[string,string]",
					Options: &parser.LineParserOptions{
						Separator: "\t",
					},
				},
			}

			loader, err := NewKVFileLoaderWithOptions[string, string](options)
			So(err, ShouldBeNil)

			listener := func(stream KVStream[string, string]) error {
				return stream.Each(func(parser.ChangeType, string, string) error {
					return nil
				})
			}

			err = loader.OnChange(listener)
			So(err, ShouldNotBeNil)
		})

		Convey("监听器返回错误", func() {
			content := "key1\tvalue1\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			options := &KVFileLoaderOptions{
				FilePath: testFile,
				Parser: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/parser",
					Type:      "LineParser[string,string]",
					Options: &parser.LineParserOptions{
						Separator: "\t",
					},
				},
			}

			loader, err := NewKVFileLoaderWithOptions[string, string](options)
			So(err, ShouldBeNil)

			listener := func(stream KVStream[string, string]) error {
				return stream.Each(func(parser.ChangeType, string, string) error {
					return os.ErrInvalid
				})
			}

			err = loader.OnChange(listener)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "listener failed")
		})
	})
}

func TestKVFileLoaderClose(t *testing.T) {
	Convey("KVFileLoader.Close", t, func() {
		tmpDir := os.TempDir()
		testFile := filepath.Join(tmpDir, "test_close.txt")

		Reset(func() {
			os.RemoveAll(testFile)
		})

		Convey("正常关闭", func() {
			content := "key1\tvalue1\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			options := &KVFileLoaderOptions{
				FilePath: testFile,
				Parser: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/parser",
					Type:      "LineParser[string,string]",
					Options: &parser.LineParserOptions{
						Separator: "\t",
					},
				},
			}

			loader, err := NewKVFileLoaderWithOptions[string, string](options)
			So(err, ShouldBeNil)

			// 启动监听
			listener := func(stream KVStream[string, string]) error {
				return stream.Each(func(parser.ChangeType, string, string) error {
					return nil
				})
			}

			err = loader.OnChange(listener)
			So(err, ShouldBeNil)

			// 关闭应该成功
			err = loader.Close()
			So(err, ShouldBeNil)
		})

		Convey("未启动监听的关闭", func() {
			options := &KVFileLoaderOptions{
				FilePath: testFile,
				Parser: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/kv/parser",
					Type:      "LineParser[string,string]",
					Options: &parser.LineParserOptions{
						Separator: "\t",
					},
				},
			}

			loader, err := NewKVFileLoaderWithOptions[string, string](options)
			So(err, ShouldBeNil)

			// 直接关闭应该也能正常工作
			err = loader.Close()
			So(err, ShouldBeNil)
		})
	})
}

func TestKVFileStreamEach(t *testing.T) {
	Convey("KVFileStream.Each", t, func() {
		tmpDir := os.TempDir()
		testFile := filepath.Join(tmpDir, "test_stream.txt")

		Reset(func() {
			os.RemoveAll(testFile)
		})

		Convey("解析正常数据", func() {
			content := "key1\tvalue1\nkey2\tvalue2\nkey3\tvalue3\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			lineParser, err := parser.NewLineParserWithOptions[string, string](nil)
			So(err, ShouldBeNil)

			stream := &KVFileStream[string, string]{
				filePath:             testFile,
				kvFileLineParser:     lineParser,
				scannerBufferMinSize: 1024,
				scannerBufferMaxSize: 4096,
				logger:               &MockLogger{},
			}

			var results []string
			err = stream.Each(func(changeType parser.ChangeType, key, value string) error {
				results = append(results, key+":"+value)
				return nil
			})

			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 3)
			So(results, ShouldContain, "key1:value1")
			So(results, ShouldContain, "key2:value2")
			So(results, ShouldContain, "key3:value3")
		})

		Convey("跳过脏数据", func() {
			content := "key1\tvalue1\ninvalid_line\nkey2\tvalue2\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			// 创建mock logger
			mockLogger := &MockLogger{}

			lineParser, err := parser.NewLineParserWithOptions[string, string](nil)
			So(err, ShouldBeNil)

			stream := &KVFileStream[string, string]{
				filePath:             testFile,
				kvFileLineParser:     lineParser,
				skipDirtyRows:        true,
				scannerBufferMinSize: 1024,
				scannerBufferMaxSize: 4096,
				logger:               mockLogger,
			}

			var results []string
			err = stream.Each(func(changeType parser.ChangeType, key, value string) error {
				if key != "" { // 跳过空键值
					results = append(results, key+":"+value)
				}
				return nil
			})

			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 2)
			So(results, ShouldContain, "key1:value1")
			So(results, ShouldContain, "key2:value2")
		})

		Convey("不跳过脏数据时返回错误", func() {
			content := "key1\tvalue1\ninvalid_line\nkey2\tvalue2\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			lineParser, err := parser.NewLineParserWithOptions[string, string](nil)
			So(err, ShouldBeNil)

			stream := &KVFileStream[string, string]{
				filePath:             testFile,
				kvFileLineParser:     lineParser,
				skipDirtyRows:        false,
				scannerBufferMinSize: 1024,
				scannerBufferMaxSize: 4096,
				logger:               &MockLogger{},
			}

			var results []string
			err = stream.Each(func(changeType parser.ChangeType, key, value string) error {
				if key == "" { // 模拟handler对空key返回错误
					return os.ErrInvalid
				}
				results = append(results, key+":"+value)
				return nil
			})

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "parse failed")
		})

		Convey("处理器返回错误", func() {
			content := "key1\tvalue1\nkey2\tvalue2\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			lineParser, err := parser.NewLineParserWithOptions[string, string](nil)
			So(err, ShouldBeNil)

			stream := &KVFileStream[string, string]{
				filePath:             testFile,
				kvFileLineParser:     lineParser,
				scannerBufferMinSize: 1024,
				scannerBufferMaxSize: 4096,
				logger:               &MockLogger{},
			}

			err = stream.Each(func(changeType parser.ChangeType, key, value string) error {
				if key == "key2" {
					return os.ErrInvalid
				}
				return nil
			})

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "parse failed")
		})

		Convey("文件不存在", func() {
			lineParser, err := parser.NewLineParserWithOptions[string, string](nil)
			So(err, ShouldBeNil)

			stream := &KVFileStream[string, string]{
				filePath:             "/nonexistent/file.txt",
				kvFileLineParser:     lineParser,
				scannerBufferMinSize: 1024,
				scannerBufferMaxSize: 4096,
				logger:               &MockLogger{},
			}

			err = stream.Each(func(parser.ChangeType, string, string) error {
				return nil
			})

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "os.Open failed")
		})
	})
}

// MockLogger 用于测试的mock logger
type MockLogger struct {
	DebugMessages []string
	InfoMessages  []string
	WarnMessages  []string
	ErrorMessages []string
}

func (m *MockLogger) Debug(msg string, args ...any) {
	m.DebugMessages = append(m.DebugMessages, msg)
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.InfoMessages = append(m.InfoMessages, msg)
}

func (m *MockLogger) Warn(msg string, args ...any) {
	m.WarnMessages = append(m.WarnMessages, msg)
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.ErrorMessages = append(m.ErrorMessages, msg)
}

func (m *MockLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	m.Debug(msg, args...)
}

func (m *MockLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	m.Info(msg, args...)
}

func (m *MockLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	m.Warn(msg, args...)
}

func (m *MockLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	m.Error(msg, args...)
}

func (m *MockLogger) With(args ...any) logger.Logger {
	return m
}

func (m *MockLogger) WithGroup(name string) logger.Logger {
	return m
}
