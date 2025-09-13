package storage

import (
	"testing"
	"time"
)

func TestMapStorage_Equals(t *testing.T) {
	data1 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		},
		"servers": []interface{}{"server1", "server2"},
	}

	data2 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		},
		"servers": []interface{}{"server1", "server2"},
	}

	data3 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3307, // 不同的端口
		},
		"servers": []interface{}{"server1", "server2"},
	}

	storage1 := NewMapStorage(data1)
	storage2 := NewMapStorage(data2)
	storage3 := NewMapStorage(data3)

	// 测试相同数据的比较
	if !storage1.Equals(storage2) {
		t.Error("Expected storage1 to equal storage2")
	}

	// 测试不同数据的比较
	if storage1.Equals(storage3) {
		t.Error("Expected storage1 to not equal storage3")
	}

	// 测试与 nil 的比较
	if storage1.Equals(nil) {
		t.Error("Expected storage1 to not equal nil")
	}

	// 测试 Sub 后的比较
	sub1 := storage1.Sub("database")
	sub2 := storage2.Sub("database")
	sub3 := storage3.Sub("database")

	if !sub1.Equals(sub2) {
		t.Error("Expected sub1 to equal sub2")
	}

	if sub1.Equals(sub3) {
		t.Error("Expected sub1 to not equal sub3")
	}

	// 测试空数据比较
	empty1 := NewMapStorage(nil)
	empty2 := NewMapStorage(nil)
	emptyMap1 := NewMapStorage(map[string]interface{}{})
	emptyMap2 := NewMapStorage(map[string]interface{}{})

	if !empty1.Equals(empty2) {
		t.Error("Expected empty1 to equal empty2")
	}

	if !emptyMap1.Equals(emptyMap2) {
		t.Error("Expected emptyMap1 to equal emptyMap2")
	}

	// nil 和 {} 在 reflect.DeepEqual 中不相等，这是预期的行为
	if empty1.Equals(emptyMap1) {
		t.Error("Expected empty1 to not equal emptyMap1 (nil vs empty map)")
	}
}

func TestMapStorage_Sub(t *testing.T) {
	data := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
			"connections": []interface{}{
				map[string]interface{}{
					"name": "primary",
					"user": "admin",
				},
				map[string]interface{}{
					"name": "secondary",
					"user": "readonly",
				},
			},
		},
		"servers": []interface{}{
			"server1",
			"server2",
		},
	}

	storage := NewMapStorage(data)

	// 测试简单字段访问
	dbStorage := storage.Sub("database")
	var dbConfig map[string]interface{}
	err := dbStorage.ConvertTo(&dbConfig)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if dbConfig["host"] != "localhost" {
		t.Errorf("Expected host to be localhost, got %v", dbConfig["host"])
	}

	// 测试嵌套字段访问
	hostStorage := storage.Sub("database.host")
	var host string
	err = hostStorage.ConvertTo(&host)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if host != "localhost" {
		t.Errorf("Expected host to be localhost, got %v", host)
	}

	// 测试数组索引访问
	firstConnStorage := storage.Sub("database.connections[0]")
	var firstConn map[string]interface{}
	err = firstConnStorage.ConvertTo(&firstConn)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if firstConn["name"] != "primary" {
		t.Errorf("Expected first connection name to be primary, got %v", firstConn["name"])
	}

	// 测试嵌套数组字段访问
	firstConnNameStorage := storage.Sub("database.connections[0].name")
	var connName string
	err = firstConnNameStorage.ConvertTo(&connName)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if connName != "primary" {
		t.Errorf("Expected connection name to be primary, got %v", connName)
	}
}

func TestMapStorage_ConvertTo_Struct(t *testing.T) {
	data := map[string]interface{}{
		"name":    "test-server",
		"port":    8080,
		"enabled": true,
	}

	storage := NewMapStorage(data)

	type ServerConfig struct {
		Name    string `json:"name"`
		Port    int    `json:"port"`
		Enabled bool   `json:"enabled"`
	}

	var config ServerConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if config.Name != "test-server" {
		t.Errorf("Expected name to be test-server, got %v", config.Name)
	}
	if config.Port != 8080 {
		t.Errorf("Expected port to be 8080, got %v", config.Port)
	}
	if !config.Enabled {
		t.Errorf("Expected enabled to be true, got %v", config.Enabled)
	}
}

