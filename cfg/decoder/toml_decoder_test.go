package decoder

import (
	"strings"
	"testing"
	"time"
)

func TestTomlDecoder_BasicTOML(t *testing.T) {
	decoder := NewTomlDecoder()

	tomlData := `
# 应用配置
name = "test-app"
version = "1.0.0"

[database]
host = "localhost"
port = 5432

[[database.pools]]
name = "primary"
max_connections = 10

[[database.pools]]
name = "secondary"
max_connections = 5
`

	storage, err := decoder.Decode([]byte(tomlData))
	if err != nil {
		t.Fatalf("Failed to decode TOML: %v", err)
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

func TestTomlDecoder_TOMLWithComments(t *testing.T) {
	decoder := NewTomlDecoder()

	tomlData := `
# 应用基本配置
name = "test-app"        # 应用名称
version = "1.0.0"

[database]
host = "localhost"       # 数据库主机
port = 5432
timeout = "30s"          # 连接超时时间

# 连接池配置
[[database.pools]]
name = "primary"
max_connections = 10     # 最大连接数

[[database.pools]]
name = "secondary"
max_connections = 5

# 功能开关
[features]
logging = true
metrics = false
`

	storage, err := decoder.Decode([]byte(tomlData))
	if err != nil {
		t.Fatalf("Failed to decode TOML with comments: %v", err)
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

func TestTomlDecoder_ComplexStructure(t *testing.T) {
	decoder := NewTomlDecoder()

	tomlData := `
# 服务配置
[[services]]
name = "auth-service"

[services.config]
timeout = "15s"

[services.config.retry]
attempts = 3
backoff = "1s"           # 退避时间

[[services]]
name = "user-service"

[services.config]
timeout = "10s"

[services.config.cache]
enabled = true
ttl = "5m"               # 缓存过期时间

# 监控配置
[monitoring]

[monitoring.metrics]
enabled = true
interval = "1m"          # 采集间隔
exporters = ["prometheus", "statsd"]

[monitoring.alerts]
email = "admin@example.com"
webhook = "https://hooks.example.com/alerts"
`

	storage, err := decoder.Decode([]byte(tomlData))
	if err != nil {
		t.Fatalf("Failed to decode complex TOML: %v", err)
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

	// 测试结构体转换 - 这里TOML的结构和JSON/YAML不同，需要适配
	type ServiceConfig struct {
		Name   string `toml:"name"`
		Config struct {
			Timeout time.Duration `toml:"timeout"`
			Retry   struct {
				Attempts int           `toml:"attempts"`
				Backoff  time.Duration `toml:"backoff"`
			} `toml:"retry"`
		} `toml:"config"`
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

func TestTomlDecoder_Encode(t *testing.T) {
	decoder := NewTomlDecoder()

	// 原始数据
	originalData := `
name = "test-app"
version = "1.0.0"

[database]
host = "localhost"
port = 5432
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

func TestTomlDecoder_SpecialTOMLFeatures(t *testing.T) {
	decoder := NewTomlDecoder()

	tomlData := `
# TOML 特殊功能测试
title = "TOML Example"

[owner]
name = "Tom Preston-Werner"
dob = 1979-05-27T07:32:00-08:00  # 日期时间

# 数组
products = [
  "Hammer",
  "Nail"
]

# 嵌套表
[person]
first = "Tom"
last = "Preston-Werner"

# 数字类型
[numbers]
integer = 42
float = 3.14
scientific = 1.2e+10
hex = 0xDEADBEEF
octal = 0o755
binary = 0b11010110

# 字符串类型
[strings]
basic = "I'm a string"
multiline = """
Roses are red
Violets are blue"""

literal = 'C:\Users\nodejs\templates'
literal_multiline = '''
The first newline is
trimmed in raw strings.
   All other whitespace
   is preserved.
'''

# 布尔值
[booleans]
true_val = true
false_val = false

# 数组表
[[database.servers]]
ip = "10.0.0.1"
dc = "eqdc10"

[[database.servers]]
ip = "10.0.0.2"
dc = "eqdc10"
`

	storage, err := decoder.Decode([]byte(tomlData))
	if err != nil {
		t.Fatalf("Failed to decode TOML with special features: %v", err)
	}

	// 测试基本字符串
	titleStorage := storage.Sub("title")
	var title string
	err = titleStorage.ConvertTo(&title)
	if err != nil {
		t.Fatalf("Failed to get title: %v", err)
	}
	if title != "TOML Example" {
		t.Errorf("Expected title 'TOML Example', got %v", title)
	}

	// 测试嵌套表
	firstNameStorage := storage.Sub("person.first")
	var firstName string
	err = firstNameStorage.ConvertTo(&firstName)
	if err != nil {
		t.Fatalf("Failed to get first name: %v", err)
	}
	if firstName != "Tom" {
		t.Errorf("Expected first name 'Tom', got %v", firstName)
	}

	// 测试数字类型
	intStorage := storage.Sub("numbers.integer")
	var intVal int64
	err = intStorage.ConvertTo(&intVal)
	if err != nil {
		t.Fatalf("Failed to get integer: %v", err)
	}
	if intVal != 42 {
		t.Errorf("Expected integer 42, got %v", intVal)
	}

	floatStorage := storage.Sub("numbers.float")
	var floatVal float64
	err = floatStorage.ConvertTo(&floatVal)
	if err != nil {
		t.Fatalf("Failed to get float: %v", err)
	}
	if floatVal != 3.14 {
		t.Errorf("Expected float 3.14, got %v", floatVal)
	}

	// 测试多行字符串
	multilineStorage := storage.Sub("strings.multiline")
	var multiline string
	err = multilineStorage.ConvertTo(&multiline)
	if err != nil {
		t.Fatalf("Failed to get multiline string: %v", err)
	}
	if !strings.Contains(multiline, "Roses") {
		t.Errorf("Expected multiline string to contain 'Roses', got %q", multiline)
	}

	// 测试数组表
	firstServerIPStorage := storage.Sub("database.servers[0].ip")
	var firstServerIP string
	err = firstServerIPStorage.ConvertTo(&firstServerIP)
	if err != nil {
		t.Fatalf("Failed to get first server IP: %v", err)
	}
	if firstServerIP != "10.0.0.1" {
		t.Errorf("Expected first server IP '10.0.0.1', got %v", firstServerIP)
	}
}