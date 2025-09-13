package cfg

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hatlonely/gox/cfg/decoder"
	"github.com/hatlonely/gox/cfg/provider"
	"github.com/hatlonely/gox/refx"
)

// TestConfig_AdvancedHandlerExecution 测试高级的 handler 执行功能
func TestConfig_AdvancedHandlerExecution(t *testing.T) {
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

	// 创建 mock writer 来捕获日志
	mockWriter := &MockWriter{}
	mockLogger := &mockLogger{writer: mockWriter}

	t.Run("AsyncExecution", func(t *testing.T) {
		// 创建配置，启用异步执行
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
			HandlerExecution: &HandlerExecutionOptions{
				Timeout:     2 * time.Second,
				Async:       true,
				ErrorPolicy: "continue",
			},
		}

		config, err := NewConfigWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
		defer config.Close()

		// 设置 mock logger
		config.SetLogger(mockLogger)

		// 清空之前的日志
		mockWriter.logs = []string{}

		// 注册多个 handler，包括快速和慢速的
		var mu sync.Mutex
		executionOrder := []string{}

		config.OnChange(func(c *Config) error {
			mu.Lock()
			executionOrder = append(executionOrder, "fast")
			mu.Unlock()
			return nil
		})

		config.OnChange(func(c *Config) error {
			time.Sleep(100 * time.Millisecond) // 慢速 handler
			mu.Lock()
			executionOrder = append(executionOrder, "slow")
			mu.Unlock()
			return nil
		})

		config.OnChange(func(c *Config) error {
			mu.Lock()
			executionOrder = append(executionOrder, "fast2")
			mu.Unlock()
			return nil
		})

		// 等待监听器设置完成
		time.Sleep(100 * time.Millisecond)

		// 触发配置变更
		updatedData := `database:
  host: newhost
  port: 3307
`
		if err := os.WriteFile(configFile, []byte(updatedData), 0644); err != nil {
			t.Fatalf("Failed to update config file: %v", err)
		}

		// 等待所有 handler 执行完成
		time.Sleep(300 * time.Millisecond)

		// 验证异步执行：由于是并行执行，快速的 handler 可能先完成
		mu.Lock()
		if len(executionOrder) != 3 {
			t.Errorf("Expected 3 handlers to execute, got %d", len(executionOrder))
		}
		mu.Unlock()

		// 验证日志中包含所有 handler 的执行记录
		logContent := strings.Join(mockWriter.logs, "\n")
		if !strings.Contains(logContent, "onChange handler succeeded") {
			t.Error("Expected successful handler logs")
		}

		// 验证包含 index 信息
		if !strings.Contains(logContent, "index") {
			t.Error("Expected index information in logs")
		}
	})

	t.Run("TimeoutControl", func(t *testing.T) {
		// 创建配置，设置较短的超时时间
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
			HandlerExecution: &HandlerExecutionOptions{
				Timeout:     50 * time.Millisecond, // 很短的超时时间
				Async:       false,                 // 同步执行便于测试
				ErrorPolicy: "continue",
			},
		}

		config, err := NewConfigWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
		defer config.Close()

		// 设置 mock logger
		config.SetLogger(mockLogger)

		// 清空之前的日志
		mockWriter.logs = []string{}

		// 注册一个会超时的 handler
		config.OnChange(func(c *Config) error {
			time.Sleep(200 * time.Millisecond) // 超过超时时间
			return nil
		})

		// 等待监听器设置完成
		time.Sleep(100 * time.Millisecond)

		// 触发配置变更
		updatedData := `database:
  host: timeouttest
  port: 3308
`
		if err := os.WriteFile(configFile, []byte(updatedData), 0644); err != nil {
			t.Fatalf("Failed to update config file: %v", err)
		}

		// 等待超时处理完成
		time.Sleep(400 * time.Millisecond)

		// 验证超时日志
		logContent := strings.Join(mockWriter.logs, "\n")
		if !strings.Contains(logContent, "onChange handler timeout") {
			t.Errorf("Expected timeout log, got: %s", logContent)
		}
		if !strings.Contains(logContent, "handler execution timeout") {
			t.Error("Expected timeout error message in logs")
		}
	})

	t.Run("SyncExecution", func(t *testing.T) {
		// 创建配置，使用同步执行
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
			HandlerExecution: &HandlerExecutionOptions{
				Timeout:     1 * time.Second,
				Async:       false, // 同步执行
				ErrorPolicy: "continue",
			},
		}

		config, err := NewConfigWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
		defer config.Close()

		// 设置 mock logger
		config.SetLogger(mockLogger)

		// 清空之前的日志
		mockWriter.logs = []string{}

		// 注册多个 handler，测试顺序执行
		var mu sync.Mutex
		executionOrder := []string{}

		config.OnChange(func(c *Config) error {
			mu.Lock()
			executionOrder = append(executionOrder, "first")
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			return nil
		})

		config.OnChange(func(c *Config) error {
			mu.Lock()
			executionOrder = append(executionOrder, "second")
			mu.Unlock()
			return nil
		})

		// 等待监听器设置完成
		time.Sleep(100 * time.Millisecond)

		// 触发配置变更
		updatedData := `database:
  host: synctest
  port: 3309
`
		if err := os.WriteFile(configFile, []byte(updatedData), 0644); err != nil {
			t.Fatalf("Failed to update config file: %v", err)
		}

		// 等待同步执行完成
		time.Sleep(300 * time.Millisecond)

		// 验证同步执行：handler 应该按顺序执行
		mu.Lock()
		if len(executionOrder) != 2 {
			t.Errorf("Expected 2 handlers to execute, got %d", len(executionOrder))
		}
		if len(executionOrder) >= 2 && executionOrder[0] != "first" {
			t.Error("Expected first handler to execute first")
		}
		if len(executionOrder) >= 2 && executionOrder[1] != "second" {
			t.Error("Expected second handler to execute second")
		}
		mu.Unlock()
	})
}

func TestConfig_DefaultHandlerExecution(t *testing.T) {
	// 测试不提供 HandlerExecution 配置时的默认行为
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	initialData := `test: value`

	if err := os.WriteFile(configFile, []byte(initialData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

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
		// 不设置 HandlerExecution，使用默认值
	}

	config, err := NewConfigWithOptions(options)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	defer config.Close()

	// 验证默认配置
	if config.handlerExecution == nil {
		t.Error("Expected default handler execution config to be created")
	}
	if config.handlerExecution.Timeout != 5*time.Second {
		t.Errorf("Expected default timeout 5s, got %v", config.handlerExecution.Timeout)
	}
	if !config.handlerExecution.Async {
		t.Error("Expected default async to be true")
	}
	if config.handlerExecution.ErrorPolicy != "continue" {
		t.Errorf("Expected default error policy 'continue', got %s", config.handlerExecution.ErrorPolicy)
	}

	t.Log("Default handler execution config test completed successfully")
}