func TestMapStorage_ConvertTo_Slice(t *testing.T) {
	data := []interface{}{"item1", "item2", "item3"}

	storage := NewMapStorage(data)

	var items []string
	err := storage.ConvertTo(&items)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	expected := []string{"item1", "item2", "item3"}
	if len(items) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(items))
	}

	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected item %d to be %s, got %s", i, expected[i], item)
		}
	}
}

func TestMapStorage_ConvertTo_Duration(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected time.Duration
	}{
		{
			name:     "string duration",
			data:     "5m30s",
			expected: 5*time.Minute + 30*time.Second,
		},
		{
			name:     "string hour duration",
			data:     "2h15m",
			expected: 2*time.Hour + 15*time.Minute,
		},
		{
			name:     "nanoseconds as int64",
			data:     int64(1000000000), // 1 second
			expected: time.Second,
		},
		{
			name:     "seconds as float64",
			data:     2.5, // 2.5 seconds
			expected: 2*time.Second + 500*time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMapStorage(tt.data)
			var duration time.Duration
			err := storage.ConvertTo(&duration)
			if err != nil {
				t.Fatalf("ConvertTo failed: %v", err)
			}
			if duration != tt.expected {
				t.Errorf("Expected duration %v, got %v", tt.expected, duration)
			}
		})
	}
}

func TestMapStorage_ConvertTo_Time(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected time.Time
	}{
		{
			name:     "RFC3339 string",
			data:     "2023-12-25T15:30:45Z",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		},
		{
			name:     "date only string",
			data:     "2023-12-25",
			expected: time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "datetime string",
			data:     "2023-12-25 15:30:45",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		},
		{
			name:     "unix timestamp int64",
			data:     int64(1703517045), // 2023-12-25 15:30:45 UTC
			expected: time.Unix(1703517045, 0),
		},
		{
			name:     "unix timestamp float64",
			data:     1703517045.5, // 2023-12-25 15:30:45.5 UTC
			expected: time.Unix(1703517045, 500000000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMapStorage(tt.data)
			var timeValue time.Time
			err := storage.ConvertTo(&timeValue)
			if err != nil {
				t.Fatalf("ConvertTo failed: %v", err)
			}
			if !timeValue.Equal(tt.expected) {
				t.Errorf("Expected time %v, got %v", tt.expected, timeValue)
			}
		})
	}
}

func TestMapStorage_ConvertTo_TimeInStruct(t *testing.T) {
	data := map[string]interface{}{
		"timeout":    "30s",
		"created_at": "2023-12-25T15:30:45Z",
		"expires_at": int64(1703517045),
	}

	storage := NewMapStorage(data)

	type Config struct {
		Timeout   time.Duration `json:"timeout"`
		CreatedAt time.Time     `json:"created_at"`
		ExpiresAt time.Time     `json:"expires_at"`
	}

	var config Config
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	expectedTimeout := 30 * time.Second
	if config.Timeout != expectedTimeout {
		t.Errorf("Expected timeout %v, got %v", expectedTimeout, config.Timeout)
	}

	expectedCreatedAt := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
	if !config.CreatedAt.Equal(expectedCreatedAt) {
		t.Errorf("Expected created_at %v, got %v", expectedCreatedAt, config.CreatedAt)
	}

	expectedExpiresAt := time.Unix(1703517045, 0)
	if !config.ExpiresAt.Equal(expectedExpiresAt) {
		t.Errorf("Expected expires_at %v, got %v", expectedExpiresAt, config.ExpiresAt)
	}
}

