package decoder

import (
	"testing"

	"github.com/hatlonely/gox/cfg/storage"
)

func TestNewCmdDecoderWithOptions(t *testing.T) {
	tests := []struct {
		name    string
		options *CmdDecoderOptions
		wantErr bool
	}{
		{
			name:    "nil options",
			options: nil,
			wantErr: false,
		},
		{
			name:    "empty options",
			options: &CmdDecoderOptions{},
			wantErr: false,
		},
		{
			name: "custom options",
			options: &CmdDecoderOptions{
				Separator:     "_",
				ArrayFormat:   "_%d",
				AllowComments: false,
				AllowEmpty:    false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewCmdDecoderWithOptions(tt.options)
			if decoder == nil && !tt.wantErr {
				t.Errorf("NewCmdDecoderWithOptions() returned nil")
			}
		})
	}
}

func TestCmdDecoder_Decode(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "simple key-value pairs",
			data: `host=localhost
port=3306
debug=true`,
			want: map[string]interface{}{
				"host":  "localhost",
				"port":  int64(3306),
				"debug": true,
			},
			wantErr: false,
		},
		{
			name: "compound structure keys",
			data: `redis-password=123456
database-url=postgres://localhost:5432/db
jwt-secret=mysecret
api-timeout=30s`,
			want: map[string]interface{}{
				"redis-password": int64(123456), // 数字会被解析为 int64
				"database-url":   "postgres://localhost:5432/db",
				"jwt-secret":     "mysecret",
				"api-timeout":    "30s",
			},
			wantErr: false,
		},
		{
			name: "nested compound keys",
			data: `server-http-port=8080
server-grpc-port=9090
database-mysql-master-host=db1.example.com
redis-cluster-node-1-addr=redis1:6379`,
			want: map[string]interface{}{
				"server-http-port":           int64(8080),
				"server-grpc-port":           int64(9090),
				"database-mysql-master-host": "db1.example.com",
				"redis-cluster-node-1-addr":  "redis1:6379",
			},
			wantErr: false,
		},
		{
			name: "quoted values",
			data: `message="hello world"
key='secret key with spaces'
json="{\"test\":true}"`,
			want: map[string]interface{}{
				"message": "hello world",
				"key":     "secret key with spaces",
				"json":    "{\"test\":true}",
			},
			wantErr: false,
		},
		{
			name: "different value types",
			data: `string=hello
integer=42
float=3.14
boolean-true=true
boolean-false=false
empty-string=""`,
			want: map[string]interface{}{
				"string":        "hello",
				"integer":       int64(42),
				"float":         float64(3.14),
				"boolean-true":  true,
				"boolean-false": false,
				"empty-string":  "",
			},
			wantErr: false,
		},
		{
			name: "with comments and empty lines",
			data: `# This is a comment
host=localhost

// Another comment style
port=3306

debug=true`,
			want: map[string]interface{}{
				"host":  "localhost",
				"port":  int64(3306),
				"debug": true,
			},
			wantErr: false,
		},
		{
			name: "escaped values",
			data: `path="C:\\Program Files\\MyApp"
multiline="line1\nline2\ttab"
quotes="He said \"Hello\""`,
			want: map[string]interface{}{
				"path":      "C:\\Program Files\\MyApp",
				"multiline": "line1\nline2\ttab",
				"quotes":    "He said \"Hello\"",
			},
			wantErr: false,
		},
		{
			name: "kubernetes style config",
			data: `namespace=default
service-account=myapp
config-map-name=app-config
secret-tls-cert-path=/etc/ssl/certs/tls.crt
ingress-class-name=nginx`,
			want: map[string]interface{}{
				"namespace":             "default",
				"service-account":       "myapp",
				"config-map-name":       "app-config",
				"secret-tls-cert-path":  "/etc/ssl/certs/tls.crt",
				"ingress-class-name":    "nginx",
			},
			wantErr: false,
		},
		{
			name:    "invalid format - missing equals",
			data:    `host localhost`,
			wantErr: true,
		},
		{
			name:    "invalid format - empty key",
			data:    `=value`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewCmdDecoder()
			storage, err := decoder.Decode([]byte(tt.data))

			if (err != nil) != tt.wantErr {
				t.Errorf("CmdDecoder.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// 验证存储类型
			if storage == nil {
				t.Errorf("CmdDecoder.Decode() returned nil storage")
				return
			}

			// 转换为 map 进行比较
			var got map[string]interface{}
			if err := storage.ConvertTo(&got); err != nil {
				t.Errorf("Failed to convert storage to map: %v", err)
				return
			}

			// 比较结果
			for key, expectedValue := range tt.want {
				if gotValue, exists := got[key]; !exists {
					t.Errorf("CmdDecoder.Decode() missing key %s", key)
				} else if gotValue != expectedValue {
					t.Errorf("CmdDecoder.Decode() key %s = %v, want %v", key, gotValue, expectedValue)
				}
			}

			// 检查是否有多余的键
			for key := range got {
				if _, expected := tt.want[key]; !expected {
					t.Errorf("CmdDecoder.Decode() unexpected key %s = %v", key, got[key])
				}
			}
		})
	}
}

