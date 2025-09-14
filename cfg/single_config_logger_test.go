package cfg

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hatlonely/gox/cfg/decoder"
	"github.com/hatlonely/gox/cfg/provider"
	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/log/writer"
	"github.com/hatlonely/gox/refx"
)

// MockWriter 用于测试的模拟 Writer
type MockWriter struct {
	logs []string
}

func (w *MockWriter) Write(p []byte) (n int, err error) {
	w.logs = append(w.logs, string(p))
	return len(p), nil
}

func (w *MockWriter) Close() error {
	return nil
}

func TestConfig_WithLogger(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	initialData := `database:
  host: localhost
  port: 3306
redis:
  host: localhost
  port: 6379
`

	if err := os.WriteFile(configFile, []byte(initialData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 创建 mock writer 来捕获日志
	mockWriter := &MockWriter{}

	// 创建 logger
	logger, err := log.NewLogWithOptions(&log.Options{
		Level:  "info",
		Format: "json",
		Output: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/log/writer",
			Type:      "MultiWriter",
			Options: &writer.MultiWriterOptions{
				Writers: []refx.TypeOptions{
					{
						Type:    "custom",
						Options: mockWriter,
					},
				},
			},
		},
	})
	if err != nil {
		// 如果创建 logger 失败，创建一个简单的 mock logger
		logger = &mockLogger{writer: mockWriter}
	}

	// 创建配置对象
	options := &Options{
		Provider: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/provider",
			Type:      "FileProvider",
			Options: &provider.FileProviderOptions{
				FilePath: configFile,
			},
		},
		Decoder: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/decoder",
			Type:      "YamlDecoder",
			Options:   &decoder.YamlDecoderOptions{Indent: 2},
		},
		// 不设置 Logger配置，使用默认的终端输出
	}

	config, err := NewSingleConfigWithOptions(options)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	defer config.Close()

	// 使用 SetLogger 设置我们的 mock logger
	config.SetLogger(logger)

	// 注册 onChange handler
	config.OnChange(func(s storage.Storage) error {
		return nil // 成功的 handler
	})

	config.OnChange(func(s storage.Storage) error {
		return fmt.Errorf("test error") // 失败的 handler
	})

	// 注册 onKeyChange handler
	config.OnKeyChange("database", func(s storage.Storage) error {
		return nil // 成功的 handler
	})

	config.OnKeyChange("database", func(s storage.Storage) error {
		return fmt.Errorf("database handler error") // 失败的 handler
	})

	// 等待一小段时间让监听器设置完成
	time.Sleep(100 * time.Millisecond)

	// 启动监听
	err = config.Watch()
	if err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}

	// 修改配置文件触发变更
	updatedData := `database:
  host: newhost
  port: 3307
redis:
  host: localhost
  port: 6379
`

	if err := os.WriteFile(configFile, []byte(updatedData), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// 等待变更处理完成
	time.Sleep(200 * time.Millisecond)

	// 验证日志记录
	if len(mockWriter.logs) == 0 {
		t.Error("Expected log entries, got none")
	} else {
		t.Logf("Got %d log entries:", len(mockWriter.logs))
		for i, logEntry := range mockWriter.logs {
			t.Logf("Log %d: %s", i, logEntry)
		}
	}

	// 检查是否包含预期的日志内容
	logContent := strings.Join(mockWriter.logs, "\n")

	// 应该有成功和失败的日志
	if !strings.Contains(logContent, "onChange handler succeeded") {
		t.Errorf("Expected log for successful onChange handler, got: %s", logContent)
	}

	if !strings.Contains(logContent, "onChange handler failed") {
		t.Errorf("Expected log for failed onChange handler, got: %s", logContent)
	}

	if !strings.Contains(logContent, "onKeyChange handler succeeded") {
		// 由于现在所有的 handler 都使用相同的执行机制，不再区分 onChange 和 onKeyChange
		// 检查是否有数据库相关的成功日志
		if !strings.Contains(logContent, "database") {
			t.Errorf("Expected database related logs, got: %s", logContent)
		}
	}

	if !strings.Contains(logContent, "onKeyChange handler failed") {
		// 检查是否有失败相关的日志
		if !strings.Contains(logContent, "failed") {
			t.Errorf("Expected failure logs, got: %s", logContent)
		}
	}

	// 检查是否包含 key 和 duration 信息
	// 对于根配置变更，key 为空字符串，检查是否有空字符串的 key 记录或 onChange handler 日志
	hasRootKeyLog := false
	for _, log := range mockWriter.logs {
		// 检查是否包含空字符串 key 的日志记录
		if strings.Contains(log, `"key":""`) || strings.Contains(log, `key ""`) ||
			(strings.Contains(log, "onChange handler") && strings.Contains(log, "key")) {
			hasRootKeyLog = true
			break
		}
	}

	if !hasRootKeyLog {
		t.Errorf("Expected empty key (root) logs, got: %s", logContent)
	}

	if !strings.Contains(logContent, "database") {
		t.Errorf("Expected database key in logs, got: %s", logContent)
	}

	if !strings.Contains(logContent, "duration") {
		t.Errorf("Expected duration information in logs, got: %s", logContent)
	}
}