func TestMapStorage_ComplexNestedStructure(t *testing.T) {
	// 构造一个复杂的多层嵌套结构
	data := map[string]interface{}{
		"application": map[string]interface{}{
			"name":    "complex-service",
			"version": "1.0.0",
			"environment": map[string]interface{}{
				"type":   "production",
				"region": "us-west-2",
			},
		},
		"database": map[string]interface{}{
			"primary": map[string]interface{}{
				"host": "primary-db.example.com",
				"port": 5432,
				"credentials": map[string]interface{}{
					"username": "admin",
					"password": "secret123",
				},
				"pools": []interface{}{
					map[string]interface{}{
						"name":            "readonly",
						"max_connections": 10,
						"timeout":         "30s",
					},
					map[string]interface{}{
						"name":            "readwrite",
						"max_connections": 5,
						"timeout":         "60s",
					},
				},
			},
			"replicas": []interface{}{
				map[string]interface{}{
					"host":   "replica1.example.com",
					"port":   5432,
					"weight": 0.6,
				},
				map[string]interface{}{
					"host":   "replica2.example.com",
					"port":   5432,
					"weight": 0.4,
				},
			},
		},
		"services": []interface{}{
			map[string]interface{}{
				"name": "auth-service",
				"endpoints": []interface{}{
					"https://auth.example.com/login",
					"https://auth.example.com/logout",
				},
				"config": map[string]interface{}{
					"rate_limit": 1000,
					"timeout":    "15s",
					"retry": map[string]interface{}{
						"attempts": 3,
						"backoff":  "1s",
					},
				},
			},
			map[string]interface{}{
				"name": "notification-service",
				"endpoints": []interface{}{
					"https://notify.example.com/send",
				},
				"config": map[string]interface{}{
					"rate_limit": 500,
					"timeout":    "10s",
					"providers": []interface{}{
						map[string]interface{}{
							"type": "email",
							"smtp": map[string]interface{}{
								"host": "smtp.example.com",
								"port": 587,
								"tls":  true,
							},
						},
						map[string]interface{}{
							"type":    "sms",
							"gateway": "twilio",
							"credentials": map[string]interface{}{
								"account_sid": "AC123",
								"auth_token":  "token456",
							},
						},
					},
				},
			},
		},
		"monitoring": map[string]interface{}{
			"metrics": map[string]interface{}{
				"enabled":  true,
				"interval": "1m",
				"collectors": []interface{}{
					"prometheus",
					"statsd",
				},
			},
			"logging": map[string]interface{}{
				"level": "info",
				"outputs": []interface{}{
					map[string]interface{}{
						"type": "file",
						"path": "/var/log/app.log",
						"rotation": map[string]interface{}{
							"max_size": "100MB",
							"max_age":  "7d",
							"compress": true,
						},
					},
					map[string]interface{}{
						"type":  "elasticsearch",
						"url":   "https://es.example.com",
						"index": "app-logs",
					},
				},
			},
		},
	}

	storage := NewMapStorage(data)

	// 测试1: 简单路径访问
	appNameStorage := storage.Sub("application.name")
	var appName string
	err := appNameStorage.ConvertTo(&appName)
	if err != nil {
		t.Fatalf("Failed to get application name: %v", err)
	}
	if appName != "complex-service" {
		t.Errorf("Expected app name 'complex-service', got %v", appName)
	}

	// 测试2: 数组索引访问
	firstReplicaHostStorage := storage.Sub("database.replicas[0].host")
	var firstReplicaHost string
	err = firstReplicaHostStorage.ConvertTo(&firstReplicaHost)
	if err != nil {
		t.Fatalf("Failed to get first replica host: %v", err)
	}
	if firstReplicaHost != "replica1.example.com" {
		t.Errorf("Expected first replica host 'replica1.example.com', got %v", firstReplicaHost)
	}

	// 测试3: 深层嵌套访问
	authRetryAttemptsStorage := storage.Sub("services[0].config.retry.attempts")
	var authRetryAttempts int
	err = authRetryAttemptsStorage.ConvertTo(&authRetryAttempts)
	if err != nil {
		t.Fatalf("Failed to get auth retry attempts: %v", err)
	}
	if authRetryAttempts != 3 {
		t.Errorf("Expected auth retry attempts 3, got %v", authRetryAttempts)
	}

	// 测试4: 时间类型转换
	poolTimeoutStorage := storage.Sub("database.primary.pools[0].timeout")
	var poolTimeout time.Duration
	err = poolTimeoutStorage.ConvertTo(&poolTimeout)
	if err != nil {
		t.Fatalf("Failed to get pool timeout: %v", err)
	}
	expectedTimeout := 30 * time.Second
	if poolTimeout != expectedTimeout {
		t.Errorf("Expected pool timeout %v, got %v", expectedTimeout, poolTimeout)
	}

	// 测试5: 复杂结构体转换
	type DatabasePool struct {
		Name           string        `json:"name"`
		MaxConnections int           `json:"max_connections"`
		Timeout        time.Duration `json:"timeout"`
	}

	firstPoolStorage := storage.Sub("database.primary.pools[0]")
	var firstPool DatabasePool
	err = firstPoolStorage.ConvertTo(&firstPool)
	if err != nil {
		t.Fatalf("Failed to convert first pool: %v", err)
	}
	if firstPool.Name != "readonly" {
		t.Errorf("Expected pool name 'readonly', got %v", firstPool.Name)
	}
	if firstPool.MaxConnections != 10 {
		t.Errorf("Expected max connections 10, got %v", firstPool.MaxConnections)
	}
	if firstPool.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", firstPool.Timeout)
	}

	// 测试6: 嵌套的slice转换
	type ServiceEndpoint struct {
		Name      string                 `json:"name"`
		Endpoints []string               `json:"endpoints"`
		Config    map[string]interface{} `json:"config"`
	}

	servicesStorage := storage.Sub("services")
	var services []ServiceEndpoint
	err = servicesStorage.ConvertTo(&services)
	if err != nil {
		t.Fatalf("Failed to convert services: %v", err)
	}
	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %v", len(services))
	}
	if services[0].Name != "auth-service" {
		t.Errorf("Expected first service name 'auth-service', got %v", services[0].Name)
	}
	if len(services[0].Endpoints) != 2 {
		t.Errorf("Expected 2 endpoints for first service, got %v", len(services[0].Endpoints))
	}

	// 测试7: 深层Map访问
	type SMTPConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
		TLS  bool   `json:"tls"`
	}

	smtpConfigStorage := storage.Sub("services[1].config.providers[0].smtp")
	var smtpConfig SMTPConfig
	err = smtpConfigStorage.ConvertTo(&smtpConfig)
	if err != nil {
		t.Fatalf("Failed to convert SMTP config: %v", err)
	}
	if smtpConfig.Host != "smtp.example.com" {
		t.Errorf("Expected SMTP host 'smtp.example.com', got %v", smtpConfig.Host)
	}
	if smtpConfig.Port != 587 {
		t.Errorf("Expected SMTP port 587, got %v", smtpConfig.Port)
	}
	if !smtpConfig.TLS {
		t.Errorf("Expected SMTP TLS to be true, got %v", smtpConfig.TLS)
	}

	// 测试8: 完整的复杂结构转换
	type ComplexConfig struct {
		Application struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Environment struct {
				Type   string `json:"type"`
				Region string `json:"region"`
			} `json:"environment"`
		} `json:"application"`

		Database struct {
			Primary struct {
				Host  string         `json:"host"`
				Port  int            `json:"port"`
				Pools []DatabasePool `json:"pools"`
			} `json:"primary"`
			Replicas []struct {
				Host   string  `json:"host"`
				Port   int     `json:"port"`
				Weight float64 `json:"weight"`
			} `json:"replicas"`
		} `json:"database"`

		Services []ServiceEndpoint `json:"services"`

		Monitoring struct {
			Metrics struct {
				Enabled    bool          `json:"enabled"`
				Interval   time.Duration `json:"interval"`
				Collectors []string      `json:"collectors"`
			} `json:"metrics"`
		} `json:"monitoring"`
	}

	var complexConfig ComplexConfig
	err = storage.ConvertTo(&complexConfig)
	if err != nil {
		t.Fatalf("Failed to convert complex config: %v", err)
	}

	// 验证转换结果
	if complexConfig.Application.Name != "complex-service" {
		t.Errorf("Expected application name 'complex-service', got %v", complexConfig.Application.Name)
	}
	if complexConfig.Database.Primary.Host != "primary-db.example.com" {
		t.Errorf("Expected primary DB host 'primary-db.example.com', got %v", complexConfig.Database.Primary.Host)
	}
	if len(complexConfig.Database.Primary.Pools) != 2 {
		t.Errorf("Expected 2 DB pools, got %v", len(complexConfig.Database.Primary.Pools))
	}
	if len(complexConfig.Database.Replicas) != 2 {
		t.Errorf("Expected 2 DB replicas, got %v", len(complexConfig.Database.Replicas))
	}
	if complexConfig.Database.Replicas[0].Weight != 0.6 {
		t.Errorf("Expected first replica weight 0.6, got %v", complexConfig.Database.Replicas[0].Weight)
	}
	if len(complexConfig.Services) != 2 {
		t.Errorf("Expected 2 services, got %v", len(complexConfig.Services))
	}
	if complexConfig.Monitoring.Metrics.Interval != time.Minute {
		t.Errorf("Expected monitoring interval 1m, got %v", complexConfig.Monitoring.Metrics.Interval)
	}
	if len(complexConfig.Monitoring.Metrics.Collectors) != 2 {
		t.Errorf("Expected 2 metric collectors, got %v", len(complexConfig.Monitoring.Metrics.Collectors))
	}
}

