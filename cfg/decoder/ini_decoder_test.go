package decoder

import (
	"strings"
	"testing"
	"time"
)

func TestIniDecoder_BasicINI(t *testing.T) {
	decoder := NewIniDecoder()

	iniData := `
# 应用配置
name = test-app
version = 1.0.0

[database]
host = localhost
port = 5432
max_connections = 10
timeout = 30
`

	storage, err := decoder.Decode([]byte(iniData))
	if err != nil {
		t.Fatalf("Failed to decode INI: %v", err)
	}

	// 测试简单字段访问（默认section）
	nameStorage := storage.Sub("name")
	var name string
	err = nameStorage.ConvertTo(&name)
	if err != nil {
		t.Fatalf("Failed to get name: %v", err)
	}
	if name != "test-app" {
		t.Errorf("Expected name 'test-app', got %v", name)
	}

	// 测试section字段访问
	hostStorage := storage.Sub("database.host")
	var host string
	err = hostStorage.ConvertTo(&host)
	if err != nil {
		t.Fatalf("Failed to get database host: %v", err)
	}
	if host != "localhost" {
		t.Errorf("Expected host 'localhost', got %v", host)
	}

	// 测试section内的字段访问
	maxConnStorage := storage.Sub("database.max_connections")
	var maxConn int64
	err = maxConnStorage.ConvertTo(&maxConn)
	if err != nil {
		t.Fatalf("Failed to get max connections: %v", err)
	}
	if maxConn != 10 {
		t.Errorf("Expected max connections 10, got %v", maxConn)
	}
}

