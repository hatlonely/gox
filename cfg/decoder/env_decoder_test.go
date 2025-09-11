package decoder

import (
	"testing"
	"time"
)

func TestEnvDecoder_BasicParsing(t *testing.T) {
	decoder := NewEnvDecoder()

	envData := `# 应用基本配置
APP_NAME=test-app
APP_VERSION=1.0.0
DEBUG=true
PORT=8080
TIMEOUT=30.5`

	storage, err := decoder.Decode([]byte(envData))
	if err != nil {
		t.Fatalf("Failed to decode .env: %v", err)
	}

	// 定义配置结构体，使用 cfg 标签进行智能匹配
	type Config struct {
		Name    string  `cfg:"name"`
		Debug   bool    `cfg:"debug"`
		Port    int     `cfg:"port"`
		Timeout float64 `cfg:"timeout"`
	}

	var config Config
	err = storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to config struct: %v", err)
	}

	// 验证字段值
	if config.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got %v", config.Name)
	}
	if !config.Debug {
		t.Errorf("Expected debug to be true, got %v", config.Debug)
	}
	if config.Port != 8080 {
		t.Errorf("Expected port 8080, got %v", config.Port)
	}
	if config.Timeout != 30.5 {
		t.Errorf("Expected timeout 30.5, got %v", config.Timeout)
	}
}

func TestEnvDecoder_QuotedValues(t *testing.T) {
	decoder := NewEnvDecoder()

	envData := `# 带引号的值
MESSAGE="Hello World"
PATH="/usr/local/bin:/usr/bin"
DESCRIPTION='This is a test'
EMPTY=""
SPECIAL="Line 1\nLine 2\tTab"
QUOTE="He said \"Hello\""`

	storage, err := decoder.Decode([]byte(envData))
	if err != nil {
		t.Fatalf("Failed to decode .env: %v", err)
	}

	// 定义配置结构体
	type Config struct {
		Message     string `cfg:"message"`
		Path        string `cfg:"path"`
		Description string `cfg:"description"`
		Empty       string `cfg:"empty"`
		Special     string `cfg:"special"`
		Quote       string `cfg:"quote"`
	}

	var config Config
	err = storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to config struct: %v", err)
	}

	// 验证字段值
	if config.Message != "Hello World" {
		t.Errorf("Expected message 'Hello World', got %v", config.Message)
	}
	if config.Description != "This is a test" {
		t.Errorf("Expected description 'This is a test', got %v", config.Description)
	}
	if config.Empty != "" {
		t.Errorf("Expected empty string, got %v", config.Empty)
	}

	expected := "Line 1\nLine 2\tTab"
	if config.Special != expected {
		t.Errorf("Expected special '%s', got '%s'", expected, config.Special)
	}

	expectedQuote := "He said \"Hello\""
	if config.Quote != expectedQuote {
		t.Errorf("Expected quote '%s', got '%s'", expectedQuote, config.Quote)
	}
}

func TestEnvDecoder_NestedStructureMapping(t *testing.T) {
	decoder := NewEnvDecoder()

	envData := `# 嵌套结构配置
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=myapp
CACHE_REDIS_HOST=redis.example.com
CACHE_REDIS_PORT=6379
LOG_LEVEL=info
LOG_FILE=/var/log/app.log`

	storage, err := decoder.Decode([]byte(envData))
	if err != nil {
		t.Fatalf("Failed to decode .env: %v", err)
	}

	// 定义目标结构体
	type Config struct {
		Database struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
			Name string `cfg:"name"`
		} `cfg:"database"`
		Cache struct {
			Redis struct {
				Host string `cfg:"host"`
				Port int    `cfg:"port"`
			} `cfg:"redis"`
		} `cfg:"cache"`
		Log struct {
			Level string `cfg:"level"`
			File  string `cfg:"file"`
		} `cfg:"log"`
	}

	var config Config
	err = storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to config struct: %v", err)
	}

	// 验证数据库配置
	if config.Database.Host != "localhost" {
		t.Errorf("Expected database host 'localhost', got %v", config.Database.Host)
	}
	if config.Database.Port != 5432 {
		t.Errorf("Expected database port 5432, got %v", config.Database.Port)
	}
	if config.Database.Name != "myapp" {
		t.Errorf("Expected database name 'myapp', got %v", config.Database.Name)
	}

	// 验证缓存配置
	if config.Cache.Redis.Host != "redis.example.com" {
		t.Errorf("Expected redis host 'redis.example.com', got %v", config.Cache.Redis.Host)
	}
	if config.Cache.Redis.Port != 6379 {
		t.Errorf("Expected redis port 6379, got %v", config.Cache.Redis.Port)
	}

	// 验证日志配置
	if config.Log.Level != "info" {
		t.Errorf("Expected log level 'info', got %v", config.Log.Level)
	}
	if config.Log.File != "/var/log/app.log" {
		t.Errorf("Expected log file '/var/log/app.log', got %v", config.Log.File)
	}
}

