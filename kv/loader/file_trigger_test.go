package loader

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hatlonely/gox/kv/parser"
	"github.com/hatlonely/gox/ref"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewFileTriggerWithOptions(t *testing.T) {
	Convey("NewFileTriggerWithOptions", t, func() {
		Convey("创建基本FileTrigger", func() {
			options := &FileTriggerOptions{
				FilePath: "/tmp/test.txt",
			}

			trigger, err := NewFileTriggerWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(trigger, ShouldNotBeNil)
			So(trigger.filePath, ShouldEqual, "/tmp/test.txt")
			So(trigger.logger, ShouldNotBeNil)
		})

		Convey("空配置返回错误", func() {
			trigger, err := NewFileTriggerWithOptions[string, string](nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "options is nil")
			So(trigger, ShouldBeNil)
		})

		Convey("配置自定义Logger", func() {
			options := &FileTriggerOptions{
				FilePath: "/tmp/test.txt",
				Logger: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/log",
					Type:      "GetLogger",
					Options:   "test",
				},
			}

			trigger, err := NewFileTriggerWithOptions[string, string](options)
			So(err, ShouldBeNil)
			So(trigger.logger, ShouldNotBeNil)
		})
	})
}

func TestFileTriggerOnChange(t *testing.T) {
	Convey("FileTrigger.OnChange", t, func() {
		// 创建临时测试文件
		tmpDir := os.TempDir()
		testFile := filepath.Join(tmpDir, "test_file_trigger.txt")

		// 清理函数
		Reset(func() {
			os.RemoveAll(testFile)
		})

		Convey("监听文件变化并触发通知", func() {
			// 创建初始文件
			content := "initial content\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			options := &FileTriggerOptions{
				FilePath: testFile,
			}

			trigger, err := NewFileTriggerWithOptions[string, string](options)
			So(err, ShouldBeNil)

			callCount := 0
			var streamCallCount int

			listener := func(stream KVStream[string, string]) error {
				callCount++
				
				// 验证收到的是空流
				err := stream.Each(func(changeType parser.ChangeType, key, value string) error {
					streamCallCount++
					return nil
				})
				return err
			}

			err = trigger.OnChange(listener)
			So(err, ShouldBeNil)

			// 验证初始触发
			So(callCount, ShouldEqual, 1)
			So(streamCallCount, ShouldEqual, 0) // 空流，不应该调用handler

			// 修改文件触发变化
			newContent := "updated content\n"
			err = os.WriteFile(testFile, []byte(newContent), 0644)
			So(err, ShouldBeNil)

			// 等待文件变化被检测到
			time.Sleep(100 * time.Millisecond)

			// 验证触发了第二次通知
			So(callCount, ShouldBeGreaterThanOrEqualTo, 2)
			So(streamCallCount, ShouldEqual, 0) // 仍然是空流

			// 清理
			trigger.Close()
		})

		Convey("监听器返回错误", func() {
			content := "test content\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			options := &FileTriggerOptions{
				FilePath: testFile,
			}

			trigger, err := NewFileTriggerWithOptions[string, string](options)
			So(err, ShouldBeNil)

			listener := func(stream KVStream[string, string]) error {
				return os.ErrInvalid
			}

			err = trigger.OnChange(listener)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "listener failed")
		})

		Convey("文件不存在时仍能正常启动监听", func() {
			options := &FileTriggerOptions{
				FilePath: "/tmp/nonexistent_file.txt",
			}

			trigger, err := NewFileTriggerWithOptions[string, string](options)
			So(err, ShouldBeNil)

			callCount := 0
			listener := func(stream KVStream[string, string]) error {
				callCount++
				return nil
			}

			// 应该能正常启动监听（即使文件不存在）
			err = trigger.OnChange(listener)
			So(err, ShouldBeNil)

			// 验证初始触发
			So(callCount, ShouldEqual, 1)

			// 清理
			trigger.Close()
		})
	})
}

func TestFileTriggerClose(t *testing.T) {
	Convey("FileTrigger.Close", t, func() {
		tmpDir := os.TempDir()
		testFile := filepath.Join(tmpDir, "test_close_trigger.txt")

		Reset(func() {
			os.RemoveAll(testFile)
		})

		Convey("正常关闭", func() {
			content := "test content\n"
			err := os.WriteFile(testFile, []byte(content), 0644)
			So(err, ShouldBeNil)

			options := &FileTriggerOptions{
				FilePath: testFile,
			}

			trigger, err := NewFileTriggerWithOptions[string, string](options)
			So(err, ShouldBeNil)

			// 启动监听
			listener := func(stream KVStream[string, string]) error {
				return nil
			}

			err = trigger.OnChange(listener)
			So(err, ShouldBeNil)

			// 关闭应该成功
			err = trigger.Close()
			So(err, ShouldBeNil)
		})

		Convey("未启动监听的关闭", func() {
			options := &FileTriggerOptions{
				FilePath: testFile,
			}

			trigger, err := NewFileTriggerWithOptions[string, string](options)
			So(err, ShouldBeNil)

			// 直接关闭应该也能正常工作
			err = trigger.Close()
			So(err, ShouldBeNil)
		})
	})
}

func TestEmptyKVStream(t *testing.T) {
	Convey("EmptyKVStream.Each", t, func() {
		Convey("空流不调用处理器", func() {
			stream := &EmptyKVStream[string, string]{
				logger: &MockLogger{},
			}

			handlerCallCount := 0
			err := stream.Each(func(changeType parser.ChangeType, key, value string) error {
				handlerCallCount++
				return nil
			})

			So(err, ShouldBeNil)
			So(handlerCallCount, ShouldEqual, 0)
		})

		Convey("即使处理器会返回错误，空流也不会调用", func() {
			stream := &EmptyKVStream[string, string]{
				logger: &MockLogger{},
			}

			err := stream.Each(func(changeType parser.ChangeType, key, value string) error {
				return os.ErrInvalid // 这个不会被调用
			})

			So(err, ShouldBeNil)
		})
	})
}