func TestMapStorage_CfgTagPriority(t *testing.T) {
	// 测试数据
	data := map[string]interface{}{
		"custom_name": "test_value",
		"json_name":   "json_value",
		"yaml_name":   "yaml_value",
		"toml_name":   "toml_value",
		"ini_name":    "ini_value",
		"CustomName":  "field_name_value", // 用于测试字段名匹配
	}

	storage := NewMapStorage(data)

	// 定义测试结构体，cfg 标签应该有最高优先级
	type TestStruct struct {
		// cfg 标签优先级最高
		Field1 string `cfg:"custom_name" json:"json_name" yaml:"yaml_name" toml:"toml_name" ini:"ini_name"`

		// json 标签优先于 yaml/toml/ini
		Field2 string `json:"json_name" yaml:"yaml_name" toml:"toml_name" ini:"ini_name"`

		// yaml 标签优先于 toml/ini
		Field3 string `yaml:"yaml_name" toml:"toml_name" ini:"ini_name"`

		// toml 标签优先于 ini
		Field4 string `toml:"toml_name" ini:"ini_name"`

		// 只有 ini 标签
		Field5 string `ini:"ini_name"`

		// 没有标签，使用字段名
		CustomName string
	}

	var result TestStruct
	err := storage.ConvertTo(&result)
	if err != nil {
		t.Fatalf("Failed to convert to struct: %v", err)
	}

	// 验证 cfg 标签优先级最高
	if result.Field1 != "test_value" {
		t.Errorf("Expected Field1 to use cfg tag value 'test_value', got %v", result.Field1)
	}

	// 验证 json 标签优先于其他标签
	if result.Field2 != "json_value" {
		t.Errorf("Expected Field2 to use json tag value 'json_value', got %v", result.Field2)
	}

	// 验证 yaml 标签优先于 toml/ini
	if result.Field3 != "yaml_value" {
		t.Errorf("Expected Field3 to use yaml tag value 'yaml_value', got %v", result.Field3)
	}

	// 验证 toml 标签优先于 ini
	if result.Field4 != "toml_value" {
		t.Errorf("Expected Field4 to use toml tag value 'toml_value', got %v", result.Field4)
	}

	// 验证 ini 标签
	if result.Field5 != "ini_value" {
		t.Errorf("Expected Field5 to use ini tag value 'ini_value', got %v", result.Field5)
	}

	// 验证字段名匹配
	if result.CustomName != "field_name_value" {
		t.Errorf("Expected CustomName to use field name matching 'field_name_value', got %v", result.CustomName)
	}
}

