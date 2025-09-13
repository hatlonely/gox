package writer

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"
)

func TestNewConsoleWriterWithOptions(t *testing.T) {
	tests := []struct {
		name       string
		options    *ConsoleWriterOptions
		wantErr    bool
		wantTarget string
	}{
		{
			name:       "nil options",
			options:    nil,
			wantErr:    false,
			wantTarget: "stdout",
		},
		{
			name: "default options",
			options: &ConsoleWriterOptions{
				Color:  true,
				Target: "stdout",
			},
			wantErr:    false,
			wantTarget: "stdout",
		},
		{
			name: "stderr target",
			options: &ConsoleWriterOptions{
				Color:  false,
				Target: "stderr",
			},
			wantErr:    false,
			wantTarget: "stderr",
		},
		{
			name: "empty target defaults to stdout",
			options: &ConsoleWriterOptions{
				Color:  true,
				Target: "",
			},
			wantErr:    false,
			wantTarget: "stdout",
		},
		{
			name: "invalid target defaults to stdout",
			options: &ConsoleWriterOptions{
				Color:  true,
				Target: "invalid",
			},
			wantErr:    false,
			wantTarget: "stdout",
		},
		{
			name: "color disabled",
			options: &ConsoleWriterOptions{
				Color:  false,
				Target: "stdout",
			},
			wantErr:    false,
			wantTarget: "stdout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewConsoleWriterWithOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConsoleWriterWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if writer == nil {
					t.Error("NewConsoleWriterWithOptions() returned nil writer without error")
					return
				}

				// 验证 color 设置
				expectedColor := tt.options != nil && tt.options.Color
				if tt.options == nil {
					expectedColor = true // 默认值
				}
				if writer.color != expectedColor {
					t.Errorf("Expected color = %v, got %v", expectedColor, writer.color)
				}

				// 验证目标输出流
				var expectedWriter io.Writer
				switch tt.wantTarget {
				case "stderr":
					expectedWriter = os.Stderr
				case "stdout", "":
					expectedWriter = os.Stdout
				default:
					expectedWriter = os.Stdout
				}

				if writer.writer != expectedWriter {
					t.Errorf("Expected writer target to be %v, got %v", expectedWriter, writer.writer)
				}
			}
		})
	}
}

