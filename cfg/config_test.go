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

func TestConfig_RealUsage(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	configData := `database:
  host: localhost
  port: 3306
  name: testdb
  timeout: "30s"
servers:
  - name: web1
    port: 8080
  - name: web2  
    port: 8081
redis:
  host: localhost
  port: 6379
`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	t.Run("NewConfigWithOptions", func(t *testing.T) {
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
		}

		config, err := NewConfigWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
		defer config.provider.Close()

		if config == nil {
			t.Fatal("Config should not be nil")
		}

		// 测试 ConvertTo 功能
		var result map[string]any
		err = config.ConvertTo(&result)
		if err != nil {
			t.Fatalf("Failed to convert config: %v", err)
		}

		if result["database"] == nil {
			t.Error("Database section should exist")
		}
	})

	t.Run("Sub and ConvertTo", func(t *testing.T) {
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
		}

		config, err := NewConfigWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
		defer config.provider.Close()

		// 测试 Sub 功能
		dbConfig := config.Sub("database")
		if dbConfig == nil {
			t.Fatal("Database config should not be nil")
		}

		// 测试子配置的 ConvertTo
		type DatabaseConfig struct {
			Host    string        `yaml:"host"`
			Port    int           `yaml:"port"`
			Name    string        `yaml:"name"`
			Timeout time.Duration `yaml:"timeout"`
		}

		var dbSettings DatabaseConfig
		err = dbConfig.ConvertTo(&dbSettings)
		if err != nil {
			t.Fatalf("Failed to convert database config: %v", err)
		}

		if dbSettings.Host != "localhost" {
			t.Errorf("Expected host 'localhost', got '%s'", dbSettings.Host)
		}
		if dbSettings.Port != 3306 {
			t.Errorf("Expected port 3306, got %d", dbSettings.Port)
		}
		if dbSettings.Timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", dbSettings.Timeout)
		}

		// 测试数组访问
		serversConfig := config.Sub("servers")
		var servers []map[string]any
		err = serversConfig.ConvertTo(&servers)
		if err != nil {
			t.Fatalf("Failed to convert servers config: %v", err)
		}

		if len(servers) != 2 {
			t.Errorf("Expected 2 servers, got %d", len(servers))
		}

		// 测试数组索引访问
		server1Config := config.Sub("servers[0]")
		var server1 map[string]any
		err = server1Config.ConvertTo(&server1)
		if err != nil {
			t.Fatalf("Failed to convert server1 config: %v", err)
		}

		if server1["name"] != "web1" {
			t.Errorf("Expected server name 'web1', got '%v'", server1["name"])
		}
	})

	t.Run("OnChange and OnKeyChange", func(t *testing.T) {
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
		}

		config, err := NewConfigWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
		defer config.provider.Close()

		// 测试监听器注册
		rootChangeCalled := false
		config.OnChange(func(c *Config) error {
			rootChangeCalled = true
			return nil
		})

		dbChangeCalled := false
		config.OnKeyChange("database", func(c *Config) error {
			dbChangeCalled = true
			return nil
		})

		// 测试子配置的 OnChange（应该重定向到 OnKeyChange）
		redisChangeCalled := false
		redisConfig := config.Sub("redis")
		redisConfig.OnChange(func(c *Config) error {
			redisChangeCalled = true
			return nil
		})

		// 验证监听器已注册（根配置变更监听器现在使用空字符串key）
		if len(config.onKeyChangeHandlers[""]) != 1 {
			t.Errorf("Expected 1 root change handler, got %d", len(config.onKeyChangeHandlers[""]))
		}

		if len(config.onKeyChangeHandlers["database"]) != 1 {
			t.Errorf("Expected 1 database change handler, got %d", len(config.onKeyChangeHandlers["database"]))
		}

		if len(config.onKeyChangeHandlers["redis"]) != 1 {
			t.Errorf("Expected 1 redis change handler, got %d", len(config.onKeyChangeHandlers["redis"]))
		}

		// 修改配置文件触发变更
		newConfigData := `database:
  host: newhost
  port: 3307
  name: testdb
  timeout: "60s"
servers:
  - name: web1
    port: 8080
  - name: web2  
    port: 8081
redis:
  host: newredis
  port: 6380
`
		err = os.WriteFile(configFile, []byte(newConfigData), 0644)
		if err != nil {
			t.Fatalf("Failed to update config file: %v", err)
		}

		// 等待文件变更通知
		time.Sleep(200 * time.Millisecond)

		// 验证监听器被调用
		if !rootChangeCalled {
			t.Error("Root change handler should have been called")
		}
		if !dbChangeCalled {
			t.Error("Database change handler should have been called")
		}
		if !redisChangeCalled {
			t.Error("Redis change handler should have been called")
		}

		// 验证配置已更新
		var dbConfig map[string]any
		err = config.Sub("database").ConvertTo(&dbConfig)
		if err != nil {
			t.Fatalf("Failed to get updated database config: %v", err)
		}

		if dbConfig["host"] != "newhost" {
			t.Errorf("Expected updated host 'newhost', got '%v'", dbConfig["host"])
		}
	})

	t.Run("Nested Sub Config", func(t *testing.T) {
		// 创建嵌套配置文件
		nestedConfigFile := filepath.Join(tempDir, "nested.yaml")
		nestedConfigData := `app:
  database:
    primary:
      host: primary-host
      port: 5432
    secondary:
      host: secondary-host
      port: 5433
  cache:
    redis:
      host: redis-host
      port: 6379
`
		if err := os.WriteFile(nestedConfigFile, []byte(nestedConfigData), 0644); err != nil {
			t.Fatalf("Failed to write nested config file: %v", err)
		}

		options := &Options{
			Provider: refx.TypeOptions{
				Namespace: "github.com/hatlonely/gox/cfg/provider",
				Type:      "FileProvider",
				Options: &provider.FileProviderOptions{
					FilePath: nestedConfigFile,
				},
			},
			Decoder: refx.TypeOptions{
				Namespace: "github.com/hatlonely/gox/cfg/decoder",
				Type:      "YamlDecoder",
				Options:   &decoder.YamlDecoderOptions{Indent: 2},
			},
		}

		config, err := NewConfigWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
		defer config.provider.Close()

		// 测试多层嵌套访问
		primaryDB := config.Sub("app").Sub("database").Sub("primary")

		var dbConfig map[string]any
		err = primaryDB.ConvertTo(&dbConfig)
		if err != nil {
			t.Fatalf("Failed to convert primary database config: %v", err)
		}

		if dbConfig["host"] != "primary-host" {
			t.Errorf("Expected host 'primary-host', got '%v'", dbConfig["host"])
		}

		// 测试 getFullKey 功能
		expectedFullKey := "app.database.primary"
		actualFullKey := primaryDB.getFullKey()
		if actualFullKey != expectedFullKey {
			t.Errorf("Expected full key '%s', got '%s'", expectedFullKey, actualFullKey)
		}

		// 测试嵌套配置的监听
		primaryDB.OnChange(func(c *Config) error {
			return nil
		})

		// 验证监听器注册到根配置
		if len(config.onKeyChangeHandlers["app.database.primary"]) != 1 {
			t.Error("Nested config OnChange should register to root config with full key")
		}
	})

	t.Run("NewConfig Simple Usage", func(t *testing.T) {
		// 测试 NewConfig 函数的简单用法
		yamlFile := filepath.Join(tempDir, "simple.yaml")
		yamlContent := `database:
  host: localhost
  port: 3306`

		if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("Failed to write YAML file: %v", err)
		}

		config, err := NewConfig(yamlFile)
		if err != nil {
			t.Fatalf("Failed to create config with NewConfig: %v", err)
		}
		defer config.provider.Close()

		// 测试配置读取
		var result map[string]any
		err = config.ConvertTo(&result)
		if err != nil {
			t.Fatalf("Failed to convert config: %v", err)
		}

		if result["database"] == nil {
			t.Error("Database section should exist")
		}

		// 测试子配置访问
		dbConfig := config.Sub("database")
		var db map[string]any
		err = dbConfig.ConvertTo(&db)
		if err != nil {
			t.Fatalf("Failed to convert database config: %v", err)
		}

		if db["host"] != "localhost" {
			t.Errorf("Expected host 'localhost', got '%v'", db["host"])
		}
	})

	t.Run("NewConfig Error Handling", func(t *testing.T) {
		// 测试错误情况
		tests := []struct {
			name        string
			filename    string
			expectError string
		}{
			{
				name:        "Empty filename",
				filename:    "",
				expectError: "filename cannot be empty",
			},
			{
				name:        "Unsupported extension",
				filename:    "config.xml",
				expectError: "unsupported file extension: .xml",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := NewConfig(tt.filename)
				if err == nil {
					t.Fatal("Expected error, got nil")
				}

				if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.expectError, err.Error())
				}
			})
		}
	})
}