func TestEnvDecoder_ArraySupport(t *testing.T) {
	decoder := NewEnvDecoder()

	envData := `# 数组支持
FEATURES_AUTH_0_NAME=oauth
FEATURES_AUTH_0_ENABLED=true
FEATURES_AUTH_1_NAME=ldap
FEATURES_AUTH_1_ENABLED=false`

	storage, err := decoder.Decode([]byte(envData))
	if err != nil {
		t.Fatalf("Failed to decode .env: %v", err)
	}

	// 定义目标结构体 - 只测试嵌套结构体数组，因为 FlatStorage 对简单字符串数组支持有限
	type Config struct {
		Features struct {
			Auth []struct {
				Name    string `cfg:"name"`
				Enabled bool   `cfg:"enabled"`
			} `cfg:"auth"`
		} `cfg:"features"`
	}

	var config Config
	err = storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to config struct: %v", err)
	}

	// 验证认证配置
	if len(config.Features.Auth) != 2 {
		t.Errorf("Expected 2 auth features, got %d", len(config.Features.Auth))
	}

	if len(config.Features.Auth) >= 1 {
		if config.Features.Auth[0].Name != "oauth" {
			t.Errorf("Expected auth[0] name 'oauth', got %v", config.Features.Auth[0].Name)
		}
		if !config.Features.Auth[0].Enabled {
			t.Errorf("Expected auth[0] to be enabled, got %v", config.Features.Auth[0].Enabled)
		}
	}

	if len(config.Features.Auth) >= 2 {
		if config.Features.Auth[1].Name != "ldap" {
			t.Errorf("Expected auth[1] name 'ldap', got %v", config.Features.Auth[1].Name)
		}
		if config.Features.Auth[1].Enabled {
			t.Errorf("Expected auth[1] to be disabled, got %v", config.Features.Auth[1].Enabled)
		}
	}
}

func TestEnvDecoder_CommentsAndEmptyLines(t *testing.T) {
	decoder := NewEnvDecoder()

	envData := `# 这是注释
APP_NAME=test

// 这也是注释
DEBUG=true

# 空行测试

PORT=8080`

	storage, err := decoder.Decode([]byte(envData))
	if err != nil {
		t.Fatalf("Failed to decode .env: %v", err)
	}

	// 定义配置结构体
	type Config struct {
		Name  string `cfg:"name"`
		Debug bool   `cfg:"debug"`
		Port  int    `cfg:"port"`
	}

	var config Config
	err = storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to config struct: %v", err)
	}

	// 验证只有有效的键值对被解析
	if config.Name != "test" {
		t.Errorf("Expected name 'test', got %v", config.Name)
	}
	if !config.Debug {
		t.Errorf("Expected debug to be true, got %v", config.Debug)
	}
	if config.Port != 8080 {
		t.Errorf("Expected port 8080, got %v", config.Port)
	}
}

func TestEnvDecoder_InvalidFormat(t *testing.T) {
	decoder := NewEnvDecoder()

	testCases := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "missing equals sign",
			data:    "INVALID_LINE",
			wantErr: true,
		},
		{
			name:    "empty key",
			data:    "=value",
			wantErr: true,
		},
		{
			name:    "valid with equals in value",
			data:    "URL=https://example.com?param=value",
			wantErr: false,
		},
		{
			name:    "empty value",
			data:    "EMPTY_VALUE=",
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decoder.Decode([]byte(tc.data))
			if tc.wantErr && err == nil {
				t.Errorf("Expected error for %s, but got none", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tc.name, err)
			}
		})
	}
}

