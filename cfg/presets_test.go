package cfg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	// 创建临时配置文件
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 JSON 配置文件
	configFile := filepath.Join(tmpDir, "config.json")
	configContent := `{
		"database": {
			"host": "localhost",
			"port": 5432,
			"name": "testdb"
		},
		"server": {
			"host": "0.0.0.0",
			"port": 8080
		},
		"debug": false
	}`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 设置环境变量（会覆盖文件中的配置）
	os.Setenv("DATABASE_HOST", "env-host")
	os.Setenv("DEBUG", "true")
	defer func() {
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("DEBUG")
	}()

	// 模拟命令行参数（会覆盖环境变量和文件配置）
	originalArgs := os.Args
	os.Args = []string{"program", "--database-port=3306", "--server-host=cmd-host"}
	defer func() {
		os.Args = originalArgs
	}()

	// 创建配置对象
	cfg, err := NewConfig(configFile)
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()

	// 测试配置合并结果
	var config struct {
		Database struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
			Name string `cfg:"name"`
		} `cfg:"database"`
		Server struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
		} `cfg:"server"`
		Debug bool `cfg:"debug"`
	}

	if err := cfg.ConvertTo(&config); err != nil {
		t.Fatal(err)
	}

	// 验证配置优先级
	// 文件中 database.host = "localhost"，环境变量覆盖为 "env-host"
	if config.Database.Host != "env-host" {
		t.Errorf("expected database.host=env-host, got %s", config.Database.Host)
	}

	// 文件中 database.port = 5432，命令行覆盖为 3306
	if config.Database.Port != 3306 {
		t.Errorf("expected database.port=3306, got %d", config.Database.Port)
	}

	// 文件中 database.name = "testdb"，没有被覆盖
	if config.Database.Name != "testdb" {
		t.Errorf("expected database.name=testdb, got %s", config.Database.Name)
	}

	// 文件中 server.host = "0.0.0.0"，命令行覆盖为 "cmd-host"
	if config.Server.Host != "cmd-host" {
		t.Errorf("expected server.host=cmd-host, got %s", config.Server.Host)
	}

	// 文件中 server.port = 8080，没有被覆盖
	if config.Server.Port != 8080 {
		t.Errorf("expected server.port=8080, got %d", config.Server.Port)
	}

	// 文件中 debug = false，环境变量覆盖为 true
	if !config.Debug {
		t.Errorf("expected debug=true, got %v", config.Debug)
	}
}

func TestNewConfigWithPrefix(t *testing.T) {
	// 创建临时配置文件
	tmpDir, err := os.MkdirTemp("", "config_prefix_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 YAML 配置文件
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `
database:
  host: localhost
  port: 5432
server:
  port: 8080
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 设置环境变量（包括带前缀和不带前缀的）
	os.Setenv("APP_DATABASE_HOST", "app-env-host")
	os.Setenv("OTHER_HOST", "other-host") // 这个不会被处理
	defer func() {
		os.Unsetenv("APP_DATABASE_HOST")
		os.Unsetenv("OTHER_HOST")
	}()

	// 模拟命令行参数（包括带前缀和不带前缀的）
	originalArgs := os.Args
	os.Args = []string{"program", "--app-server-port=9090", "--other-port=1111"}
	defer func() {
		os.Args = originalArgs
	}()

	// 创建配置对象（使用前缀）
	cfg, err := NewConfigWithPrefix(configFile, "APP_", "app-")
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()

	// 测试配置结果
	var config struct {
		Database struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
		} `cfg:"database"`
		Server struct {
			Port int `cfg:"port"`
		} `cfg:"server"`
	}

	if err := cfg.ConvertTo(&config); err != nil {
		t.Fatal(err)
	}

	// 验证前缀过滤和优先级

	// 环境变量 APP_DATABASE_HOST=app-env-host 会覆盖文件中的 database.host=localhost
	if config.Database.Host != "app-env-host" {
		t.Errorf("expected database.host=app-env-host, got %s", config.Database.Host)
	}

	// 命令行 --app-server-port=9090 会覆盖文件中的 server.port=8080
	if config.Server.Port != 9090 {
		t.Errorf("expected server.port=9090, got %d", config.Server.Port)
	}

	// 文件中的 database.port=5432 不会被 --other-port=1111 影响（前缀不匹配）
	if config.Database.Port != 5432 {
		t.Errorf("expected database.port=5432, got %d", config.Database.Port)
	}
}

func TestNewConfigSupportedFormats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_formats_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testCases := []struct {
		filename string
		content  string
	}{
		{
			"config.json",
			`{"server": {"port": 8080}}`,
		},
		{
			"config.yaml",
			`server:
  port: 8080`,
		},
		{
			"config.toml",
			`[server]
port = 8080`,
		},
		{
			"config.ini",
			`[server]
port = 8080`,
		},
		{
			"config.env",
			`SERVER_PORT=8080`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			configFile := filepath.Join(tmpDir, tc.filename)
			if err := os.WriteFile(configFile, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := NewConfig(configFile)
			if err != nil {
				t.Fatal(err)
			}
			defer cfg.Close()

			var config struct {
				Server struct {
					Port int `cfg:"port"`
				} `cfg:"server"`
			}

			if err := cfg.ConvertTo(&config); err != nil {
				t.Fatal(err)
			}

			if config.Server.Port != 8080 {
				t.Errorf("expected server.port=8080, got %d", config.Server.Port)
			}
		})
	}
}

func TestNewConfigFileNotExists(t *testing.T) {
	cfg, err := NewConfig("nonexistent.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
		cfg.Close()
	}
}

func TestNewConfigEmptyFilename(t *testing.T) {
	cfg, err := NewConfig("")
	if err == nil {
		t.Error("expected error for empty filename, got nil")
		cfg.Close()
	}
}

func TestNewConfigUnsupportedFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_unsupported_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	unsupportedFile := filepath.Join(tmpDir, "config.txt")
	if err := os.WriteFile(unsupportedFile, []byte("some content"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := NewConfig(unsupportedFile)
	if err == nil {
		t.Error("expected error for unsupported file format, got nil")
		cfg.Close()
	}
}
