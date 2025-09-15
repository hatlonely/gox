package cfg

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// 测试用的复杂配置结构体
type DatabasePool struct {
	Name     string        `cfg:"name" help:"连接池名称" eg:"main-pool"`
	Host     string        `cfg:"host" help:"数据库主机地址" eg:"localhost" def:"127.0.0.1"`
	Port     int           `cfg:"port" help:"数据库端口号" eg:"3306" def:"5432"`
	MaxConns int           `cfg:"max_conns" help:"最大连接数" eg:"100" def:"10"`
	Timeout  time.Duration `cfg:"timeout" help:"连接超时时间" eg:"30s" def:"10s"`
}

type ServerConfig struct {
	Host    string        `cfg:"host" help:"服务器绑定地址" eg:"0.0.0.0" def:"localhost"`
	Port    int           `cfg:"port" help:"服务器监听端口" eg:"8080" def:"80"`
	Timeout time.Duration `cfg:"timeout" help:"请求超时时间" eg:"60s" def:"30s"`
}

type ComplexConfig struct {
	// 基本类型
	Name    string `cfg:"name" help:"应用名称" eg:"my-app" def:"app"`
	Version string `cfg:"version" help:"应用版本号" eg:"1.2.3" def:"1.0.0"`
	Debug   bool   `cfg:"debug" help:"是否启用调试模式" eg:"true" def:"false"`

	// 嵌套结构体
	Server ServerConfig `cfg:"server" help:"服务器配置"`

	// 指针结构体
	Database *DatabaseConfig `cfg:"database" help:"数据库配置"`

	// 切片类型
	Pools []DatabasePool `cfg:"pools" help:"数据库连接池列表"`

	// Map类型
	Cache    map[string]string       `cfg:"cache" help:"缓存配置映射" eg:"redis=localhost:6379,memcache=localhost:11211"`
	Features map[string]bool         `cfg:"features" help:"功能开关映射" eg:"feature1=true,feature2=false"`
	Services map[string]ServerConfig `cfg:"services" help:"服务配置映射"`

	// 时间类型
	StartTime time.Time     `cfg:"start_time" help:"服务启动时间" eg:"2023-12-25T15:30:45Z"`
	Interval  time.Duration `cfg:"interval" help:"执行间隔" eg:"5m" def:"1m"`
}

type DatabaseConfig struct {
	Host     string `cfg:"host" help:"数据库主机" eg:"db.example.com" def:"localhost"`
	Port     int    `cfg:"port" help:"数据库端口" eg:"5432" def:"5432"`
	Username string `cfg:"username" help:"数据库用户名" eg:"admin" def:"postgres"`
	Password string `cfg:"password" help:"数据库密码" eg:"secret123"`
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

	// 验证切片类型信息：现在只展示叶子节点字段，使用占位符格式
	expectedContent := []string{
		"pools[N].host",
		"pools[N].port",
		"pools[N].name",
		"pools[N].max_conns",
		"pools[N].timeout",
		"数据库主机地址",
		"数据库端口号",
		"连接池名称",
		"最大连接数",
		"连接超时时间",
		"APP_POOLS_{N}_HOST", // 新的占位符格式
		"APP_POOLS_{N}_PORT",
		"--app-pools-{n}-host", // 命令行参数使用小写
		"--app-pools-{n}-port",
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

	// 验证Map类型信息：现在只展示基本类型 map 的叶子节点
	expectedContent := []string{
		"cache (map[string]string)",
		"缓存配置",
		"APP_CACHE",
		"--app-cache",
		"features (map[string]bool)",
		"功能开关",
		"APP_FEATURES",
		"--app-features",
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

func TestGenerateHelp_PrintComplexConfig(t *testing.T) {
	config := ComplexConfig{}
	help := GenerateHelp(&config, "APP_", "app-")

	// 打印完整的帮助信息用于查看效果
	fmt.Println("=== ComplexConfig 完整帮助信息 ===")
	fmt.Println(help)
	fmt.Println("=== 帮助信息结束 ===")

	// 基本验证
	if !strings.Contains(help, "配置参数说明") {
		t.Error("帮助信息应包含标题")
	}

	// 验证数组类型占位符
	if !strings.Contains(help, "pools[N]") {
		t.Error("帮助信息应包含数组占位符 pools[N]")
	}

	// 验证Map类型占位符
	if !strings.Contains(help, "services.{KEY}") {
		t.Error("帮助信息应包含Map占位符 services.{KEY}")
	}

	// 验证所有复杂类型都有对应的字段信息
	expectedFields := []string{
		"pools[N].host",
		"pools[N].port",
		"pools[N].name",
		"pools[N].timeout",
		"services.{KEY}.host",
		"services.{KEY}.port",
		"services.{KEY}.timeout",
		"database.host",
		"database.port",
		"server.host",
		"server.port",
	}

	for _, field := range expectedFields {
		if !strings.Contains(help, field) {
			t.Errorf("帮助信息应包含字段: %s", field)
		}
	}
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

func TestGenerateHelp_TagSupport(t *testing.T) {
	type Config struct {
		Host    string `cfg:"host" help:"服务器地址" eg:"localhost" def:"127.0.0.1"`
		Port    int    `cfg:"port" help:"端口号" eg:"8080" def:"80"`
		Debug   bool   `cfg:"debug" help:"调试模式" def:"false"`
		Timeout string `cfg:"timeout" help:"超时时间"` // 没有 eg 和 def 标签
	}

	config := Config{}
	help := GenerateHelp(&config, "APP_", "app-")

	// 验证 eg 标签的示例值
	if !strings.Contains(help, "示例: localhost") {
		t.Error("应该使用 eg 标签的示例值")
	}
	if !strings.Contains(help, "示例: 8080") {
		t.Error("应该使用 eg 标签的示例值")
	}

	// 验证 def 标签的默认值
	if !strings.Contains(help, "默认值: 127.0.0.1") {
		t.Error("应该使用 def 标签的默认值")
	}
	if !strings.Contains(help, "默认值: 80") {
		t.Error("应该使用 def 标签的默认值")
	}
	if !strings.Contains(help, "默认值: false") {
		t.Error("应该使用 def 标签的默认值")
	}

	// 验证没有 eg 标签时不显示示例值
	// timeout 字段没有 eg 标签，不应该有示例值信息
	timeoutSection := extractFieldSection(help, "timeout")
	if strings.Contains(timeoutSection, "示例:") {
		t.Error("timeout 字段没有 eg 标签，不应该显示示例值")
	}
	// timeout 字段也没有 def 标签，不应该有默认值信息
	if strings.Contains(timeoutSection, "默认值:") {
		t.Error("timeout 字段没有 def 标签，不应该显示默认值")
	}

	t.Logf("帮助信息:\n%s", help)
}

// extractFieldSection 提取指定字段的帮助信息部分
func extractFieldSection(help, fieldName string) string {
	lines := strings.Split(help, "\n")
	var fieldLines []string
	inField := false

	for _, line := range lines {
		if strings.Contains(line, fieldName+" (") {
			inField = true
			fieldLines = append(fieldLines, line)
			continue
		}

		if inField {
			if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
				// 新的字段开始
				break
			}
			fieldLines = append(fieldLines, line)
		}
	}

	return strings.Join(fieldLines, "\n")
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
