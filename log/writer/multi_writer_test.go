package writer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hatlonely/gox/refx"
)

func TestNewMultiWriterWithOptions(t *testing.T) {
	tests := []struct {
		name    string
		options *MultiWriterOptions
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil options",
			options: nil,
			wantErr: true,
			errMsg:  "at least one writer is required",
		},
		{
			name: "empty writers",
			options: &MultiWriterOptions{
				Writers: []refx.TypeOptions{},
			},
			wantErr: true,
			errMsg:  "at least one writer is required",
		},
		{
			name: "single console writer",
			options: &MultiWriterOptions{
				Writers: []refx.TypeOptions{
					{
						Namespace: "github.com/hatlonely/gox/log/writer",
						Type:      "ConsoleWriter",
						Options: &ConsoleWriterOptions{
							Color:  false,
							Target: "stdout",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple writers",
			options: &MultiWriterOptions{
				Writers: []refx.TypeOptions{
					{
						Namespace: "github.com/hatlonely/gox/log/writer",
						Type:      "ConsoleWriter",
						Options: &ConsoleWriterOptions{
							Color:  false,
							Target: "stdout",
						},
					},
					{
						Namespace: "github.com/hatlonely/gox/log/writer",
						Type:      "FileWriter",
						Options: &FileWriterOptions{
							Path: filepath.Join(t.TempDir(), "multi_test.log"),
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewMultiWriterWithOptions(tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewMultiWriterWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if writer == nil {
				t.Error("NewMultiWriterWithOptions() returned nil writer without error")
				return
			}

			// 验证 writers 数量
			if tt.options != nil {
				expectedCount := len(tt.options.Writers)
				if len(writer.writers) != expectedCount {
					t.Errorf("Expected %d writers, got %d", expectedCount, len(writer.writers))
				}
			}

			// 清理资源
			writer.Close()
		})
	}
}

func TestMultiWriter_Write(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "multi_write_test.log")

	// 创建多个输出器：控制台和文件
	writer, err := NewMultiWriterWithOptions(&MultiWriterOptions{
		Writers: []refx.TypeOptions{
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "ConsoleWriter",
				Options: &ConsoleWriterOptions{
					Color:  false,
					Target: "stdout",
				},
			},
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "FileWriter",
				Options: &FileWriterOptions{
					Path: logFile,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewMultiWriterWithOptions() error = %v", err)
	}
	defer writer.Close()

	tests := []struct {
		name     string
		data     []byte
		wantN    int
		wantErr  bool
		expected string
	}{
		{
			name:     "simple message",
			data:     []byte("Hello, Multi Writer!"),
			wantN:    20,
			wantErr:  false,
			expected: "Hello, Multi Writer!",
		},
		{
			name:     "empty message",
			data:     []byte(""),
			wantN:    0,
			wantErr:  false,
			expected: "",
		},
		{
			name:     "multiline message",
			data:     []byte("Line 1\nLine 2\n"),
			wantN:    14,
			wantErr:  false,
			expected: "Line 1\nLine 2\n",
		},
		{
			name:     "unicode message",
			data:     []byte("多输出器测试 🚀"),
			wantN:    23,
			wantErr:  false,
			expected: "多输出器测试 🚀",
		},
		{
			name:     "json log message",
			data:     []byte(`{"level":"info","msg":"multi writer test","timestamp":"2023-01-01T00:00:00Z"}` + "\n"),
			wantN:    78,
			wantErr:  false,
			expected: `{"level":"info","msg":"multi writer test","timestamp":"2023-01-01T00:00:00Z"}` + "\n",
		},
	}

	var allContent strings.Builder
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := writer.Write(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("MultiWriter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if n != tt.wantN {
				t.Errorf("MultiWriter.Write() wrote %d bytes, want %d", n, tt.wantN)
			}

			allContent.Write(tt.data)
		})
	}

	// 验证文件内容（应该包含所有写入的内容）
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	expectedContent := allContent.String()
	if string(content) != expectedContent {
		t.Errorf("File content mismatch:\nGot: %q\nWant: %q", string(content), expectedContent)
	}
}

func TestMultiWriter_WriteConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "multi_concurrent_test.log")

	// 创建多输出器
	writer, err := NewMultiWriterWithOptions(&MultiWriterOptions{
		Writers: []refx.TypeOptions{
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "ConsoleWriter",
				Options: &ConsoleWriterOptions{
					Color:  false,
					Target: "stderr", // 使用 stderr 避免与测试输出混合
				},
			},
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "FileWriter",
				Options: &FileWriterOptions{
					Path: logFile,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewMultiWriterWithOptions() error = %v", err)
	}
	defer writer.Close()

	// 测试并发写入
	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 10
	message := []byte("concurrent multi writer test\n")

	// 启动多个 goroutines 并发写入
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				n, err := writer.Write(message)
				if err != nil {
					t.Errorf("Goroutine %d: Write error: %v", id, err)
					return
				}
				if n != len(message) {
					t.Errorf("Goroutine %d: Write returned %d, expected %d", id, n, len(message))
					return
				}
			}
		}(i)
	}

	// 等待所有 goroutines 完成
	wg.Wait()

	// 验证文件大小
	stat, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	expectedSize := int64(numGoroutines * messagesPerGoroutine * len(message))
	if stat.Size() != expectedSize {
		t.Errorf("Expected file size %d, got %d", expectedSize, stat.Size())
	}

	// 验证内容正确性
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// 计算实际的消息数量
	messageCount := strings.Count(string(content), "concurrent multi writer test")
	expectedCount := numGoroutines * messagesPerGoroutine
	if messageCount != expectedCount {
		t.Errorf("Expected %d messages, got %d", expectedCount, messageCount)
	}
}

func TestMultiWriter_Close(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "multi_close_test.log")

	writer, err := NewMultiWriterWithOptions(&MultiWriterOptions{
		Writers: []refx.TypeOptions{
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "ConsoleWriter",
				Options: &ConsoleWriterOptions{
					Color:  false,
					Target: "stdout",
				},
			},
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "FileWriter",
				Options: &FileWriterOptions{
					Path: logFile,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewMultiWriterWithOptions() error = %v", err)
	}

	// 写入一些数据
	testData := []byte("before close")
	n, err := writer.Write(testData)
	if err != nil {
		t.Errorf("Write before close error: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write before close returned %d, expected %d", n, len(testData))
	}

	// 测试 Close 方法
	err = writer.Close()
	if err != nil {
		t.Errorf("MultiWriter.Close() error = %v", err)
	}

	// 测试多次调用 Close
	err = writer.Close()
	if err != nil {
		// 第二次 Close 可能会返回错误，因为文件已关闭
		t.Logf("Second close returned: %v", err)
	}

	// 验证文件内容
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if string(content) != "before close" {
		t.Errorf("Expected file content 'before close', got '%s'", string(content))
	}
}

func TestMultiWriter_WriteFailure(t *testing.T) {
	// 创建一个会失败的 writer
	failingWriter := &FailingWriter{shouldFail: true}

	// 创建 multiWriter 使用 NewMultiWriterFromWriters
	multiWriter := NewMultiWriterFromWriters(failingWriter)

	// 尝试写入应该失败
	_, err := multiWriter.Write([]byte("test"))
	if err == nil {
		t.Error("Expected write to fail, got nil error")
	}

	// 测试 Close
	err = multiWriter.Close()
	if err == nil {
		t.Error("Expected close to fail, got nil error")
	}
}

func TestMultiWriter_Interface(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "interface_test.log")

	writer, err := NewMultiWriterWithOptions(&MultiWriterOptions{
		Writers: []refx.TypeOptions{
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "FileWriter",
				Options: &FileWriterOptions{
					Path: logFile,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewMultiWriterWithOptions() error = %v", err)
	}
	defer writer.Close()

	// 验证实现了 Writer 接口
	var _ Writer = writer

	// 验证实现了 io.Writer 接口
	var _ io.Writer = writer

	// 验证实现了 io.Closer 接口
	var _ io.Closer = writer
}

func TestNewMultiWriterFromWriters(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	writer1 := &ConsoleWriter{writer: &buf1, color: false}
	writer2 := &ConsoleWriter{writer: &buf2, color: false}

	multiWriter := NewMultiWriterFromWriters(writer1, writer2)
	if multiWriter == nil {
		t.Fatal("NewMultiWriterFromWriters() returned nil")
	}

	// 测试写入
	testData := []byte("test message")
	n, err := multiWriter.Write(testData)
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write returned %d, expected %d", n, len(testData))
	}

	// 验证两个 buffer 都收到了数据
	if buf1.String() != "test message" {
		t.Errorf("Buffer 1 content: got %q, want 'test message'", buf1.String())
	}
	if buf2.String() != "test message" {
		t.Errorf("Buffer 2 content: got %q, want 'test message'", buf2.String())
	}

	// 测试 Close
	err = multiWriter.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestMultiWriter_PartialWriteFailure(t *testing.T) {
	var buf bytes.Buffer

	// 创建一个正常的 writer 和一个会失败的 writer
	normalWriter := &ConsoleWriter{writer: &buf, color: false}
	failingWriter := &FailingWriter{shouldFail: true}

	multiWriter := NewMultiWriterFromWriters(normalWriter, failingWriter)

	// 写入应该在第二个 writer 失败
	_, err := multiWriter.Write([]byte("test"))
	if err == nil {
		t.Error("Expected write to fail due to failing writer")
	}

	// 第一个 writer 应该已经成功写入
	if buf.String() != "test" {
		t.Errorf("Normal writer should have received data, got %q", buf.String())
	}
}

func TestMultiWriter_EmptyWriters(t *testing.T) {
	// 测试空的 writers 切片
	multiWriter := NewMultiWriterFromWriters()

	// 写入应该成功但什么都不做
	n, err := multiWriter.Write([]byte("test"))
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != 4 {
		t.Errorf("Write returned %d, expected 4", n)
	}

	// Close 应该成功
	err = multiWriter.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}
}

// FailingWriter 是用于测试的失败 writer
type FailingWriter struct {
	shouldFail bool
}

func (w *FailingWriter) Write(p []byte) (n int, err error) {
	if w.shouldFail {
		return 0, fmt.Errorf("write failed")
	}
	return len(p), nil
}

func (w *FailingWriter) Close() error {
	if w.shouldFail {
		return fmt.Errorf("close failed")
	}
	return nil
}

func TestMultiWriter_InvalidWriterType(t *testing.T) {
	// 测试无效的 writer 类型
	options := &MultiWriterOptions{
		Writers: []refx.TypeOptions{
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "InvalidWriter", // 不存在的类型
				Options:   nil,
			},
		},
	}

	writer, err := NewMultiWriterWithOptions(options)
	if err == nil {
		t.Error("Expected error for invalid writer type, got nil")
		if writer != nil {
			writer.Close()
		}
	}
	if !strings.Contains(err.Error(), "failed to create writer") {
		t.Errorf("Expected 'failed to create writer' in error message, got: %v", err)
	}
}

func TestMultiWriter_ShortWrite(t *testing.T) {
	// 创建一个会返回短写入的 writer
	shortWriter := &ShortWriter{}
	multiWriter := NewMultiWriterFromWriters(shortWriter)

	// 写入应该返回 ErrShortWrite
	_, err := multiWriter.Write([]byte("test message"))
	if err != io.ErrShortWrite {
		t.Errorf("Expected ErrShortWrite, got: %v", err)
	}
}

// ShortWriter 是用于测试短写入的 writer
type ShortWriter struct{}

func (w *ShortWriter) Write(p []byte) (n int, err error) {
	// 总是返回比实际长度少的字节数
	return len(p) - 1, nil
}

func (w *ShortWriter) Close() error {
	return nil
}