func TestEnvDecoder_Encode(t *testing.T) {
	decoder := NewEnvDecoder()

	// 原始数据
	originalData := `APP_NAME=test-app
DEBUG=true
PORT=8080
MESSAGE="Hello World"
TIMEOUT=30.5`

	// 解码
	storage, err := decoder.Decode([]byte(originalData))
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// 编码
	encodedData, err := decoder.Encode(storage)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// 重新解码验证
	storage2, err := decoder.Decode(encodedData)
	if err != nil {
		t.Fatalf("Failed to decode encoded data: %v", err)
	}

	// 定义配置结构体验证数据一致性
	type Config struct {
		Name    string  `cfg:"name"`
		Debug   bool    `cfg:"debug"`
		Port    int     `cfg:"port"`
		Message string  `cfg:"message"`
		Timeout float64 `cfg:"timeout"`
	}

	var config Config
	err = storage2.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert encoded data to config struct: %v", err)
	}

	if config.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got %v", config.Name)
	}
	if !config.Debug {
		t.Errorf("Expected debug to be true, got %v", config.Debug)
	}
	if config.Message != "Hello World" {
		t.Errorf("Expected message 'Hello World', got %v", config.Message)
	}
}

func TestEnvDecoder_CustomOptions(t *testing.T) {
	// 测试自定义分隔符和数组格式
	decoder := NewEnvDecoderWithOptions("_", "_%d")

	// 使用和 FlatStorage 测试中相似的数据格式
	envData := `APP_NAME=test-service
APP_VERSION=1.0.0
DATABASE_PRIMARY_HOST=db1.example.com
DATABASE_PRIMARY_PORT=5432
CACHE_REDIS_HOST=redis.example.com
CACHE_REDIS_PORT=6379
SERVER_TIMEOUT=30s`

	storage, err := decoder.Decode([]byte(envData))
	if err != nil {
		t.Fatalf("Failed to decode .env with custom options: %v", err)
	}

	// 测试智能字段匹配，参考 FlatStorage 测试
	type DatabaseConfig struct {
		Primary struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
		} `cfg:"primary"`
	}

	type CacheConfig struct {
		Redis struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
		} `cfg:"redis"`
	}

	type Config struct {
		Name     string         `cfg:"name"`
		Version  string         `cfg:"version"`
		Database DatabaseConfig `cfg:"database"`
		Cache    CacheConfig    `cfg:"cache"`
		Server   struct {
			Timeout time.Duration `cfg:"timeout"`
		} `cfg:"server"`
	}

	var config Config
	err = storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to config struct: %v", err)
	}

	// 验证智能匹配的结果
	if config.Name != "test-service" {
		t.Errorf("Expected app name 'test-service', got %v", config.Name)
	}
	if config.Version != "1.0.0" {
		t.Errorf("Expected app version '1.0.0', got %v", config.Version)
	}
	if config.Database.Primary.Host != "db1.example.com" {
		t.Errorf("Expected database primary host 'db1.example.com', got %v", config.Database.Primary.Host)
	}
	if config.Database.Primary.Port != 5432 {
		t.Errorf("Expected database primary port 5432, got %v", config.Database.Primary.Port)
	}
	if config.Cache.Redis.Host != "redis.example.com" {
		t.Errorf("Expected cache redis host 'redis.example.com', got %v", config.Cache.Redis.Host)
	}
	if config.Cache.Redis.Port != 6379 {
		t.Errorf("Expected cache redis port 6379, got %v", config.Cache.Redis.Port)
	}
	if config.Server.Timeout != 30*time.Second {
		t.Errorf("Expected server timeout 30s, got %v", config.Server.Timeout)
	}
}

func TestEnvDecoder_TypeConversions(t *testing.T) {
	decoder := NewEnvDecoder()

	envData := `# 类型转换测试
TIMEOUT_DURATION=30s
COUNT=42
RATE=3.14
ENABLED=true
DISABLED=false`

	storage, err := decoder.Decode([]byte(envData))
	if err != nil {
		t.Fatalf("Failed to decode .env: %v", err)
	}

	// 使用结构体进行类型转换测试
	type Config struct {
		TimeoutDuration time.Duration `cfg:"timeout_duration"`
		Count           int64         `cfg:"count"`
		Rate            float64       `cfg:"rate"`
		Enabled         bool          `cfg:"enabled"`
		Disabled        bool          `cfg:"disabled"`
	}

	var config Config
	err = storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to config struct: %v", err)
	}

	// 验证各种基本类型的转换
	if config.TimeoutDuration != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.TimeoutDuration)
	}
	if config.Count != 42 {
		t.Errorf("Expected count 42, got %v", config.Count)
	}
	if config.Rate != 3.14 {
		t.Errorf("Expected rate 3.14, got %v", config.Rate)
	}
	if !config.Enabled {
		t.Errorf("Expected enabled to be true, got %v", config.Enabled)
	}
	if config.Disabled {
		t.Errorf("Expected disabled to be false, got %v", config.Disabled)
	}
}
