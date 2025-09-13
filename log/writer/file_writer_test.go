package writer

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewFileWriterWithOptions(t *testing.T) {
	tests := []struct {
		name    string
		options *FileWriterOptions
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil options",
			options: nil,
			wantErr: true,
			errMsg:  "file path is required",
		},
		{
			name: "empty path",
			options: &FileWriterOptions{
				Path: "",
			},
			wantErr: true,
			errMsg:  "file path is required",
		},
		{
			name: "valid file path",
			options: &FileWriterOptions{
				Path: filepath.Join(t.TempDir(), "test.log"),
			},
			wantErr: false,
		},
		{
			name: "with all options",
			options: &FileWriterOptions{
				Path:       filepath.Join(t.TempDir(), "test_full.log"),
				MaxSize:    10,
				MaxBackups: 3,
				MaxAge:     7,
				Compress:   true,
			},
			wantErr: false,
		},
		{
			name: "nested directory",
			options: &FileWriterOptions{
				Path: filepath.Join(t.TempDir(), "logs", "app", "test.log"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewFileWriterWithOptions(tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileWriterWithOptions() error = %v, wantErr %v", err, tt.wantErr)
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
				t.Error("NewFileWriterWithOptions() returned nil writer without error")
				return
			}

			// 清理资源
			writer.Close()

			// 验证文件是否被创建
			if tt.options != nil && tt.options.Path != "" {
				if _, err := os.Stat(tt.options.Path); os.IsNotExist(err) {
					t.Errorf("Expected file %s to be created", tt.options.Path)
				}
			}
		})
	}
}

func TestFileWriter_Write(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_write.log")

	writer, err := NewFileWriterWithOptions(&FileWriterOptions{
		Path: logFile,
	})
	if err != nil {
		t.Fatalf("NewFileWriterWithOptions() error = %v", err)
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
			data:     []byte(`{"level":"info","msg":"test message","timestamp":"2023-01-01T00:00:00Z"}` + "\n"),
			wantN:    73,
			wantErr:  false,
			expected: `{"level":"info","msg":"test message","timestamp":"2023-01-01T00:00:00Z"}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := writer.Write(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("FileWriter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if n != tt.wantN {
				t.Errorf("FileWriter.Write() wrote %d bytes, want %d", n, tt.wantN)
			}
		})
	}

	// 验证文件内容
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	expectedContent := "Hello, World!Line 1\nLine 2\n测试消息 🚀" + `{"level":"info","msg":"test message","timestamp":"2023-01-01T00:00:00Z"}` + "\n"
	if string(content) != expectedContent {
		t.Errorf("File content mismatch:\nGot: %q\nWant: %q", string(content), expectedContent)
	}
}

func TestFileWriter_WriteConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "concurrent_test.log")

	writer, err := NewFileWriterWithOptions(&FileWriterOptions{
		Path: logFile,
	})
	if err != nil {
		t.Fatalf("NewFileWriterWithOptions() error = %v", err)
	}
	defer writer.Close()

	// 测试并发写入
	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 10
	message := []byte("concurrent test message\n")

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

	// 验证文件大小（100 条消息，每条 24 字节）
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
	messageCount := strings.Count(string(content), "concurrent test message")
	expectedCount := numGoroutines * messagesPerGoroutine
	if messageCount != expectedCount {
		t.Errorf("Expected %d messages, got %d", expectedCount, messageCount)
	}
}

func TestFileWriter_Close(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "close_test.log")

	writer, err := NewFileWriterWithOptions(&FileWriterOptions{
		Path: logFile,
	})
	if err != nil {
		t.Fatalf("NewFileWriterWithOptions() error = %v", err)
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
		t.Errorf("FileWriter.Close() error = %v", err)
	}

	// 测试多次调用 Close
	err = writer.Close()
	if err != nil {
		t.Errorf("FileWriter.Close() second call error = %v", err)
	}

	// Close 后写入应该失败
	_, err = writer.Write([]byte("after close"))
	if err == nil {
		t.Error("Expected error when writing after close, got nil")
	}
	if !strings.Contains(err.Error(), "file is closed") {
		t.Errorf("Expected 'file is closed' error, got: %v", err)
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

func TestFileWriter_Interface(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "interface_test.log")

	writer, err := NewFileWriterWithOptions(&FileWriterOptions{
		Path: logFile,
	})
	if err != nil {
		t.Fatalf("NewFileWriterWithOptions() error = %v", err)
	}
	defer writer.Close()

	// 验证实现了 Writer 接口
	var _ Writer = writer

	// 验证实现了 io.Writer 接口
	var _ = writer.Write

	// 验证实现了 io.Closer 接口
	var _ = writer.Close
}

func TestFileWriter_DirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "single nested directory",
			path:    filepath.Join(tempDir, "logs", "test.log"),
			wantErr: false,
		},
		{
			name:    "multiple nested directories",
			path:    filepath.Join(tempDir, "app", "logs", "2023", "01", "test.log"),
			wantErr: false,
		},
		{
			name:    "existing directory",
			path:    filepath.Join(tempDir, "test_existing.log"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewFileWriterWithOptions(&FileWriterOptions{
				Path: tt.path,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileWriterWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if writer == nil {
					t.Error("Expected writer to be created")
					return
				}

				defer writer.Close()

				// 验证目录被创建
				dir := filepath.Dir(tt.path)
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					t.Errorf("Expected directory %s to be created", dir)
				}

				// 验证可以写入
				testData := []byte("directory creation test")
				n, err := writer.Write(testData)
				if err != nil {
					t.Errorf("Write error: %v", err)
				}
				if n != len(testData) {
					t.Errorf("Write returned %d, expected %d", n, len(testData))
				}

				// 验证文件存在
				if _, err := os.Stat(tt.path); os.IsNotExist(err) {
					t.Errorf("Expected file %s to exist", tt.path)
				}
			}
		})
	}
}

func TestFileWriter_AppendMode(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "append_test.log")

	// 先写入一些初始内容
	initialContent := []byte("initial content\n")
	err := os.WriteFile(logFile, initialContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// 创建 FileWriter，应该以追加模式打开
	writer, err := NewFileWriterWithOptions(&FileWriterOptions{
		Path: logFile,
	})
	if err != nil {
		t.Fatalf("NewFileWriterWithOptions() error = %v", err)
	}
	defer writer.Close()

	// 写入新内容
	newContent := []byte("appended content\n")
	n, err := writer.Write(newContent)
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != len(newContent) {
		t.Errorf("Write returned %d, expected %d", n, len(newContent))
	}

	// 验证文件内容包括初始内容和新内容
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	expectedContent := "initial content\nappended content\n"
	if string(content) != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, string(content))
	}
}

func TestFileWriter_InvalidPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "invalid characters in filename",
			path:    "/tmp/test\x00.log",
			wantErr: true,
		},
		{
			name:    "read-only directory",
			path:    "/root/test.log", // 假设这是只读目录
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewFileWriterWithOptions(&FileWriterOptions{
				Path: tt.path,
			})

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
					if writer != nil {
						writer.Close()
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if writer != nil {
					writer.Close()
				}
			}
		})
	}
}