func TestConsoleWriter_Write(t *testing.T) {
	// 创建一个用于测试的 buffer
	var buf bytes.Buffer

	writer := &ConsoleWriter{
		writer: &buf,
		color:  true,
	}

	tests := []struct {
		name     string
		data     []byte
		wantN    int
		wantErr  bool
		expected string
	}{
		{
			name:     "simple message",
			data:     []byte("Hello, World!"),
			wantN:    13,
			wantErr:  false,
			expected: "Hello, World!",
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
			data:     []byte("测试消息 🚀"),
			wantN:    17,
			wantErr:  false,
			expected: "测试消息 🚀",
		},
		{
			name:     "json log message",
			data:     []byte(`{"level":"info","msg":"test message","timestamp":"2023-01-01T00:00:00Z"}`),
			wantN:    72,
			wantErr:  false,
			expected: `{"level":"info","msg":"test message","timestamp":"2023-01-01T00:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset() // 清空 buffer

			n, err := writer.Write(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConsoleWriter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if n != tt.wantN {
				t.Errorf("ConsoleWriter.Write() wrote %d bytes, want %d", n, tt.wantN)
			}

			if got := buf.String(); got != tt.expected {
				t.Errorf("ConsoleWriter.Write() output = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConsoleWriter_WriteConcurrency(t *testing.T) {
	// 为并发测试创建一个简单的计数器
	var totalWrites int64
	var mu sync.Mutex

	// 创建一个模拟 writer
	counterWriter := &CountingWriter{}
	writer := &ConsoleWriter{
		writer: counterWriter,
		color:  true,
	}

	// 测试并发写入
	var wg sync.WaitGroup
	message := []byte("concurrent test message\n")

	// 启动多个 goroutines 并发写入
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				n, err := writer.Write(message)
				if err != nil {
					t.Errorf("Goroutine %d: Write error: %v", id, err)
					return
				}
				if n != len(message) {
					t.Errorf("Goroutine %d: Write returned %d, expected %d", id, n, len(message))
					return
				}
				mu.Lock()
				totalWrites++
				mu.Unlock()
			}
		}(i)
	}

	// 等待所有 goroutines 完成
	wg.Wait()

	// 验证总写入次数
	mu.Lock()
	expectedWrites := int64(100)
	if totalWrites != expectedWrites {
		t.Errorf("Expected %d writes, got %d", expectedWrites, totalWrites)
	}
	mu.Unlock()

	// 验证总字节数
	expectedBytes := int64(100 * len(message))
	if counterWriter.BytesWritten != expectedBytes {
		t.Errorf("Expected %d bytes written, got %d", expectedBytes, counterWriter.BytesWritten)
	}
}

// CountingWriter 用于测试的计数 writer
type CountingWriter struct {
	BytesWritten int64
	mu           sync.Mutex
}

func (c *CountingWriter) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BytesWritten += int64(len(p))
	return len(p), nil
}

func TestConsoleWriter_Close(t *testing.T) {
	writer, err := NewConsoleWriterWithOptions(&ConsoleWriterOptions{
		Color:  true,
		Target: "stdout",
	})
	if err != nil {
		t.Fatalf("NewConsoleWriterWithOptions() error = %v", err)
	}

	// 测试 Close 方法
	err = writer.Close()
	if err != nil {
		t.Errorf("ConsoleWriter.Close() error = %v", err)
	}

	// 测试多次调用 Close
	err = writer.Close()
	if err != nil {
		t.Errorf("ConsoleWriter.Close() second call error = %v", err)
	}

	// Close 后仍然可以写入（因为控制台不需要真正关闭）
	n, err := writer.Write([]byte("test after close"))
	if err != nil {
		t.Errorf("Write after close error = %v", err)
	}
	if n != 16 {
		t.Errorf("Write after close returned %d, expected 16", n)
	}
}

func TestConsoleWriter_Interface(t *testing.T) {
	writer, err := NewConsoleWriterWithOptions(nil)
	if err != nil {
		t.Fatalf("NewConsoleWriterWithOptions() error = %v", err)
	}

	// 验证实现了 Writer 接口
	var _ Writer = writer

	// 验证实现了 io.Writer 接口
	var _ io.Writer = writer

	// 验证实现了 io.Closer 接口
	var _ io.Closer = writer
}

func TestConsoleWriter_ColorConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		color       bool
		expectColor bool
	}{
		{
			name:        "color enabled",
			color:       true,
			expectColor: true,
		},
		{
			name:        "color disabled",
			color:       false,
			expectColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewConsoleWriterWithOptions(&ConsoleWriterOptions{
				Color:  tt.color,
				Target: "stdout",
			})
			if err != nil {
				t.Fatalf("NewConsoleWriterWithOptions() error = %v", err)
			}

			if writer.color != tt.expectColor {
				t.Errorf("Expected color = %v, got %v", tt.expectColor, writer.color)
			}
		})
	}
}

func TestConsoleWriter_TargetValidation(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		expectedWriter io.Writer
	}{
		{
			name:           "stdout target",
			target:         "stdout",
			expectedWriter: os.Stdout,
		},
		{
			name:           "stderr target",
			target:         "stderr",
			expectedWriter: os.Stderr,
		},
		{
			name:           "empty target defaults to stdout",
			target:         "",
			expectedWriter: os.Stdout,
		},
		{
			name:           "invalid target defaults to stdout",
			target:         "invalid_target",
			expectedWriter: os.Stdout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewConsoleWriterWithOptions(&ConsoleWriterOptions{
				Color:  true,
				Target: tt.target,
			})
			if err != nil {
				t.Fatalf("NewConsoleWriterWithOptions() error = %v", err)
			}

			if writer.writer != tt.expectedWriter {
				t.Errorf("Expected writer target %v, got %v", tt.expectedWriter, writer.writer)
			}
		})
	}
}
