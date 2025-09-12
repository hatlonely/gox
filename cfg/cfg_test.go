package cfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/refx"
)

// mockStorage 模拟的 Storage 实现
type mockStorage struct{}

func (m *mockStorage) Sub(key string) storage.Storage {
	return &mockStorage{}
}

func (m *mockStorage) ConvertTo(object any) error {
	return nil
}

func init() {
	// 注册测试需要的 Provider 和 Decoder
	refx.MustRegister("github.com/hatlonely/gox/cfg/provider", "FileProvider", func() interface{} {
		return &struct{}{}
	})
	refx.MustRegister("github.com/hatlonely/gox/cfg/decoder", "JSONDecoder", func() interface{} {
		return &struct{}{}
	})
}

func TestConfig_BasicUsage(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")
	configData := `{
		"database": {
			"host": "localhost",
			"port": 3306,
			"name": "testdb"
		},
		"servers": [
			{"name": "web1", "port": 8080},
			{"name": "web2", "port": 8081}
		]
	}`
	
	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 测试基本的 Sub 结构
	t.Run("Sub structure", func(t *testing.T) {
		// 创建模拟的 storage
		mockStorage := &mockStorage{}
		
		config := &Config{
			storage: mockStorage,
		}
		
		subConfig := config.Sub("database")
		if subConfig == nil {
			t.Error("Sub should return a valid config object")
		}
		
		if subConfig.parent != config {
			t.Error("Sub config should have correct parent reference")
		}
		
		if subConfig.key != "database" {
			t.Error("Sub config should have correct key")
		}
	})

	t.Run("OnChange registration", func(t *testing.T) {
		rootConfig := &Config{
			onChangeHandlers:    make([]func(*Config) error, 0),
			onKeyChangeHandlers: make(map[string][]func(*Config) error),
		}

		// 测试根配置的 OnChange
		rootConfig.OnChange(func(c *Config) error {
			return nil
		})

		if len(rootConfig.onChangeHandlers) != 1 {
			t.Error("OnChange handler should be registered to root config")
		}

		// 测试子配置的 OnChange（应该重定向到 OnKeyChange）
		subConfig := &Config{
			parent: rootConfig,
			key:    "database",
		}

		subConfig.OnChange(func(c *Config) error {
			return nil
		})

		if len(rootConfig.onKeyChangeHandlers["database"]) != 1 {
			t.Error("Sub config OnChange should be registered as OnKeyChange")
		}
	})

	t.Run("getRoot and getFullKey", func(t *testing.T) {
		rootConfig := &Config{}
		
		level1 := &Config{parent: rootConfig, key: "database"}
		level2 := &Config{parent: level1, key: "connection"}
		level3 := &Config{parent: level2, key: "pool"}

		if level3.getRoot() != rootConfig {
			t.Error("getRoot should return the root config")
		}

		expected := "database.connection.pool"
		if level3.getFullKey() != expected {
			t.Errorf("getFullKey should return %s, got %s", expected, level3.getFullKey())
		}

		if rootConfig.getFullKey() != "" {
			t.Error("Root config getFullKey should return empty string")
		}
	})
}