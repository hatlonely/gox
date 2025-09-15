package cfg

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// 测试用的复杂配置结构体
type DatabasePool struct {
	Name     string        `cfg:"name" help:"连接池名称"`
	Host     string        `cfg:"host" help:"数据库主机地址"`
	Port     int           `cfg:"port" help:"数据库端口号"`
	MaxConns int           `cfg:"max_conns" help:"最大连接数"`
	Timeout  time.Duration `cfg:"timeout" help:"连接超时时间"`
}

type ServerConfig struct {
	Host    string        `cfg:"host" help:"服务器绑定地址"`
	Port    int           `cfg:"port" help:"服务器监听端口"`
	Timeout time.Duration `cfg:"timeout" help:"请求超时时间"`
}

type ComplexConfig struct {
	// 基本类型
	Name    string `cfg:"name" help:"应用名称"`
	Version string `cfg:"version" help:"应用版本号"`
	Debug   bool   `cfg:"debug" help:"是否启用调试模式"`

	// 嵌套结构体
	Server ServerConfig `cfg:"server" help:"服务器配置"`

	// 指针结构体
	Database *DatabaseConfig `cfg:"database" help:"数据库配置"`

	// 切片类型
	Pools []DatabasePool `cfg:"pools" help:"数据库连接池列表"`

	// Map类型
	Cache    map[string]string       `cfg:"cache" help:"缓存配置映射"`
	Features map[string]bool         `cfg:"features" help:"功能开关映射"`
	Services map[string]ServerConfig `cfg:"services" help:"服务配置映射"`

	// 时间类型
	StartTime time.Time     `cfg:"start_time" help:"服务启动时间"`
	Interval  time.Duration `cfg:"interval" help:"执行间隔"`
}

type DatabaseConfig struct {
	Host     string `cfg:"host" help:"数据库主机"`
	Port     int    `cfg:"port" help:"数据库端口"`
	Username string `cfg:"username" help:"数据库用户名"`
	Password string `cfg:"password" help:"数据库密码"`
}

func TestGenerateHelp_BasicTypes(t *testing.T) {
	type SimpleConfig struct {
		Name  string `cfg:"name" help:"应用名称"`
		Port  int    `cfg:"port" help:"监听端口"`
		Debug bool   `cfg:"debug" help:"调试模式"`
	}

	config := SimpleConfig{}
	help := GenerateHelp(&config, "APP_", "app-")

	// 验证基本结构
	if !strings.Contains(help, "配置参数说明") {
		t.Error("帮助信息应包含标题")
	}

	// 验证字段信息
	expectedContent := []string{
		"name (string)",
		"应用名称",
		"APP_NAME",
		"--app-name",
		"port (int)",
		"监听端口",
		"APP_PORT",
		"--app-port",
		"debug (bool)",
		"调试模式",
		"APP_DEBUG",
		"--app-debug",
	}

	for _, content := range expectedContent {
		if !strings.Contains(help, content) {
			t.Errorf("帮助信息应包含: %s\n实际输出:\n%s", content, help)
		}
	}
}

func TestGenerateHelp_NestedStruct(t *testing.T) {
	type Config struct {
		Database DatabaseConfig `cfg:"database" help:"数据库配置"`
	}

	config := Config{}
	help := GenerateHelp(&config, "APP_", "app-")

	// 验证嵌套字段
	expectedContent := []string{
		"database.host",
		"database.port",
		"database.username",
		"database.password",
		"APP_DATABASE_HOST",
		"APP_DATABASE_PORT",
		"--app-database-host",
		"--app-database-port",
	}

	for _, content := range expectedContent {
		if !strings.Contains(help, content) {
			t.Errorf("帮助信息应包含嵌套字段: %s\n实际输出:\n%s", content, help)
		}
	}
}

func TestGenerateHelp_SliceType(t *testing.T) {
	type Config struct {
		Pools []DatabasePool `cfg:"pools" help:"连接池列表"`
	}

	config := Config{}
	help := GenerateHelp(&config, "APP_", "app-")

	// 验证切片类型信息
	expectedContent := []string{
		"pools",
		"连接池列表",
		"数组类型",
		"APP_POOLS_0",
		"APP_POOLS_1",
		"--app-pools-0",
		"--app-pools-1",
		"数组元素配置格式",
	}

	for _, content := range expectedContent {
		if !strings.Contains(help, content) {
			t.Errorf("帮助信息应包含切片信息: %s\n实际输出:\n%s", content, help)
		}
	}
}

func TestGenerateHelp_MapType(t *testing.T) {
	type Config struct {
		Cache    map[string]string `cfg:"cache" help:"缓存配置"`
		Features map[string]bool   `cfg:"features" help:"功能开关"`
	}

	config := Config{}
	help := GenerateHelp(&config, "APP_", "app-")

	// 验证Map类型信息
	expectedContent := []string{
		"cache",
		"缓存配置",
		"映射类型",
		"键类型: string, 值类型: string",
		"APP_CACHE_REDIS", // 新的示例格式
		"--app-cache-redis",
		"features",
		"功能开关",
		"键类型: string, 值类型: bool",
		"映射配置格式",
	}

	for _, content := range expectedContent {
		if !strings.Contains(help, content) {
			t.Errorf("帮助信息应包含Map信息: %s\n实际输出:\n%s", content, help)
		}
	}
}

