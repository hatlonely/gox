package provider

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGormProvider_SQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	provider, err := NewGormProviderWithOptions(&GormProviderOptions{
		ConfigID: "test_config",
		Driver:   "sqlite",
		DSN:      dbFile,
	})
	if err != nil {
		t.Fatalf("Failed to create GormProvider: %v", err)
	}
	defer provider.Close()

	// 测试保存配置
	testData := []byte(`{"database": {"host": "localhost", "port": 3306}}`)
	err = provider.Save(testData)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// 测试读取配置
	data, err := provider.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(data))
	}
}

func TestGormProvider_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	provider, err := NewGormProviderWithOptions(&GormProviderOptions{
		ConfigID: "nonexistent_config",
		Driver:   "sqlite",
		DSN:      dbFile,
	})
	if err != nil {
		t.Fatalf("Failed to create GormProvider: %v", err)
	}
	defer provider.Close()

	_, err = provider.Load()
	if err == nil {
		t.Error("Expected error when loading nonexistent config")
	}
}

func TestGormProvider_OnChange(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	provider, err := NewGormProviderWithOptions(&GormProviderOptions{
		ConfigID:     "test_config",
		Driver:       "sqlite",
		DSN:          dbFile,
		PollInterval: 100 * time.Millisecond, // 快速轮询用于测试
	})
	if err != nil {
		t.Fatalf("Failed to create GormProvider: %v", err)
	}
	defer provider.Close()

	// 保存初始配置
	initialData := []byte(`{"key": "value1"}`)
	err = provider.Save(initialData)
	if err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// 读取一次以设置 lastVersion
	_, err = provider.Load()
	if err != nil {
		t.Fatalf("Failed to load initial config: %v", err)
	}

	// 设置变更监听
	changeChan := make(chan []byte, 1)
	provider.OnChange(func(data []byte) error {
		changeChan <- data
		return nil
	})

	// 等待一小段时间让轮询启动
	time.Sleep(200 * time.Millisecond)

	// 更新配置
	updatedData := []byte(`{"key": "value2"}`)
	err = provider.Save(updatedData)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// 等待变更通知
	select {
	case data := <-changeChan:
		if string(data) != string(updatedData) {
			t.Errorf("Expected %s, got %s", string(updatedData), string(data))
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for config change notification")
	}
}

func TestGormProvider_CustomTableName(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")
	customTableName := "custom_config"

	provider, err := NewGormProviderWithOptions(&GormProviderOptions{
		ConfigID:  "test_config",
		Driver:    "sqlite",
		DSN:       dbFile,
		TableName: customTableName,
	})
	if err != nil {
		t.Fatalf("Failed to create GormProvider: %v", err)
	}
	defer provider.Close()

	testData := []byte(`{"custom": "table"}`)
	err = provider.Save(testData)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	data, err := provider.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(data))
	}
}

func TestGormProvider_InvalidOptions(t *testing.T) {
	// 测试 nil options
	_, err := NewGormProviderWithOptions(nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	// 测试空 ConfigID
	_, err = NewGormProviderWithOptions(&GormProviderOptions{
		Driver: "sqlite",
		DSN:    ":memory:",
	})
	if err == nil {
		t.Error("Expected error for empty ConfigID")
	}

	// 测试空 Driver
	_, err = NewGormProviderWithOptions(&GormProviderOptions{
		ConfigID: "test",
		DSN:      ":memory:",
	})
	if err == nil {
		t.Error("Expected error for empty Driver")
	}

	// 测试空 DSN
	_, err = NewGormProviderWithOptions(&GormProviderOptions{
		ConfigID: "test",
		Driver:   "sqlite",
	})
	if err == nil {
		t.Error("Expected error for empty DSN")
	}

	// 测试不支持的驱动
	_, err = NewGormProviderWithOptions(&GormProviderOptions{
		ConfigID: "test",
		Driver:   "unsupported",
		DSN:      ":memory:",
	})
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}
}

// 如果有 MySQL 测试环境，可以启用这个测试
func TestGormProvider_MySQL_Skip(t *testing.T) {
	mysqlDSN := os.Getenv("MYSQL_TEST_DSN")
	if mysqlDSN == "" {
		t.Skip("MYSQL_TEST_DSN not set, skipping MySQL test")
	}

	provider, err := NewGormProviderWithOptions(&GormProviderOptions{
		ConfigID: "mysql_test_config",
		Driver:   "mysql",
		DSN:      mysqlDSN,
	})
	if err != nil {
		t.Fatalf("Failed to create MySQL GormProvider: %v", err)
	}
	defer provider.Close()

	testData := []byte(`{"mysql": "test"}`)
	err = provider.Save(testData)
	if err != nil {
		t.Fatalf("Failed to save config to MySQL: %v", err)
	}

	data, err := provider.Load()
	if err != nil {
		t.Fatalf("Failed to load config from MySQL: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(data))
	}
}