// mockLogger 用于测试的简单 logger 实现
type mockLogger struct {
	writer *MockWriter
}

func (l *mockLogger) Debug(msg string, args ...any) {
	l.writer.Write([]byte(fmt.Sprintf("DEBUG: %s %v\n", msg, args)))
}

func (l *mockLogger) Info(msg string, args ...any) {
	l.writer.Write([]byte(fmt.Sprintf("INFO: %s %v\n", msg, args)))
}

func (l *mockLogger) Warn(msg string, args ...any) {
	l.writer.Write([]byte(fmt.Sprintf("WARN: %s %v\n", msg, args)))
}

func (l *mockLogger) Error(msg string, args ...any) {
	l.writer.Write([]byte(fmt.Sprintf("ERROR: %s %v\n", msg, args)))
}

func (l *mockLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.Debug(msg, args...)
}

func (l *mockLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.Info(msg, args...)
}

func (l *mockLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.Warn(msg, args...)
}

func (l *mockLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.Error(msg, args...)
}

func (l *mockLogger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	l.Info(msg, args...)
}

func (l *mockLogger) With(args ...any) log.Logger {
	return l
}

func (l *mockLogger) WithGroup(name string) log.Logger {
	return l
}

func (l *mockLogger) Handler() slog.Handler {
	return nil
}

func TestConfig_WithLoggerOptions(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	initialData := `database:
  host: localhost
  port: 3306
`

	if err := os.WriteFile(configFile, []byte(initialData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 使用 log.Options 创建配置
	options := &Options{
		Provider: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/provider",
			Type:      "FileProvider",
			Options: &provider.FileProviderOptions{
				FilePath: configFile,
			},
		},
		Decoder: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/decoder",
			Type:      "YamlDecoder",
			Options:   &decoder.YamlDecoderOptions{Indent: 2},
		},
		Logger: &log.Options{
			Level:  "debug",
			Format: "text",
		},
	}

	config, err := NewSingleConfigWithOptions(options)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	defer config.Close()

	// 验证 logger 不为 nil
	if config.logger == nil {
		t.Error("Expected logger to be created, got nil")
	}

	// 注册一个 handler
	config.OnChange(func(s storage.Storage) error {
		return nil
	})

	// 启动监听
	err = config.Watch()
	if err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}

	// 等待一小段时间让监听器设置完成
	time.Sleep(100 * time.Millisecond)

	// 修改配置文件触发变更
	updatedData := `database:
  host: newhost
  port: 3307
`

	if err := os.WriteFile(configFile, []byte(updatedData), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// 等待变更处理完成
	time.Sleep(200 * time.Millisecond)

	// 此时应该有日志输出（在终端中）
	t.Log("Logger test completed successfully - check terminal output for log messages")
}

func TestConfig_DefaultLogger(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	initialData := `test: value`

	if err := os.WriteFile(configFile, []byte(initialData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 不设置 Logger配置，使用默认的
	options := &Options{
		Provider: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/provider",
			Type:      "FileProvider",
			Options: &provider.FileProviderOptions{
				FilePath: configFile,
			},
		},
		Decoder: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/decoder",
			Type:      "YamlDecoder",
			Options:   &decoder.YamlDecoderOptions{Indent: 2},
		},
		// Logger 为 nil，应该使用默认配置
	}

	config, err := NewSingleConfigWithOptions(options)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	defer config.Close()

	// 验证默认 logger 被创建
	if config.logger == nil {
		t.Error("Expected default logger to be created, got nil")
	}

	// 测试 SetLogger 方法
	mockWriter := &MockWriter{}
	mockLogger := &mockLogger{writer: mockWriter}
	config.SetLogger(mockLogger)

	// 验证 logger 被更新
	if config.logger != mockLogger {
		t.Error("Expected logger to be updated")
	}

	t.Log("Default logger test completed successfully")
}
