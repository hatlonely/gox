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

	// 启动监听
	err = provider.Watch()
	if err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}

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

func TestGormProvider_MultipleOnChange(t *testing.T) {
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

	// 注册多个回调函数
	changeChan1 := make(chan []byte, 1)
	changeChan2 := make(chan []byte, 1)
	changeChan3 := make(chan []byte, 1)

	provider.OnChange(func(data []byte) error {
		changeChan1 <- data
		return nil
	})

	provider.OnChange(func(data []byte) error {
		changeChan2 <- data
		return nil
	})

	provider.OnChange(func(data []byte) error {
		changeChan3 <- data
		return nil
	})

	// 启动监听
	err = provider.Watch()
	if err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}

	// 等待一小段时间让轮询启动
	time.Sleep(200 * time.Millisecond)

	// 更新配置
	updatedData := []byte(`{"key": "value2"}`)
	err = provider.Save(updatedData)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// 验证所有回调都被调用
	select {
	case data := <-changeChan1:
		if string(data) != string(updatedData) {
			t.Errorf("Callback 1: Expected %s, got %s", string(updatedData), string(data))
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for callback 1")
	}

	select {
	case data := <-changeChan2:
		if string(data) != string(updatedData) {
			t.Errorf("Callback 2: Expected %s, got %s", string(updatedData), string(data))
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for callback 2")
	}

	select {
	case data := <-changeChan3:
		if string(data) != string(updatedData) {
			t.Errorf("Callback 3: Expected %s, got %s", string(updatedData), string(data))
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for callback 3")
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

func TestGormProvider_Watch(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	provider, err := NewGormProviderWithOptions(&GormProviderOptions{
		ConfigID:     "test_config",
		Driver:       "sqlite",
		DSN:          dbFile,
		PollInterval: 50 * time.Millisecond, // 快速轮询用于测试
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

	// 测试在没有调用 Watch 的情况下，OnChange 不会触发
	callbackTriggered := false
	provider.OnChange(func(data []byte) error {
		callbackTriggered = true
		return nil
	})

	// 更新配置但不应该触发回调
	updatedData := []byte(`{"key": "value2"}`)
	err = provider.Save(updatedData)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// 等待一下确保没有回调被触发
	time.Sleep(200 * time.Millisecond)
	if callbackTriggered {
		t.Error("Callback should not be triggered before Watch() is called")
	}

	// 现在调用 Watch，应该开始监听
	err = provider.Watch()
	if err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}

	// 再次更新配置，这次应该触发回调
	updatedData2 := []byte(`{"key": "value3"}`)
	err = provider.Save(updatedData2)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// 等待回调被触发
	time.Sleep(200 * time.Millisecond)
	if !callbackTriggered {
		t.Error("Callback should be triggered after Watch() is called")
	}
}

func TestGormProvider_WatchMultipleTimes(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	provider, err := NewGormProviderWithOptions(&GormProviderOptions{
		ConfigID:     "test_config",
		Driver:       "sqlite",
		DSN:          dbFile,
		PollInterval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to create GormProvider: %v", err)
	}
	defer provider.Close()

	// 多次调用 Watch 应该是安全的（通过 sync.Once 保证）
	err = provider.Watch()
	if err != nil {
		t.Fatalf("First Watch() call failed: %v", err)
	}

	err = provider.Watch()
	if err != nil {
		t.Fatalf("Second Watch() call failed: %v", err)
	}

	err = provider.Watch()
	if err != nil {
		t.Fatalf("Third Watch() call failed: %v", err)
	}

	// 应该仍然正常工作
	initialData := []byte(`{"key": "value1"}`)
	err = provider.Save(initialData)
	if err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	_, err = provider.Load()
	if err != nil {
		t.Fatalf("Failed to load initial config: %v", err)
	}

	callbackTriggered := false
	provider.OnChange(func(data []byte) error {
		callbackTriggered = true
		return nil
	})

	updatedData := []byte(`{"key": "value2"}`)
	err = provider.Save(updatedData)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	if !callbackTriggered {
		t.Error("Callback should be triggered after multiple Watch() calls")
	}
}