func TestIniDecoder_INIWithComments(t *testing.T) {
	decoder := NewIniDecoder()

	iniData := `
# 应用基本配置
name = test-app          ; 应用名称
version = 1.0.0
enabled = true           # 是否启用

[database]
host = localhost         ; 数据库主机
port = 5432
timeout = 30s            # 连接超时时间

# 功能开关
[features]
logging = true
metrics = false
debug_mode = false
`

	storage, err := decoder.Decode([]byte(iniData))
	if err != nil {
		t.Fatalf("Failed to decode INI with comments: %v", err)
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

	// 测试布尔值解析
	enabledStorage := storage.Sub("enabled")
	var enabled bool
	err = enabledStorage.ConvertTo(&enabled)
	if err != nil {
		t.Fatalf("Failed to get enabled: %v", err)
	}
	if !enabled {
		t.Errorf("Expected enabled to be true, got %v", enabled)
	}

	// 测试时间类型转换（通过Storage）
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

	// 测试section中的布尔字段
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

func TestIniDecoder_ArrayValues(t *testing.T) {
	decoder := NewIniDecoder()

	iniData := `
# 数组测试
hosts = server1,server2,server3
ports = 8080,8081,8082
tags = web,api,backend

[monitoring]
exporters = prometheus,statsd
`

	storage, err := decoder.Decode([]byte(iniData))
	if err != nil {
		t.Fatalf("Failed to decode INI with arrays: %v", err)
	}

	// 测试字符串数组
	hostsStorage := storage.Sub("hosts")
	var hosts []string
	err = hostsStorage.ConvertTo(&hosts)
	if err != nil {
		t.Fatalf("Failed to get hosts array: %v", err)
	}
	expectedHosts := []string{"server1", "server2", "server3"}
	if len(hosts) != len(expectedHosts) {
		t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(hosts))
	}
	for i, expected := range expectedHosts {
		if i < len(hosts) && hosts[i] != expected {
			t.Errorf("Expected host %d to be %s, got %s", i, expected, hosts[i])
		}
	}

	// 测试数字数组转换
	portsStorage := storage.Sub("ports")
	var ports []int
	err = portsStorage.ConvertTo(&ports)
	if err != nil {
		t.Fatalf("Failed to get ports array: %v", err)
	}
	expectedPorts := []int{8080, 8081, 8082}
	if len(ports) != len(expectedPorts) {
		t.Errorf("Expected %d ports, got %d", len(expectedPorts), len(ports))
	}
	for i, expected := range expectedPorts {
		if i < len(ports) && ports[i] != expected {
			t.Errorf("Expected port %d to be %d, got %d", i, expected, ports[i])
		}
	}
}

func TestIniDecoder_BooleanKeys(t *testing.T) {
	decoder := NewIniDecoder()

	iniData := `
# 布尔键测试（无值的键）
enable_feature_a
enable_feature_b

[flags]
debug
verbose
production = false
`

	storage, err := decoder.Decode([]byte(iniData))
	if err != nil {
		t.Fatalf("Failed to decode INI with boolean keys: %v", err)
	}

	// 测试默认section的布尔键
	featureAStorage := storage.Sub("enable_feature_a")
	var featureA bool
	err = featureAStorage.ConvertTo(&featureA)
	if err != nil {
		t.Fatalf("Failed to get feature A: %v", err)
	}
	if !featureA {
		t.Errorf("Expected feature A to be true, got %v", featureA)
	}

	// 测试section中的布尔键
	debugStorage := storage.Sub("flags.debug")
	var debug bool
	err = debugStorage.ConvertTo(&debug)
	if err != nil {
		t.Fatalf("Failed to get debug flag: %v", err)
	}
	if !debug {
		t.Errorf("Expected debug to be true, got %v", debug)
	}

	// 测试section中的明确布尔值
	productionStorage := storage.Sub("flags.production")
	var production bool
	err = productionStorage.ConvertTo(&production)
	if err != nil {
		t.Fatalf("Failed to get production flag: %v", err)
	}
	if production {
		t.Errorf("Expected production to be false, got %v", production)
	}
}

func TestIniDecoder_Encode(t *testing.T) {
	decoder := NewIniDecoder()

	// 原始数据
	originalData := `
name = test-app
version = 1.0.0

[database]
host = localhost
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

func TestIniDecoder_ComplexStructure(t *testing.T) {
	decoder := NewIniDecoder()

	iniData := `
# 应用配置
app_name = config-service
version = 1.2.3
listen_port = 8080

[database]
driver = mysql
host = db.example.com
port = 3306
username = dbuser
password = secret123
max_connections = 20
timeout = 30s

[cache]
type = redis
host = cache.example.com
port = 6379
ttl = 300s

[logging]
level = info
format = json
output = /var/log/app.log

[features]
enable_auth = true
enable_cache = true
enable_metrics = false
`

	storage, err := decoder.Decode([]byte(iniData))
	if err != nil {
		t.Fatalf("Failed to decode complex INI: %v", err)
	}

	// 测试应用配置
	appNameStorage := storage.Sub("app_name")
	var appName string
	err = appNameStorage.ConvertTo(&appName)
	if err != nil {
		t.Fatalf("Failed to get app name: %v", err)
	}
	if appName != "config-service" {
		t.Errorf("Expected app name 'config-service', got %v", appName)
	}

	// 测试数字类型转换
	portStorage := storage.Sub("listen_port")
	var port int64
	err = portStorage.ConvertTo(&port)
	if err != nil {
		t.Fatalf("Failed to get listen port: %v", err)
	}
	if port != 8080 {
		t.Errorf("Expected listen port 8080, got %v", port)
	}

	// 测试时间类型转换
	dbTimeoutStorage := storage.Sub("database.timeout")
	var dbTimeout time.Duration
	err = dbTimeoutStorage.ConvertTo(&dbTimeout)
	if err != nil {
		t.Fatalf("Failed to get database timeout: %v", err)
	}
	expectedTimeout := 30 * time.Second
	if dbTimeout != expectedTimeout {
		t.Errorf("Expected database timeout %v, got %v", expectedTimeout, dbTimeout)
	}

	// 测试结构体转换
	type DatabaseConfig struct {
		Driver         string        `ini:"driver"`
		Host           string        `ini:"host"`
		Port           int64         `ini:"port"`
		Username       string        `ini:"username"`
		Password       string        `ini:"password"`
		MaxConnections int64         `ini:"max_connections"`
		Timeout        time.Duration `ini:"timeout"`
	}

	dbConfigStorage := storage.Sub("database")
	var dbConfig DatabaseConfig
	err = dbConfigStorage.ConvertTo(&dbConfig)
	if err != nil {
		t.Fatalf("Failed to convert database config: %v", err)
	}

	if dbConfig.Driver != "mysql" {
		t.Errorf("Expected driver 'mysql', got %v", dbConfig.Driver)
	}
	if dbConfig.Host != "db.example.com" {
		t.Errorf("Expected host 'db.example.com', got %v", dbConfig.Host)
	}
	if dbConfig.Port != 3306 {
		t.Errorf("Expected port 3306, got %v", dbConfig.Port)
	}
	if dbConfig.MaxConnections != 20 {
		t.Errorf("Expected max connections 20, got %v", dbConfig.MaxConnections)
	}
	if dbConfig.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", dbConfig.Timeout)
	}
}

func TestIniDecoder_SpecialValues(t *testing.T) {
	decoder := NewIniDecoder()

	iniData := `
# 特殊值测试
empty_string = 
quoted_string = "hello world"
path_value = /usr/local/bin
url_value = https://example.com/api
email_value = admin@example.com

# 数字类型
integer_value = 42
float_value = 3.14
negative_value = -100

# 布尔值的不同表示
bool_true = true
bool_false = false
bool_yes = yes
bool_no = no
bool_on = on
bool_off = off

[special]
# 包含特殊字符的值
special_chars = !@#$%^&*()
multiline_value = line1\nline2\nline3
`

	storage, err := decoder.Decode([]byte(iniData))
	if err != nil {
		t.Fatalf("Failed to decode INI with special values: %v", err)
	}

	// 测试空字符串
	emptyStorage := storage.Sub("empty_string")
	var empty string
	err = emptyStorage.ConvertTo(&empty)
	if err != nil {
		t.Fatalf("Failed to get empty string: %v", err)
	}
	if empty != "" {
		t.Errorf("Expected empty string, got %q", empty)
	}

	// 测试引号字符串
	quotedStorage := storage.Sub("quoted_string")
	var quoted string
	err = quotedStorage.ConvertTo(&quoted)
	if err != nil {
		t.Fatalf("Failed to get quoted string: %v", err)
	}
	// INI库可能会保留引号，也可能会去掉引号
	if !strings.Contains(quoted, "hello world") {
		t.Errorf("Expected quoted string to contain 'hello world', got %q", quoted)
	}

	// 测试数字解析
	intStorage := storage.Sub("integer_value")
	var intVal int64
	err = intStorage.ConvertTo(&intVal)
	if err != nil {
		t.Fatalf("Failed to get integer value: %v", err)
	}
	if intVal != 42 {
		t.Errorf("Expected integer 42, got %v", intVal)
	}

	floatStorage := storage.Sub("float_value")
	var floatVal float64
	err = floatStorage.ConvertTo(&floatVal)
	if err != nil {
		t.Fatalf("Failed to get float value: %v", err)
	}
	if floatVal != 3.14 {
		t.Errorf("Expected float 3.14, got %v", floatVal)
	}

	// 测试布尔值解析
	boolTrueStorage := storage.Sub("bool_true")
	var boolTrue bool
	err = boolTrueStorage.ConvertTo(&boolTrue)
	if err != nil {
		t.Fatalf("Failed to get bool true: %v", err)
	}
	if !boolTrue {
		t.Errorf("Expected bool true, got %v", boolTrue)
	}
}