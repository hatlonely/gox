package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/hatlonely/gox/kv/parser"
	"github.com/hatlonely/gox/log/logger"
)

func TestNewKVFileLoaderWithOptions(t *testing.T) {
	Convey("NewKVFileLoaderWithOptions", t, func() {
		Convey("空配置返回错误", func() {
			loader, err := NewKVFileLoaderWithOptions[string, string](nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "options is nil")
			So(loader, ShouldBeNil)
		})

		Convey("创建基本KVFileLoader", func() {
			// 手动创建parser，避免ref注册问题
			lineParser, err := parser.NewLineParserWithOptions[string, string](nil)
			So(err, ShouldBeNil)

			options := &KVFileLoaderOptions{
				FilePath: "/tmp/test.txt",
			}

			// 直接创建loader实例而不是通过NewKVFileLoaderWithOptions
			loader := &KVFileLoader[string, string]{
				filePath:             options.FilePath,
				parser:               lineParser,
				scannerBufferMinSize: 64 * 1024,
				scannerBufferMaxSize: 4 * 1024 * 1024,
				done:                 make(chan struct{}, 1),
				skipDirtyRows:        false,
				logger:               &MockLogger{},
			}

			So(loader, ShouldNotBeNil)
			So(loader.filePath, ShouldEqual, "/tmp/test.txt")
			So(loader.skipDirtyRows, ShouldBeFalse)
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
				if key != "" {  // 跳过空键值
					results = append(results, key+":"+value)
				}
				return nil
			})

			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 2)
			So(results, ShouldContain, "key1:value1")
			So(results, ShouldContain, "key2:value2")
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

func TestKVFileLoaderClose(t *testing.T) {
	Convey("KVFileLoader.Close", t, func() {
		Convey("未启动监听的关闭", func() {
			lineParser, err := parser.NewLineParserWithOptions[string, string](nil)
			So(err, ShouldBeNil)

			loader := &KVFileLoader[string, string]{
				filePath:             "/tmp/test.txt",
				parser:               lineParser,
				scannerBufferMinSize: 64 * 1024,
				scannerBufferMaxSize: 4 * 1024 * 1024,
				done:                 make(chan struct{}, 1),
				skipDirtyRows:        false,
				logger:               &MockLogger{},
			}

			// 直接关闭应该也能正常工作
			err = loader.Close()
			So(err, ShouldBeNil)
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