func TestGenerateHelp_TimeTypes(t *testing.T) {
	type Config struct {
		Timeout   time.Duration `cfg:"timeout" help:"超时时间"`
		CreatedAt time.Time     `cfg:"created_at" help:"创建时间"`
	}

	config := Config{}
	help := GenerateHelp(&config, "APP_", "app-")

	// 验证时间类型
	expectedContent := []string{
		"timeout (time.Duration)",
		"超时时间",
		"30s", "5m", "1h",
		"created_at (time.Time)",
		"创建时间",
		"2023-12-25T15:30:45Z",
		"1703517045",
	}

	for _, content := range expectedContent {
		if !strings.Contains(help, content) {
			t.Errorf("帮助信息应包含时间类型信息: %s\n实际输出:\n%s", content, help)
		}
	}
}

func TestGenerateHelp_PointerType(t *testing.T) {
	type Config struct {
		Database *DatabaseConfig `cfg:"database" help:"数据库配置"`
	}

	config := Config{}
	help := GenerateHelp(&config, "APP_", "app-")

	// 验证指针类型处理
	expectedContent := []string{
		"database.host",
		"database.port",
		"database.username",
		"database.password",
	}

	for _, content := range expectedContent {
		if !strings.Contains(help, content) {
			t.Errorf("帮助信息应包含指针字段: %s\n实际输出:\n%s", content, help)
		}
	}
}

func TestGenerateHelp_ComplexStructure(t *testing.T) {
	config := ComplexConfig{}
	help := GenerateHelp(&config, "APP_", "app-")

	// 验证复杂结构的完整性
	expectedSections := []string{
		"配置参数说明",
		"类型说明",
		"配置优先级",
		"数组配置示例",
		"映射配置示例",
	}

	for _, section := range expectedSections {
		if !strings.Contains(help, section) {
			t.Errorf("帮助信息应包含章节: %s", section)
		}
	}

	// 验证所有字段类型都被包含
	expectedTypes := []string{
		"string", "int", "bool",
		"time.Duration", "time.Time",
		"数组类型", "映射类型",
	}

	for _, typeStr := range expectedTypes {
		if !strings.Contains(help, typeStr) {
			t.Errorf("帮助信息应包含类型: %s", typeStr)
		}
	}

	// 打印完整的帮助信息用于手动验证
	t.Logf("完整帮助信息:\n%s", help)
}

func TestGenerateHelp_NoPrefix(t *testing.T) {
	type SimpleConfig struct {
		Name string `cfg:"name" help:"应用名称"`
	}

	config := SimpleConfig{}
	help := GenerateHelp(&config, "", "")

	// 验证无前缀的情况
	expectedContent := []string{
		"NAME",   // 环境变量无前缀
		"--name", // 命令行参数无前缀
	}

	for _, content := range expectedContent {
		if !strings.Contains(help, content) {
			t.Errorf("无前缀时帮助信息应包含: %s\n实际输出:\n%s", content, help)
		}
	}
}

func TestGenerateHelp_IgnoredFields(t *testing.T) {
	type Config struct {
		Name         string `cfg:"name" help:"应用名称"`
		InternalData string `cfg:"-"`  // 忽略字段
		Hidden       string `json:"-"` // 忽略字段
	}

	config := Config{}
	help := GenerateHelp(&config, "APP_", "app-")

	// 验证忽略字段不出现在帮助中
	if strings.Contains(help, "internal_data") || strings.Contains(help, "InternalData") {
		t.Error("被忽略的字段不应出现在帮助信息中")
	}

	if strings.Contains(help, "hidden") || strings.Contains(help, "Hidden") {
		t.Error("被忽略的字段不应出现在帮助信息中")
	}

	// 验证正常字段存在
	if !strings.Contains(help, "name") {
		t.Error("正常字段应出现在帮助信息中")
	}
}

func TestFieldConfigName_TagPriority(t *testing.T) {
	tests := []struct {
		name     string
		field    reflect.StructField
		expected string
	}{
		{
			name: "cfg标签优先级最高",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `cfg:"cfg_name" json:"json_name" yaml:"yaml_name"`,
			},
			expected: "cfg_name",
		},
		{
			name: "json标签次高",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `json:"json_name" yaml:"yaml_name" toml:"toml_name"`,
			},
			expected: "json_name",
		},
		{
			name: "yaml标签第三",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `yaml:"yaml_name" toml:"toml_name" ini:"ini_name"`,
			},
			expected: "yaml_name",
		},
		{
			name: "使用字段名",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  ``,
			},
			expected: "TestField",
		},
		{
			name: "忽略字段",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `cfg:"-"`,
			},
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFieldConfigName(tt.field)
			if result != tt.expected {
				t.Errorf("getFieldConfigName() = %v, want %v", result, tt.expected)
			}
		})
	}
}