// TestConfig_Close 测试 Close 方法的多次调用行为
func TestConfig_Close(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-close.yaml")
	configData := `test:
  value: "hello"`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	t.Run("Root config multiple close calls", func(t *testing.T) {
		config, err := NewConfig(configFile)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		// 第一次调用 Close
		err1 := config.Close()
		if err1 != nil {
			t.Errorf("First close should succeed, got error: %v", err1)
		}

		// 第二次调用 Close，应该返回同样的结果
		err2 := config.Close()
		if err2 != err1 {
			t.Errorf("Second close should return same result as first, got %v vs %v", err2, err1)
		}

		// 第三次调用 Close，也应该返回同样的结果
		err3 := config.Close()
		if err3 != err1 {
			t.Errorf("Third close should return same result as first, got %v vs %v", err3, err1)
		}
	})

	t.Run("Sub config multiple close calls", func(t *testing.T) {
		config, err := NewConfig(configFile)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		// 获取子配置
		subConfig := config.Sub("test")

		// 子配置的 Close 应该转发到根配置
		err1 := subConfig.Close()
		if err1 != nil {
			t.Errorf("Sub config first close should succeed, got error: %v", err1)
		}

		// 第二次调用子配置的 Close
		err2 := subConfig.Close()
		if err2 != err1 {
			t.Errorf("Sub config second close should return same result, got %v vs %v", err2, err1)
		}

		// 调用根配置的 Close 也应该返回同样的结果
		err3 := config.Close()
		if err3 != err1 {
			t.Errorf("Root config close should return same result, got %v vs %v", err3, err1)
		}
	})

	t.Run("Concurrent close calls", func(t *testing.T) {
		config, err := NewConfig(configFile)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		// 使用 goroutines 并发调用 Close
		results := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func() {
				results <- config.Close()
			}()
		}

		// 收集结果
		var errors []error
		for i := 0; i < 10; i++ {
			errors = append(errors, <-results)
		}

		// 所有结果应该相同
		firstResult := errors[0]
		for i, result := range errors {
			if result != firstResult {
				t.Errorf("Concurrent close result %d differs: got %v, expected %v", i, result, firstResult)
			}
		}
	})
}

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