func TestCmdDecoder_Encode(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name: "simple values",
			input: map[string]interface{}{
				"host":  "localhost",
				"port":  3306,
				"debug": true,
			},
			want: `debug=true
host=localhost
port=3306`,
			wantErr: false,
		},
		{
			name: "compound keys",
			input: map[string]interface{}{
				"redis-password": "123456",
				"database-url":   "postgres://localhost:5432/db",
				"api-timeout":    "30s",
			},
			want: `api-timeout=30s
database-url=postgres://localhost:5432/db
redis-password=123456`,
			wantErr: false,
		},
		{
			name: "values needing quotes",
			input: map[string]interface{}{
				"message": "hello world",
				"path":    "C:\\Program Files\\MyApp",
				"empty":   "",
			},
			want: `empty=""
message="hello world"
path="C:\\Program Files\\MyApp"`,
			wantErr: false,
		},
		{
			name: "different types",
			input: map[string]interface{}{
				"string":  "hello",
				"int":     42,
				"int64":   int64(123),
				"float64": 3.14,
				"bool":    true,
			},
			want: `bool=true
float64=3.14
int=42
int64=123
string=hello`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewCmdDecoder()
			
			// 创建 FlatStorage
			storage := storage.NewFlatStorageWithOptions(tt.input, decoder.Separator, decoder.ArrayFormat)
			
			data, err := decoder.Encode(storage)
			if (err != nil) != tt.wantErr {
				t.Errorf("CmdDecoder.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got := string(data)
			if got != tt.want {
				t.Errorf("CmdDecoder.Encode() got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestCmdDecoder_parseValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  interface{}
	}{
		{"string", "hello", "hello"},
		{"quoted string", `"hello world"`, "hello world"},
		{"single quoted", "'test value'", "test value"},
		{"boolean true", "true", true},
		{"boolean false", "false", false},
		{"integer", "42", int64(42)},
		{"negative integer", "-123", int64(-123)},
		{"float", "3.14", float64(3.14)},
		{"negative float", "-2.5", float64(-2.5)},
		{"empty string", "", ""},
		{"quoted empty", `""`, ""},
		{"escaped string", `"line1\nline2"`, "line1\nline2"},
		{"url", "https://example.com:8080/path", "https://example.com:8080/path"},
	}

	decoder := NewCmdDecoder()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decoder.parseValue(tt.value)
			if err != nil {
				t.Errorf("CmdDecoder.parseValue() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("CmdDecoder.parseValue() = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestCmdDecoder_needsQuoting(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"simple string", "hello", false},
		{"string with space", "hello world", true},
		{"string with quotes", `say "hello"`, true},
		{"string with equals", "key=value", true},
		{"string with hash", "value#comment", true},
		{"empty string", "", true},
		{"path", "/etc/ssl/cert.pem", false},
		{"url", "https://example.com", false},
		{"string with newline", "line1\nline2", true},
	}

	decoder := NewCmdDecoder()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decoder.needsQuoting(tt.value)
			if got != tt.want {
				t.Errorf("CmdDecoder.needsQuoting(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestCmdDecoder_ConvertToStruct(t *testing.T) {
	// 定义测试用的配置结构体
	type ServerConfig struct {
		Http struct {
			Port int    `cfg:"port"`
			Host string `cfg:"host"`
		} `cfg:"http"`
		Grpc struct {
			Port int `cfg:"port"`
		} `cfg:"grpc"`
	}

	type DatabaseConfig struct {
		Mysql struct {
			Master struct {
				Host string `cfg:"host"`
				Port int    `cfg:"port"`
			} `cfg:"master"`
			Slave struct {
				Host string `cfg:"host"`
				Port int    `cfg:"port"`
			} `cfg:"slave"`
		} `cfg:"mysql"`
		Url string `cfg:"url"`
	}

	type AppConfig struct {
		Name     string         `cfg:"name"`
		Debug    bool           `cfg:"debug"`
		Server   ServerConfig   `cfg:"server"`
		Database DatabaseConfig `cfg:"database"`
		Redis    struct {
			Password string `cfg:"password"`
		} `cfg:"redis"`
		Features struct {
			NewUI struct {
				Enabled bool `cfg:"enabled"`
			} `cfg:"new-ui"`
		} `cfg:"features"`
	}

	// 测试数据 - 使用命令行参数风格的键名
	data := `name=MyApp
debug=true
server-http-port=8080
server-http-host=localhost
server-grpc-port=9090
database-mysql-master-host=db1.example.com
database-mysql-master-port=3306
database-mysql-slave-host=db2.example.com
database-mysql-slave-port=3306
database-url=postgres://localhost:5432/mydb
redis-password="123456"
features-new-ui-enabled=true`

	decoder := NewCmdDecoder()
	storage, err := decoder.Decode([]byte(data))
	if err != nil {
		t.Fatalf("CmdDecoder.Decode() error = %v", err)
	}

	// 转换为结构体
	var config AppConfig
	err = storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("storage.ConvertTo() error = %v", err)
	}

	// 验证转换结果
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"app name", config.Name, "MyApp"},
		{"debug flag", config.Debug, true},
		{"server http port", config.Server.Http.Port, 8080},
		{"server http host", config.Server.Http.Host, "localhost"},
		{"server grpc port", config.Server.Grpc.Port, 9090},
		{"database mysql master host", config.Database.Mysql.Master.Host, "db1.example.com"},
		{"database mysql master port", config.Database.Mysql.Master.Port, 3306},
		{"database mysql slave host", config.Database.Mysql.Slave.Host, "db2.example.com"},
		{"database mysql slave port", config.Database.Mysql.Slave.Port, 3306},
		{"database url", config.Database.Url, "postgres://localhost:5432/mydb"},
		{"redis password", config.Redis.Password, "123456"}, // 数字字符串会被解析为数字，然后转换回字符串
		{"features new ui enabled", config.Features.NewUI.Enabled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("ConvertTo() %s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestCmdDecoder_ConvertToSimpleStruct(t *testing.T) {
	// 简单结构体测试
	type SimpleConfig struct {
		AppName        string `cfg:"app-name"`
		MaxConnections int    `cfg:"max-connections"`
		EnableLogging  bool   `cfg:"enable-logging"`
		Timeout        float64 `cfg:"timeout"`
		EmptyValue     string  `cfg:"empty-value"`
	}

	data := `app-name=MyService
max-connections=100
enable-logging=true
timeout=30.5
empty-value=""`

	decoder := NewCmdDecoder()
	storage, err := decoder.Decode([]byte(data))
	if err != nil {
		t.Fatalf("CmdDecoder.Decode() error = %v", err)
	}

	var config SimpleConfig
	err = storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("storage.ConvertTo() error = %v", err)
	}

	// 验证结果
	if config.AppName != "MyService" {
		t.Errorf("AppName = %v, want MyService", config.AppName)
	}
	if config.MaxConnections != 100 {
		t.Errorf("MaxConnections = %v, want 100", config.MaxConnections)
	}
	if config.EnableLogging != true {
		t.Errorf("EnableLogging = %v, want true", config.EnableLogging)
	}
	if config.Timeout != 30.5 {
		t.Errorf("Timeout = %v, want 30.5", config.Timeout)
	}
	if config.EmptyValue != "" {
		t.Errorf("EmptyValue = %v, want empty string", config.EmptyValue)
	}
}