func TestMapStorage_CfgTagIgnore(t *testing.T) {
	// 测试 cfg 标签的忽略功能
	data := map[string]interface{}{
		"visible_field": "should_appear",
		"hidden_field":  "should_not_appear",
	}

	storage := NewMapStorage(data)

	type TestStruct struct {
		VisibleField string `cfg:"visible_field"`
		HiddenField  string `cfg:"-"`            // 使用 - 忽略字段
		DefaultField string `cfg:"hidden_field"` // 应该能获取到值
	}

	var result TestStruct
	err := storage.ConvertTo(&result)
	if err != nil {
		t.Fatalf("Failed to convert to struct: %v", err)
	}

	// 验证正常字段
	if result.VisibleField != "should_appear" {
		t.Errorf("Expected VisibleField to be 'should_appear', got %v", result.VisibleField)
	}

	// 验证忽略字段应该为空
	if result.HiddenField != "" {
		t.Errorf("Expected HiddenField to be empty (ignored), got %v", result.HiddenField)
	}

	// 验证其他字段正常工作
	if result.DefaultField != "should_not_appear" {
		t.Errorf("Expected DefaultField to be 'should_not_appear', got %v", result.DefaultField)
	}
}

func TestMapStorage_CfgTagWithComplexData(t *testing.T) {
	// 测试复杂数据结构中的 cfg 标签
	data := map[string]interface{}{
		"app_config": map[string]interface{}{
			"service_name": "test-service",
			"listen_port":  8080,
			"debug_mode":   true,
		},
		"db_settings": map[string]interface{}{
			"connection_string": "mysql://localhost:3306/test",
			"pool_size":         20,
		},
	}

	storage := NewMapStorage(data)

	type AppConfig struct {
		ServiceName string `cfg:"service_name" json:"name"`
		ListenPort  int    `cfg:"listen_port" json:"port"`
		DebugMode   bool   `cfg:"debug_mode" json:"debug"`
	}

	type DatabaseConfig struct {
		ConnectionString string `cfg:"connection_string"`
		PoolSize         int    `cfg:"pool_size"`
	}

	type CompleteConfig struct {
		App      AppConfig      `cfg:"app_config"`
		Database DatabaseConfig `cfg:"db_settings"`
	}

	var result CompleteConfig
	err := storage.ConvertTo(&result)
	if err != nil {
		t.Fatalf("Failed to convert to complex struct: %v", err)
	}

	// 验证嵌套结构
	if result.App.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got %v", result.App.ServiceName)
	}

	if result.App.ListenPort != 8080 {
		t.Errorf("Expected listen port 8080, got %v", result.App.ListenPort)
	}

	if !result.App.DebugMode {
		t.Errorf("Expected debug mode true, got %v", result.App.DebugMode)
	}

	if result.Database.ConnectionString != "mysql://localhost:3306/test" {
		t.Errorf("Expected connection string 'mysql://localhost:3306/test', got %v", result.Database.ConnectionString)
	}

	if result.Database.PoolSize != 20 {
		t.Errorf("Expected pool size 20, got %v", result.Database.PoolSize)
	}
}

