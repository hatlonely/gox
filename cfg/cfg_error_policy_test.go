package cfg

import (
	"fmt"
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

func TestConfig_ErrorPolicyStop(t *testing.T) {
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

	// 创建配置，使用 stop 错误策略
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
			Async:       false,  // 同步执行以便测试顺序
			ErrorPolicy: "stop", // 遇到错误就停止
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

	// 记录执行顺序
	var mu sync.Mutex
	executionOrder := []string{}

	// 注册多个 handler：第二个会失败，第三个不应该被执行
	config.OnChange(func(c *Config) error {
		mu.Lock()
		executionOrder = append(executionOrder, "handler1_success")
		mu.Unlock()
		return nil // 成功
	})

	config.OnChange(func(c *Config) error {
		mu.Lock()
		executionOrder = append(executionOrder, "handler2_fail")
		mu.Unlock()
		return fmt.Errorf("intentional failure") // 失败
	})

	config.OnChange(func(c *Config) error {
		mu.Lock()
		executionOrder = append(executionOrder, "handler3_should_not_execute")
		mu.Unlock()
		return nil // 这个不应该被执行
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

	// 等待执行完成
	time.Sleep(300 * time.Millisecond)

	// 验证执行顺序
	mu.Lock()
	if len(executionOrder) != 2 {
		t.Errorf("Expected 2 handlers to execute (stopped at failure), got %d: %v", len(executionOrder), executionOrder)
	}
	if len(executionOrder) >= 1 && executionOrder[0] != "handler1_success" {
		t.Error("First handler should have executed successfully")
	}
	if len(executionOrder) >= 2 && executionOrder[1] != "handler2_fail" {
		t.Error("Second handler should have failed")
	}

	// 确保第三个 handler 没有被执行
	for _, order := range executionOrder {
		if order == "handler3_should_not_execute" {
			t.Error("Third handler should not have been executed due to stop policy")
		}
	}
	mu.Unlock()

	// 验证日志内容
	logContent := strings.Join(mockWriter.logs, "\n")

	// 应该包含成功日志
	if !strings.Contains(logContent, "onChange handler succeeded") {
		t.Error("Expected success log for first handler")
	}

	// 应该包含失败日志（ERROR 级别）
	if !strings.Contains(logContent, "ERROR: onChange handler failed") {
		t.Error("Expected ERROR level failure log for second handler")
	}

	// 应该包含停止执行的日志
	if !strings.Contains(logContent, "handler execution stopped due to error policy") {
		t.Error("Expected log about execution being stopped due to error policy")
	}

	// 验证日志中包含正确的 remainingHandlers 数量
	if !strings.Contains(logContent, "remainingHandlers 1") {
		t.Error("Expected log to show 1 remaining handler was skipped")
	}

	t.Logf("Execution order: %v", executionOrder)
	t.Logf("Log content: %s", logContent)
}

func TestConfig_ErrorPolicyContinue(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	initialData := `test: value1`

	if err := os.WriteFile(configFile, []byte(initialData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 创建 mock writer 来捕获日志
	mockWriter := &MockWriter{}
	mockLogger := &mockLogger{writer: mockWriter}

	// 创建配置，使用 continue 错误策略
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
			Async:       false,      // 同步执行以便测试顺序
			ErrorPolicy: "continue", // 遇到错误继续执行
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

	// 记录执行顺序
	var mu sync.Mutex
	executionOrder := []string{}

	// 注册多个 handler：第二个会失败，但第三个应该继续执行
	config.OnChange(func(c *Config) error {
		mu.Lock()
		executionOrder = append(executionOrder, "handler1_success")
		mu.Unlock()
		return nil // 成功
	})

	config.OnChange(func(c *Config) error {
		mu.Lock()
		executionOrder = append(executionOrder, "handler2_fail")
		mu.Unlock()
		return fmt.Errorf("intentional failure") // 失败
	})

	config.OnChange(func(c *Config) error {
		mu.Lock()
		executionOrder = append(executionOrder, "handler3_success")
		mu.Unlock()
		return nil // 应该被执行
	})

	// 等待监听器设置完成
	time.Sleep(100 * time.Millisecond)

	// 触发配置变更
	updatedData := `test: value2`
	if err := os.WriteFile(configFile, []byte(updatedData), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// 等待执行完成
	time.Sleep(300 * time.Millisecond)

	// 验证执行顺序
	mu.Lock()
	if len(executionOrder) != 3 {
		t.Errorf("Expected 3 handlers to execute (continue policy), got %d: %v", len(executionOrder), executionOrder)
	}
	if len(executionOrder) >= 1 && executionOrder[0] != "handler1_success" {
		t.Error("First handler should have executed successfully")
	}
	if len(executionOrder) >= 2 && executionOrder[1] != "handler2_fail" {
		t.Error("Second handler should have failed")
	}
	if len(executionOrder) >= 3 && executionOrder[2] != "handler3_success" {
		t.Error("Third handler should have executed successfully despite previous failure")
	}
	mu.Unlock()

	// 验证日志内容
	logContent := strings.Join(mockWriter.logs, "\n")

	// 应该包含两个成功日志
	successCount := strings.Count(logContent, "onChange handler succeeded")
	if successCount != 2 {
		t.Errorf("Expected 2 success logs, got %d", successCount)
	}

	// 应该包含一个失败日志
	if !strings.Contains(logContent, "ERROR: onChange handler failed") {
		t.Error("Expected ERROR level failure log for second handler")
	}

	// 不应该包含停止执行的日志（因为是 continue 策略）
	if strings.Contains(logContent, "handler execution stopped") {
		t.Error("Should not have stop message with continue policy")
	}

	t.Logf("Execution order: %v", executionOrder)
}
