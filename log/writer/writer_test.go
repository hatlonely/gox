package writer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hatlonely/gox/refx"
)

// TestWriter_InterfaceCompliance 测试所有 writer 实现都符合 Writer 接口
func TestWriter_InterfaceCompliance(t *testing.T) {
	tests := []struct {
		name     string
		createFn func(t *testing.T) Writer
	}{
		{
			name: "ConsoleWriter",
			createFn: func(t *testing.T) Writer {
				w, err := NewConsoleWriterWithOptions(&ConsoleWriterOptions{
					Color:  false,
					Target: "stdout",
				})
				if err != nil {
					t.Fatalf("Failed to create ConsoleWriter: %v", err)
				}
				return w
			},
		},
		{
			name: "FileWriter",
			createFn: func(t *testing.T) Writer {
				tempDir := t.TempDir()
				logFile := filepath.Join(tempDir, "test.log")
				w, err := NewFileWriterWithOptions(&FileWriterOptions{
					Path: logFile,
				})
				if err != nil {
					t.Fatalf("Failed to create FileWriter: %v", err)
				}
				return w
			},
		},
		{
			name: "MultiWriter",
			createFn: func(t *testing.T) Writer {
				tempDir := t.TempDir()
				logFile := filepath.Join(tempDir, "multi_test.log")
				w, err := NewMultiWriterWithOptions(&MultiWriterOptions{
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
					t.Fatalf("Failed to create MultiWriter: %v", err)
				}
				return w
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := tt.createFn(t)
			defer writer.Close()

			// 验证实现了 Writer 接口
			var _ Writer = writer

			// 验证实现了 io.Writer 接口
			var _ io.Writer = writer

			// 验证实现了 io.Closer 接口
			var _ io.Closer = writer

			// 测试基本的写入功能
			testData := []byte("interface compliance test")
			n, err := writer.Write(testData)
			if err != nil {
				t.Errorf("%s Write() error = %v", tt.name, err)
			}
			if n != len(testData) {
				t.Errorf("%s Write() returned %d, expected %d", tt.name, n, len(testData))
			}

			// 测试 Close 功能
			err = writer.Close()
			if err != nil {
				t.Errorf("%s Close() error = %v", tt.name, err)
			}
		})
	}
}

// TestWriter_RefxRegistration 测试 refx 注册功能
func TestWriter_RefxRegistration(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		typeName  string
		options   interface{}
		wantErr   bool
	}{
		{
			name:      "ConsoleWriter registration",
			namespace: "github.com/hatlonely/gox/log/writer",
			typeName:  "ConsoleWriter",
			options: &ConsoleWriterOptions{
				Color:  true,
				Target: "stdout",
			},
			wantErr: false,
		},
		{
			name:      "FileWriter registration",
			namespace: "github.com/hatlonely/gox/log/writer",
			typeName:  "FileWriter",
			options: &FileWriterOptions{
				Path: filepath.Join(t.TempDir(), "refx_test.log"),
			},
			wantErr: false,
		},
		{
			name:      "MultiWriter registration",
			namespace: "github.com/hatlonely/gox/log/writer",
			typeName:  "MultiWriter",
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
			name:      "Invalid type",
			namespace: "github.com/hatlonely/gox/log/writer",
			typeName:  "InvalidWriter",
			options:   nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := refx.New(tt.namespace, tt.typeName, tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("refx.New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if obj == nil {
					t.Error("refx.New() returned nil without error")
					return
				}

				// 验证返回的对象实现了 Writer 接口
				writer, ok := obj.(Writer)
				if !ok {
					t.Errorf("refx.New() returned object that doesn't implement Writer interface")
					return
				}

				// 清理资源
				writer.Close()
			}
		})
	}
}

