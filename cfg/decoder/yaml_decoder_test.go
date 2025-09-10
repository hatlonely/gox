package decoder

import (
	"strings"
	"testing"
	"time"
)

func TestYamlDecoder_BasicYAML(t *testing.T) {
	decoder := NewYamlDecoder()

	yamlData := `
name: test-app
version: 1.0.0
database:
  host: localhost
  port: 5432
  pools:
    - name: primary
      max_connections: 10
    - name: secondary
      max_connections: 5
`

	storage, err := decoder.Decode([]byte(yamlData))
	if err != nil {
		t.Fatalf("Failed to decode YAML: %v", err)
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

func TestYamlDecoder_YAMLWithComments(t *testing.T) {
	decoder := NewYamlDecoder()

	yamlData := `
# 应用配置
name: test-app # 应用名称
version: 1.0.0

# 数据库配置
database:
  host: localhost  # 数据库主机
  port: 5432
  timeout: 30s     # 连接超时时间
  pools:
    - name: primary
      max_connections: 10  # 最大连接数
    - name: secondary
      max_connections: 5

# 功能开关
features:
  logging: true
  metrics: false
`

	storage, err := decoder.Decode([]byte(yamlData))
	if err != nil {
		t.Fatalf("Failed to decode YAML with comments: %v", err)
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

func TestYamlDecoder_ComplexStructure(t *testing.T) {
	decoder := NewYamlDecoder()

	yamlData := `
# 服务配置
services:
  - name: auth-service
    config:
      timeout: 15s
      retry:
        attempts: 3
        backoff: 1s  # 退避时间
  - name: user-service
    config:
      timeout: 10s
      cache:
        enabled: true
        ttl: 5m      # 缓存过期时间

# 监控配置
monitoring:
  metrics:
    enabled: true
    interval: 1m     # 采集间隔
    exporters:
      - prometheus
      - statsd
  alerts:
    email: admin@example.com
    webhook: https://hooks.example.com/alerts
`

	storage, err := decoder.Decode([]byte(yamlData))
	if err != nil {
		t.Fatalf("Failed to decode complex YAML: %v", err)
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
		Name   string `yaml:"name"`
		Config struct {
			Timeout time.Duration `yaml:"timeout"`
			Retry   struct {
				Attempts int           `yaml:"attempts"`
				Backoff  time.Duration `yaml:"backoff"`
			} `yaml:"retry"`
		} `yaml:"config"`
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

func TestYamlDecoder_Encode(t *testing.T) {
	decoder := NewYamlDecoder()

	// 原始数据
	originalData := `
name: test-app
version: 1.0.0
database:
  host: localhost
  port: 5432
`

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

func TestYamlDecoder_MultiDocument(t *testing.T) {
	decoder := NewYamlDecoder()

	// YAML多文档格式（只取第一个文档）
	yamlData := `
---
name: first-app
version: 1.0.0
---
name: second-app
version: 2.0.0
`

	storage, err := decoder.Decode([]byte(yamlData))
	if err != nil {
		t.Fatalf("Failed to decode multi-document YAML: %v", err)
	}

	// 应该解析第一个文档
	nameStorage := storage.Sub("name")
	var name string
	err = nameStorage.ConvertTo(&name)
	if err != nil {
		t.Fatalf("Failed to get name: %v", err)
	}
	if name != "first-app" {
		t.Errorf("Expected name 'first-app', got %v", name)
	}
}

func TestYamlDecoder_SpecialYAMLFeatures(t *testing.T) {
	decoder := NewYamlDecoder()

	yamlData := `
# YAML特殊功能测试
app:
  # 多行字符串 - literal block
  description: |
    This is a multi-line
    string that preserves
    line breaks
  
  # 多行字符串 - folded block  
  summary: >
    This is a long line
    that will be folded
    into a single line
    
  # 引用和锚点
  default_config: &default
    timeout: 30s
    retries: 3
    
  services:
    auth: *default
    user:
      <<: *default
      timeout: 60s  # 覆盖默认值
      
  # 类型转换
  numbers:
    integer: 42
    float: 3.14
    scientific: 1.2e+10
    
  boolean_values:
    - true
    - false
    - yes
    - no
    - on
    - off
`

	storage, err := decoder.Decode([]byte(yamlData))
	if err != nil {
		t.Fatalf("Failed to decode YAML with special features: %v", err)
	}

	// 测试多行字符串 literal
	descStorage := storage.Sub("app.description")
	var description string
	err = descStorage.ConvertTo(&description)
	if err != nil {
		t.Fatalf("Failed to get description: %v", err)
	}
	if !strings.Contains(description, "\n") {
		t.Errorf("Expected description to contain newlines, got %q", description)
	}

	// 测试多行字符串 folded
	summaryStorage := storage.Sub("app.summary")
	var summary string
	err = summaryStorage.ConvertTo(&summary)
	if err != nil {
		t.Fatalf("Failed to get summary: %v", err)
	}
	// YAML folded 字符串可能仍包含一个尾随换行符，这是正常的
	summary = strings.TrimSpace(summary)
	if strings.Contains(summary, "\n") {
		t.Errorf("Expected summary to be folded (after trimming), got %q", summary)
	}

	// 测试引用和锚点
	authTimeoutStorage := storage.Sub("app.services.auth.timeout")
	var authTimeout time.Duration
	err = authTimeoutStorage.ConvertTo(&authTimeout)
	if err != nil {
		t.Fatalf("Failed to get auth timeout: %v", err)
	}
	if authTimeout != 30*time.Second {
		t.Errorf("Expected auth timeout 30s, got %v", authTimeout)
	}

	// 测试合并和覆盖
	userTimeoutStorage := storage.Sub("app.services.user.timeout")
	var userTimeout time.Duration
	err = userTimeoutStorage.ConvertTo(&userTimeout)
	if err != nil {
		t.Fatalf("Failed to get user timeout: %v", err)
	}
	if userTimeout != 60*time.Second {
		t.Errorf("Expected user timeout 60s (overridden), got %v", userTimeout)
	}

	// 测试数字类型
	intStorage := storage.Sub("app.numbers.integer")
	var intVal int
	err = intStorage.ConvertTo(&intVal)
	if err != nil {
		t.Fatalf("Failed to get integer: %v", err)
	}
	if intVal != 42 {
		t.Errorf("Expected integer 42, got %v", intVal)
	}

	floatStorage := storage.Sub("app.numbers.float")
	var floatVal float64
	err = floatStorage.ConvertTo(&floatVal)
	if err != nil {
		t.Fatalf("Failed to get float: %v", err)
	}
	if floatVal != 3.14 {
		t.Errorf("Expected float 3.14, got %v", floatVal)
	}
}