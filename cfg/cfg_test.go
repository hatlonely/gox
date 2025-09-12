package cfg

import (
	"os"
	"path/filepath"
	"strings"
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

		// 验证监听器已注册
		if len(config.onChangeHandlers) != 1 {
			t.Errorf("Expected 1 root change handler, got %d", len(config.onChangeHandlers))
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