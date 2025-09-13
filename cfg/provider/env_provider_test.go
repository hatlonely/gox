package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewEnvProviderWithOptions(t *testing.T) {
	tests := []struct {
		name    string
		options *EnvProviderOptions
		wantErr bool
	}{
		{
			name:    "nil options",
			options: nil,
			wantErr: false,
		},
		{
			name:    "empty options",
			options: &EnvProviderOptions{},
			wantErr: false,
		},
		{
			name: "valid env files",
			options: &EnvProviderOptions{
				EnvFiles: []string{"test1.env", "test2.env"},
			},
			wantErr: false,
		},
		{
			name: "mixed valid and empty files",
			options: &EnvProviderOptions{
				EnvFiles: []string{"test.env", "", "test2.env"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewEnvProviderWithOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEnvProviderWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Errorf("NewEnvProviderWithOptions() returned nil provider")
			}
		})
	}
}

func TestEnvProvider_Load(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "env_provider_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试 .env 文件
	env1File := filepath.Join(tempDir, "test1.env")
	env1Content := `# Test env file 1
APP_NAME=TestApp
DB_HOST=localhost
DB_PORT=3306
DEBUG=true`

	if err := os.WriteFile(env1File, []byte(env1Content), 0644); err != nil {
		t.Fatalf("Failed to write env1 file: %v", err)
	}

	env2File := filepath.Join(tempDir, "test2.env")
	env2Content := `# Test env file 2
DB_PORT=5432
DB_USER=admin
API_KEY="secret key with spaces"`

	if err := os.WriteFile(env2File, []byte(env2Content), 0644); err != nil {
		t.Fatalf("Failed to write env2 file: %v", err)
	}

	tests := []struct {
		name     string
		envFiles []string
		envVars  map[string]string
		want     map[string]string
	}{
		{
			name:     "no files, only system env",
			envFiles: []string{},
			envVars:  map[string]string{"TEST_VAR": "test_value"},
			want:     map[string]string{"TEST_VAR": "test_value"},
		},
		{
			name:     "single env file",
			envFiles: []string{env1File},
			envVars:  map[string]string{"EXISTING_VAR": "existing"},
			want: map[string]string{
				"EXISTING_VAR": "existing",
				"APP_NAME":     "TestApp",
				"DB_HOST":      "localhost",
				"DB_PORT":      "3306",
				"DEBUG":        "true",
			},
		},
		{
			name:     "multiple env files with priority",
			envFiles: []string{env1File, env2File},
			envVars:  map[string]string{"SYSTEM_VAR": "system"},
			want: map[string]string{
				"SYSTEM_VAR": "system",
				"APP_NAME":   "TestApp",
				"DB_HOST":    "localhost",
				"DB_PORT":    "5432", // env2 覆盖 env1
				"DEBUG":      "true",
				"DB_USER":    "admin",
				"API_KEY":    `"secret key with spaces"`, // 保持原始格式
			},
		},
		{
			name:     "nonexistent file should not cause error",
			envFiles: []string{env1File, "/nonexistent/file.env", env2File},
			envVars:  map[string]string{},
			want: map[string]string{
				"APP_NAME": "TestApp",
				"DB_HOST":  "localhost",
				"DB_PORT":  "5432", // env2 覆盖 env1
				"DEBUG":    "true",
				"DB_USER":  "admin",
				"API_KEY":  `"secret key with spaces"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试环境变量
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			provider, err := NewEnvProviderWithOptions(&EnvProviderOptions{
				EnvFiles: tt.envFiles,
			})
			if err != nil {
				t.Fatalf("NewEnvProviderWithOptions() error = %v", err)
			}

			data, err := provider.Load()
			if err != nil {
				t.Errorf("EnvProvider.Load() error = %v", err)
				return
			}

			// 解析返回的 .env 格式数据
			got := parseEnvData(string(data))

			// 检查期望的键值对
			for key, expectedValue := range tt.want {
				if gotValue, exists := got[key]; !exists {
					t.Errorf("EnvProvider.Load() missing key %s", key)
				} else if gotValue != expectedValue {
					t.Errorf("EnvProvider.Load() key %s = %v, want %v", key, gotValue, expectedValue)
				}
			}

			// 检查是否有多余的键
			for key := range got {
				if _, expected := tt.want[key]; !expected {
					// 允许系统环境变量存在
					if !strings.HasPrefix(key, "TEST_") && !strings.HasPrefix(key, "EXISTING_") && !strings.HasPrefix(key, "SYSTEM_") {
						continue
					}
					t.Errorf("EnvProvider.Load() unexpected key %s = %v", key, got[key])
				}
			}
		})
	}
}

func TestEnvProvider_Save(t *testing.T) {
	provider, err := NewEnvProviderWithOptions(nil)
	if err != nil {
		t.Fatalf("NewEnvProviderWithOptions() error = %v", err)
	}

	err = provider.Save([]byte("test data"))
	if err == nil {
		t.Errorf("EnvProvider.Save() expected error, got nil")
	}

	// 检查错误类型
	if providerErr, ok := err.(*ProviderError); !ok {
		t.Errorf("EnvProvider.Save() error type = %T, want *ProviderError", err)
	} else if !strings.Contains(providerErr.Error(), "does not support save") {
		t.Errorf("EnvProvider.Save() error message = %v, want contains 'does not support save'", providerErr.Error())
	}
}

func TestEnvProvider_OnChange(t *testing.T) {
	provider, err := NewEnvProviderWithOptions(nil)
	if err != nil {
		t.Fatalf("NewEnvProviderWithOptions() error = %v", err)
	}

	// OnChange 应该直接返回，不做任何操作
	called := false
	provider.OnChange(func(data []byte) error {
		called = true
		return nil
	})

	// 验证回调没有被调用（因为不支持变更监听）
	if called {
		t.Errorf("EnvProvider.OnChange() callback was called, expected no call")
	}
}

func TestEnvProvider_Close(t *testing.T) {
	provider, err := NewEnvProviderWithOptions(nil)
	if err != nil {
		t.Fatalf("NewEnvProviderWithOptions() error = %v", err)
	}

	err = provider.Close()
	if err != nil {
		t.Errorf("EnvProvider.Close() error = %v, want nil", err)
	}
}

func TestEnvProvider_loadEnvFile(t *testing.T) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "test.env")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := `# Comment line
APP_NAME=TestApp
DB_HOST=localhost

# Another comment
DB_PORT=3306
EMPTY_VALUE=
QUOTED_VALUE="hello world"
SINGLE_QUOTED='test value'
// Another comment style
SPECIAL_CHARS=value with spaces and "quotes"`

	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tempFile.Close()

	provider := &EnvProvider{}
	envVars := make(map[string]string)

	err = provider.loadEnvFile(tempFile.Name(), envVars)
	if err != nil {
		t.Errorf("loadEnvFile() error = %v", err)
	}

	expected := map[string]string{
		"APP_NAME":      "TestApp",
		"DB_HOST":       "localhost",
		"DB_PORT":       "3306",
		"EMPTY_VALUE":   "",
		"QUOTED_VALUE":  `"hello world"`, // 保持原始引号
		"SINGLE_QUOTED": "'test value'",  // 保持原始引号
		"SPECIAL_CHARS": `value with spaces and "quotes"`,
	}

	for key, expectedValue := range expected {
		if gotValue, exists := envVars[key]; !exists {
			t.Errorf("loadEnvFile() missing key %s", key)
		} else if gotValue != expectedValue {
			t.Errorf("loadEnvFile() key %s = %v, want %v", key, gotValue, expectedValue)
		}
	}
}

// parseEnvData 解析 .env 格式数据为 map
func parseEnvData(data string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		equalIndex := strings.Index(line, "=")
		if equalIndex == -1 {
			continue
		}

		key := strings.TrimSpace(line[:equalIndex])
		value := line[equalIndex+1:]

		if key != "" {
			result[key] = value
		}
	}

	return result
}