func TestMapStorage_NilHandling(t *testing.T) {
	// 测试 nil 处理特性：
	// 1. Sub 方法在没有相关 key 时返回 nil
	// 2. 对于 nil Storage，ConvertTo 不修改空指针

	data := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		},
	}

	storage := NewMapStorage(data)

	// 测试 Sub 方法返回 nil MapStorage 当 key 不存在时
	nonExistentSub := storage.Sub("nonexistent")
	if nonExistentSub == nil {
		t.Error("Sub should return a nil MapStorage, not nil interface")
	}
	if ms, ok := nonExistentSub.(*MapStorage); !ok || ms != nil {
		t.Errorf("Expected Sub to return nil *MapStorage for non-existent key, got %T: %v", nonExistentSub, ms)
	}

	// 测试嵌套路径不存在的情况
	nonExistentNestedSub := storage.Sub("database.nonexistent")
	if nonExistentNestedSub == nil {
		t.Error("Sub should return a nil MapStorage, not nil interface")
	}
	if ms, ok := nonExistentNestedSub.(*MapStorage); !ok || ms != nil {
		t.Errorf("Expected Sub to return nil *MapStorage for non-existent nested key, got %T: %v", nonExistentNestedSub, ms)
	}

	// 测试数组索引不存在的情况
	nonExistentArraySub := storage.Sub("database.connections[0]")
	if nonExistentArraySub == nil {
		t.Error("Sub should return a nil MapStorage, not nil interface")
	}
	if ms, ok := nonExistentArraySub.(*MapStorage); !ok || ms != nil {
		t.Errorf("Expected Sub to return nil *MapStorage for non-existent array index, got %T: %v", nonExistentArraySub, ms)
	}

	// 测试对 nil Storage 调用 ConvertTo 的行为
	type DatabaseConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	// 测试 1: 空指针应该保持为 nil
	var nilConfig *DatabaseConfig = nil
	err := nonExistentSub.ConvertTo(&nilConfig)
	if err != nil {
		t.Errorf("ConvertTo should not fail for nil storage, got error: %v", err)
	}
	if nilConfig != nil {
		t.Error("Expected nil pointer to remain nil when converting from nil storage")
	}

	// 测试 2: 非空指针结构的情况
	existingConfig := &DatabaseConfig{Host: "existing", Port: 5432}
	err = nonExistentSub.ConvertTo(&existingConfig)
	if err != nil {
		t.Errorf("ConvertTo should not fail for nil storage, got error: %v", err)
	}
	// 对于非空指针，行为应该是保持不变
	if existingConfig.Host != "existing" || existingConfig.Port != 5432 {
		t.Error("Expected non-nil pointer values to remain unchanged when converting from nil storage")
	}

	// 测试 3: 验证正常情况仍然工作
	validSub := storage.Sub("database")
	if validSub == nil {
		t.Error("Expected Sub to return non-nil for existing key")
	}

	var validConfig DatabaseConfig
	err = validSub.ConvertTo(&validConfig)
	if err != nil {
		t.Fatalf("ConvertTo failed for valid storage: %v", err)
	}

	if validConfig.Host != "localhost" || validConfig.Port != 3306 {
		t.Errorf("Expected valid config conversion, got host=%v port=%v", validConfig.Host, validConfig.Port)
	}
}