// TestWriter_RealWorldScenario 测试真实世界的日志记录场景
func TestWriter_RealWorldScenario(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "app.log")

	// 创建一个同时输出到控制台和文件的多输出器
	writer, err := NewMultiWriterWithOptions(&MultiWriterOptions{
		Writers: []refx.TypeOptions{
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "ConsoleWriter",
				Options: &ConsoleWriterOptions{
					Color:  true,
					Target: "stderr", // 错误日志输出到 stderr
				},
			},
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "FileWriter",
				Options: &FileWriterOptions{
					Path:       logFile,
					MaxSize:    10, // 10MB
					MaxBackups: 5,
					MaxAge:     30,
					Compress:   true,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create multi writer: %v", err)
	}
	defer writer.Close()

	// 模拟各种类型的日志消息
	logMessages := []string{
		`{"level":"info","msg":"Application started","timestamp":"2023-01-01T00:00:00Z"}` + "\n",
		`{"level":"debug","msg":"Processing request","request_id":"req-123","timestamp":"2023-01-01T00:00:01Z"}` + "\n",
		`{"level":"warn","msg":"Slow query detected","duration":"2.5s","query":"SELECT * FROM users","timestamp":"2023-01-01T00:00:02Z"}` + "\n",
		`{"level":"error","msg":"Database connection failed","error":"connection timeout","timestamp":"2023-01-01T00:00:03Z"}` + "\n",
		`{"level":"info","msg":"Request completed","request_id":"req-123","status":200,"timestamp":"2023-01-01T00:00:04Z"}` + "\n",
	}

	// 写入所有日志消息
	totalBytes := 0
	for _, msg := range logMessages {
		n, err := writer.Write([]byte(msg))
		if err != nil {
			t.Errorf("Failed to write log message: %v", err)
		}
		totalBytes += n
	}

	// 验证文件内容
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	expectedContent := strings.Join(logMessages, "")
	if string(content) != expectedContent {
		t.Errorf("File content mismatch:\nGot: %q\nWant: %q", string(content), expectedContent)
	}

	// 验证文件大小
	stat, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	if int(stat.Size()) != totalBytes {
		t.Errorf("File size mismatch: got %d, want %d", stat.Size(), totalBytes)
	}

	// 验证每种日志级别的消息都存在
	contentStr := string(content)
	levels := []string{"info", "debug", "warn", "error"}
	for _, level := range levels {
		if !strings.Contains(contentStr, fmt.Sprintf(`"level":"%s"`, level)) {
			t.Errorf("Log file doesn't contain %s level messages", level)
		}
	}
}

// TestWriter_ErrorHandling 测试错误处理场景
func TestWriter_ErrorHandling(t *testing.T) {
	t.Run("FileWriter with invalid path", func(t *testing.T) {
		_, err := NewFileWriterWithOptions(&FileWriterOptions{
			Path: "", // 空路径应该失败
		})
		if err == nil {
			t.Error("Expected error for empty file path, got nil")
		}
		if !strings.Contains(err.Error(), "file path is required") {
			t.Errorf("Expected 'file path is required' error, got: %v", err)
		}
	})

	t.Run("MultiWriter with no writers", func(t *testing.T) {
		_, err := NewMultiWriterWithOptions(&MultiWriterOptions{
			Writers: []refx.TypeOptions{}, // 空的 writers 列表
		})
		if err == nil {
			t.Error("Expected error for empty writers list, got nil")
		}
		if !strings.Contains(err.Error(), "at least one writer is required") {
			t.Errorf("Expected 'at least one writer is required' error, got: %v", err)
		}
	})

	t.Run("Write to closed FileWriter", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "closed_test.log")

		writer, err := NewFileWriterWithOptions(&FileWriterOptions{
			Path: logFile,
		})
		if err != nil {
			t.Fatalf("Failed to create FileWriter: %v", err)
		}

		// 关闭 writer
		err = writer.Close()
		if err != nil {
			t.Fatalf("Failed to close writer: %v", err)
		}

		// 尝试写入已关闭的 writer
		_, err = writer.Write([]byte("test"))
		if err == nil {
			t.Error("Expected error when writing to closed writer, got nil")
		}
		if !strings.Contains(err.Error(), "file is closed") {
			t.Errorf("Expected 'file is closed' error, got: %v", err)
		}
	})
}

// TestWriter_Performance 测试性能相关场景
func TestWriter_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "perf_test.log")

	writer, err := NewFileWriterWithOptions(&FileWriterOptions{
		Path: logFile,
	})
	if err != nil {
		t.Fatalf("Failed to create FileWriter: %v", err)
	}
	defer writer.Close()

	// 写入大量小消息
	message := []byte("Performance test message with moderate length to simulate real log entries\n")
	numMessages := 10000

	for i := 0; i < numMessages; i++ {
		n, err := writer.Write(message)
		if err != nil {
			t.Errorf("Write error at message %d: %v", i, err)
			break
		}
		if n != len(message) {
			t.Errorf("Short write at message %d: got %d, want %d", i, n, len(message))
			break
		}
	}

	// 验证最终文件大小
	stat, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	expectedSize := int64(numMessages * len(message))
	if stat.Size() != expectedSize {
		t.Errorf("File size mismatch: got %d, want %d", stat.Size(), expectedSize)
	}
}

// TestWriter_BufferSizes 测试不同缓冲区大小的写入
func TestWriter_BufferSizes(t *testing.T) {
	var buf bytes.Buffer
	writer := &ConsoleWriter{
		writer: &buf,
		color:  false,
	}

	// 测试不同大小的写入
	sizes := []int{1, 10, 100, 1024, 4096, 65536}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			buf.Reset()

			data := make([]byte, size)
			for i := range data {
				data[i] = byte('A' + (i % 26)) // 填充字母
			}

			n, err := writer.Write(data)
			if err != nil {
				t.Errorf("Write error for size %d: %v", size, err)
			}
			if n != size {
				t.Errorf("Write size %d: got %d bytes, want %d", size, n, size)
			}
			if buf.Len() != size {
				t.Errorf("Buffer size %d: got %d bytes, want %d", size, buf.Len(), size)
			}
		})
	}
}

// TestWriter_SpecialCharacters 测试特殊字符处理
func TestWriter_SpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	writer := &ConsoleWriter{
		writer: &buf,
		color:  false,
	}

	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "null bytes",
			data: []byte("test\x00null\x00bytes"),
			want: "test\x00null\x00bytes",
		},
		{
			name: "control characters",
			data: []byte("line1\rcarriage\return\trab\nline2"),
			want: "line1\rcarriage\return\trab\nline2",
		},
		{
			name: "unicode",
			data: []byte("Hello 世界 🌍 🚀"),
			want: "Hello 世界 🌍 🚀",
		},
		{
			name: "mixed encoding",
			data: append([]byte("ASCII "), []byte("UTF-8: 测试")...),
			want: "ASCII UTF-8: 测试",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			n, err := writer.Write(tt.data)
			if err != nil {
				t.Errorf("Write error: %v", err)
			}
			if n != len(tt.data) {
				t.Errorf("Write returned %d, expected %d", n, len(tt.data))
			}
			if buf.String() != tt.want {
				t.Errorf("Output mismatch: got %q, want %q", buf.String(), tt.want)
			}
		})
	}
}
