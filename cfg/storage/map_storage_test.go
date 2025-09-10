package storage

import (
	"testing"
	"time"
)

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
		"name": "test-server",
		"port": 8080,
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
				"type": "production",
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
						"name":         "readonly",
						"max_connections": 10,
						"timeout":      "30s",
					},
					map[string]interface{}{
						"name":         "readwrite", 
						"max_connections": 5,
						"timeout":      "60s",
					},
				},
			},
			"replicas": []interface{}{
				map[string]interface{}{
					"host": "replica1.example.com",
					"port": 5432,
					"weight": 0.6,
				},
				map[string]interface{}{
					"host": "replica2.example.com", 
					"port": 5432,
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
							"type": "sms",
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
				"enabled": true,
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
						"type": "elasticsearch",
						"url":  "https://es.example.com",
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
		Name      string            `json:"name"`
		Endpoints []string          `json:"endpoints"`
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
			Name    string `json:"name"`
			Version string `json:"version"`
			Environment struct {
				Type   string `json:"type"`
				Region string `json:"region"`
			} `json:"environment"`
		} `json:"application"`
		
		Database struct {
			Primary struct {
				Host string `json:"host"`
				Port int    `json:"port"`
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
				Enabled   bool          `json:"enabled"`
				Interval  time.Duration `json:"interval"`
				Collectors []string     `json:"collectors"`
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