func TestMapStorage_NilEquals(t *testing.T) {
	// 测试 nil MapStorage 的 Equals 行为
	data := map[string]interface{}{
		"key": "value",
	}
	
	normalStorage := NewMapStorage(data)
	var nilStorage1 *MapStorage = nil
	var nilStorage2 *MapStorage = nil
	
	// 获取一个 nil storage（通过 Sub 方法返回）
	nilFromSub := normalStorage.Sub("nonexistent")
	
	// 测试 1: nil storage 与 nil storage 比较
	if !nilStorage1.Equals(nilStorage2) {
		t.Error("Expected nil MapStorage to equal nil MapStorage")
	}
	
	// 测试 2: nil storage 与从 Sub 返回的 nil storage 比较
	if !nilStorage1.Equals(nilFromSub) {
		t.Error("Expected nil MapStorage to equal nil MapStorage from Sub")
	}
	
	// 测试 3: 从 Sub 返回的 nil storage 与另一个从 Sub 返回的 nil storage 比较
	nilFromSub2 := normalStorage.Sub("another_nonexistent")
	if !nilFromSub.Equals(nilFromSub2) {
		t.Error("Expected nil MapStorage from Sub to equal nil MapStorage from Sub")
	}
	
	// 测试 4: nil storage 与正常 storage 比较
	if nilStorage1.Equals(normalStorage) {
		t.Error("Expected nil MapStorage to not equal normal MapStorage")
	}
	
	// 测试 5: 正常 storage 与 nil storage 比较
	if normalStorage.Equals(nilStorage1) {
		t.Error("Expected normal MapStorage to not equal nil MapStorage")
	}
	
	// 测试 6: nil storage 与 nil 接口比较
	if nilStorage1.Equals(nil) {
		t.Error("Expected nil MapStorage to not equal nil interface")
	}
}

