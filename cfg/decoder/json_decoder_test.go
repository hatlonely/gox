package decoder

import (
	"testing"
	"time"
)

func TestJsonDecoder_StandardJSON(t *testing.T) {
	decoder := NewJsonDecoderWithOptions(&JsonDecoderOptions{
		UseJSON5: false, // 不使用JSON5
	})

	jsonData := `{
		"name": "test-app",
		"version": "1.0.0",
		"database": {
			"host": "localhost",
			"port": 5432,
			"pools": [
				{
					"name": "primary",
					"max_connections": 10
				},
				{
					"name": "secondary",
					"max_connections": 5
				}
			]
		}
	}`

	storage, err := decoder.Decode([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// 测试简单字段访问
	nameStorage := storage.Sub("name")
	var name string
	err = nameStorage.ConvertTo(&name)
	if err != nil {
		t.Fatalf("Failed to get name: %v", err)
	}
	if name != "test-app" {
		t.Errorf("Expected name 'test-app', got %v", name)
	}

	// 测试嵌套字段访问
	hostStorage := storage.Sub("database.host")
	var host string
	err = hostStorage.ConvertTo(&host)
	if err != nil {
		t.Fatalf("Failed to get database host: %v", err)
	}
	if host != "localhost" {
		t.Errorf("Expected host 'localhost', got %v", host)
	}

	// 测试数组访问
	firstPoolNameStorage := storage.Sub("database.pools[0].name")
	var firstPoolName string
	err = firstPoolNameStorage.ConvertTo(&firstPoolName)
	if err != nil {
		t.Fatalf("Failed to get first pool name: %v", err)
	}
	if firstPoolName != "primary" {
		t.Errorf("Expected first pool name 'primary', got %v", firstPoolName)
	}
}

func TestJsonDecoder_JSON5WithComments(t *testing.T) {
	decoder := NewJsonDecoder() // 默认使用JSON5

	json5Data := `{
		// 应用基本信息
		"name": "test-app", // 应用名称
		"version": "1.0.0",
		
		/* 
		 * 数据库配置
		 * 包含主机、端口和连接池设置
		 */
		"database": {
			"host": "localhost", // 数据库主机
			"port": 5432,
			"timeout": "30s", // 连接超时时间
			"pools": [
				{
					"name": "primary",
					"max_connections": 10, // 最大连接数
				}, // 尾随逗号
				{
					"name": "secondary",
					"max_connections": 5,
				}
			],
		}, // 尾随逗号
		
		// 功能开关
		"features": {
			"logging": true,
			"metrics": false,
		}
	}`

	storage, err := decoder.Decode([]byte(json5Data))
	if err != nil {
		t.Fatalf("Failed to decode JSON5: %v", err)
	}

	// 测试基本字段
	nameStorage := storage.Sub("name")
	var name string
	err = nameStorage.ConvertTo(&name)
	if err != nil {
		t.Fatalf("Failed to get name: %v", err)
	}
	if name != "test-app" {
		t.Errorf("Expected name 'test-app', got %v", name)
	}

	// 测试时间类型转换
	timeoutStorage := storage.Sub("database.timeout")
	var timeout time.Duration
	err = timeoutStorage.ConvertTo(&timeout)
	if err != nil {
		t.Fatalf("Failed to get timeout: %v", err)
	}
	expectedTimeout := 30 * time.Second
	if timeout != expectedTimeout {
		t.Errorf("Expected timeout %v, got %v", expectedTimeout, timeout)
	}

	// 测试布尔字段
	loggingStorage := storage.Sub("features.logging")
	var logging bool
	err = loggingStorage.ConvertTo(&logging)
	if err != nil {
		t.Fatalf("Failed to get logging feature: %v", err)
	}
	if !logging {
		t.Errorf("Expected logging to be true, got %v", logging)
	}
}

func TestJsonDecoder_ComplexStructureWithJSON5(t *testing.T) {
	decoder := NewJsonDecoder()

	json5Data := `{
		// 服务配置
		"services": [
			{
				"name": "auth-service",
				"config": {
					"timeout": "15s",
					"retry": {
						"attempts": 3,
						"backoff": "1s", // 退避时间
					}
				}
			},
			{
				"name": "user-service", 
				"config": {
					"timeout": "10s",
					"cache": {
						"enabled": true,
						"ttl": "5m", // 缓存过期时间
					},
				}
			}, // 尾随逗号
		],
		
		/* 
		 * 监控配置
		 * 包含指标收集和告警设置
		 */
		"monitoring": {
			"metrics": {
				"enabled": true,
				"interval": "1m", // 采集间隔
				"exporters": ["prometheus", "statsd"],
			},
			"alerts": {
				"email": "admin@example.com",
				"webhook": "https://hooks.example.com/alerts",
			}
		}
	}`

	storage, err := decoder.Decode([]byte(json5Data))
	if err != nil {
		t.Fatalf("Failed to decode complex JSON5: %v", err)
	}

	// 测试深层嵌套访问
	authTimeoutStorage := storage.Sub("services[0].config.timeout")
	var authTimeout time.Duration
	err = authTimeoutStorage.ConvertTo(&authTimeout)
	if err != nil {
		t.Fatalf("Failed to get auth service timeout: %v", err)
	}
	expectedAuthTimeout := 15 * time.Second
	if authTimeout != expectedAuthTimeout {
		t.Errorf("Expected auth timeout %v, got %v", expectedAuthTimeout, authTimeout)
	}

	// 测试数组字段
	exportersStorage := storage.Sub("monitoring.metrics.exporters")
	var exporters []string
	err = exportersStorage.ConvertTo(&exporters)
	if err != nil {
		t.Fatalf("Failed to get exporters: %v", err)
	}
	expectedExporters := []string{"prometheus", "statsd"}
	if len(exporters) != len(expectedExporters) {
		t.Errorf("Expected %d exporters, got %d", len(expectedExporters), len(exporters))
	}
	for i, exp := range expectedExporters {
		if exporters[i] != exp {
			t.Errorf("Expected exporter %d to be %s, got %s", i, exp, exporters[i])
		}
	}

	// 测试结构体转换
	type ServiceConfig struct {
		Name   string `json:"name"`
		Config struct {
			Timeout time.Duration `json:"timeout"`
			Retry   struct {
				Attempts int           `json:"attempts"`
				Backoff  time.Duration `json:"backoff"`
			} `json:"retry"`
		} `json:"config"`
	}

	firstServiceStorage := storage.Sub("services[0]")
	var firstService ServiceConfig
	err = firstServiceStorage.ConvertTo(&firstService)
	if err != nil {
		t.Fatalf("Failed to convert first service: %v", err)
	}

	if firstService.Name != "auth-service" {
		t.Errorf("Expected service name 'auth-service', got %v", firstService.Name)
	}
	if firstService.Config.Timeout != 15*time.Second {
		t.Errorf("Expected service timeout 15s, got %v", firstService.Config.Timeout)
	}
	if firstService.Config.Retry.Attempts != 3 {
		t.Errorf("Expected retry attempts 3, got %v", firstService.Config.Retry.Attempts)
	}
	if firstService.Config.Retry.Backoff != time.Second {
		t.Errorf("Expected backoff 1s, got %v", firstService.Config.Retry.Backoff)
	}
}

func TestJsonDecoder_Encode(t *testing.T) {
	decoder := NewJsonDecoder()

	// 原始数据
	originalData := `{
		"name": "test-app",
		"version": "1.0.0",
		"database": {
			"host": "localhost",
			"port": 5432
		}
	}`

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

	// 验证数据一致性
	nameStorage := storage2.Sub("name")
	var name string
	err = nameStorage.ConvertTo(&name)
	if err != nil {
		t.Fatalf("Failed to get name from encoded data: %v", err)
	}
	if name != "test-app" {
		t.Errorf("Expected name 'test-app', got %v", name)
	}

	hostStorage := storage2.Sub("database.host")
	var host string
	err = hostStorage.ConvertTo(&host)
	if err != nil {
		t.Fatalf("Failed to get host from encoded data: %v", err)
	}
	if host != "localhost" {
		t.Errorf("Expected host 'localhost', got %v", host)
	}
}