func TestMapStorage_SmartPointerFieldHandling(t *testing.T) {
	// 测试指针字段的智能处理：
	// 1. 如果配置中没有该字段，保持指针字段的原始状态（nil 保持 nil，非 nil 保持不变）
	// 2. 如果配置中有该字段，即使指针字段为 nil，也创建新实例并赋值

	type InnerConfig struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	type OuterConfig struct {
		Name    string       `json:"name"`
		Inner   *InnerConfig `json:"inner"`   // 指针字段
		Optional *InnerConfig `json:"optional"` // 另一个指针字段
	}

	// 场景1: 配置中没有 inner 和 optional 字段
	t.Run("no_pointer_fields_in_config", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
			// 注意：没有 inner 和 optional 字段
		}
		storage := NewMapStorage(data)

		// 测试1a: 目标结构体的指针字段为 nil
		var config1 OuterConfig
		config1.Inner = nil
		config1.Optional = nil

		err := storage.ConvertTo(&config1)
		if err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}

		if config1.Name != "test" {
			t.Errorf("Expected name 'test', got %v", config1.Name)
		}
		// 关键断言：指针字段应该保持 nil
		if config1.Inner != nil {
			t.Error("Expected Inner to remain nil when not in config")
		}
		if config1.Optional != nil {
			t.Error("Expected Optional to remain nil when not in config")
		}

		// 测试1b: 目标结构体的指针字段已有值
		existingInner := &InnerConfig{Value: "existing", Count: 999}
		var config2 OuterConfig
		config2.Inner = existingInner
		config2.Optional = nil

		err = storage.ConvertTo(&config2)
		if err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}

		// 关键断言：已存在的指针应该保持不变
		if config2.Inner != existingInner {
			t.Error("Expected Inner to remain unchanged when not in config")
		}
		if config2.Inner.Value != "existing" || config2.Inner.Count != 999 {
			t.Error("Expected Inner values to remain unchanged when not in config")
		}
		if config2.Optional != nil {
			t.Error("Expected Optional to remain nil when not in config")
		}
	})

	// 场景2: 配置中有 inner 字段但没有 optional 字段
	t.Run("inner_field_in_config", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
			"inner": map[string]interface{}{
				"value": "configured",
				"count": 42,
			},
			// 注意：没有 optional 字段
		}
		storage := NewMapStorage(data)

		// 测试2a: 目标结构体的 inner 字段为 nil
		var config1 OuterConfig
		config1.Inner = nil
		config1.Optional = nil

		err := storage.ConvertTo(&config1)
		if err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}

		// 关键断言：Inner 应该被创建并赋值
		if config1.Inner == nil {
			t.Error("Expected Inner to be created when in config")
		} else {
			if config1.Inner.Value != "configured" {
				t.Errorf("Expected Inner.Value 'configured', got %v", config1.Inner.Value)
			}
			if config1.Inner.Count != 42 {
				t.Errorf("Expected Inner.Count 42, got %v", config1.Inner.Count)
			}
		}
		// Optional 不在配置中，应该保持 nil
		if config1.Optional != nil {
			t.Error("Expected Optional to remain nil when not in config")
		}

		// 测试2b: 目标结构体的 inner 字段已有值（应该被覆盖）
		existingInner := &InnerConfig{Value: "existing", Count: 999}
		var config2 OuterConfig
		config2.Inner = existingInner
		config2.Optional = nil

		err = storage.ConvertTo(&config2)
		if err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}

		// 关键断言：Inner 应该被重新赋值
		if config2.Inner == nil {
			t.Error("Expected Inner to be assigned when in config")
		} else {
			if config2.Inner.Value != "configured" {
				t.Errorf("Expected Inner.Value 'configured', got %v", config2.Inner.Value)
			}
			if config2.Inner.Count != 42 {
				t.Errorf("Expected Inner.Count 42, got %v", config2.Inner.Count)
			}
		}
	})

	// 场景3: 配置中两个字段都有
	t.Run("both_fields_in_config", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
			"inner": map[string]interface{}{
				"value": "inner_value",
				"count": 10,
			},
			"optional": map[string]interface{}{
				"value": "optional_value",
				"count": 20,
			},
		}
		storage := NewMapStorage(data)

		var config OuterConfig
		config.Inner = nil
		config.Optional = nil

		err := storage.ConvertTo(&config)
		if err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}

		// 两个指针字段都应该被创建并赋值
		if config.Inner == nil {
			t.Error("Expected Inner to be created when in config")
		} else {
			if config.Inner.Value != "inner_value" || config.Inner.Count != 10 {
				t.Errorf("Expected Inner values (inner_value, 10), got (%v, %v)",
					config.Inner.Value, config.Inner.Count)
			}
		}

		if config.Optional == nil {
			t.Error("Expected Optional to be created when in config")
		} else {
			if config.Optional.Value != "optional_value" || config.Optional.Count != 20 {
				t.Errorf("Expected Optional values (optional_value, 20), got (%v, %v)",
					config.Optional.Value, config.Optional.Count)
			}
		}
	})
}
