package provider

import (
	"strings"
	"testing"
)

func TestNewCmdProviderWithOptions(t *testing.T) {
	tests := []struct {
		name    string
		options *CmdProviderOptions
		wantErr bool
	}{
		{
			name:    "nil options",
			options: nil,
			wantErr: false,
		},
		{
			name:    "empty options",
			options: &CmdProviderOptions{},
			wantErr: false,
		},
		{
			name: "with prefix",
			options: &CmdProviderOptions{
				Prefix: "app-",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewCmdProviderWithOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCmdProviderWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Errorf("NewCmdProviderWithOptions() returned nil provider")
			}
		})
	}
}

func TestCmdProvider_Load(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		testArgs []string
		want     map[string]string
	}{
		{
			name:     "no arguments",
			prefix:   "",
			testArgs: []string{},
			want:     map[string]string{},
		},
		{
			name:     "simple key=value",
			prefix:   "",
			testArgs: []string{"--host=localhost", "--port=3306"},
			want: map[string]string{
				"host": "localhost",
				"port": "3306",
			},
		},
		{
			name:     "key value format",
			prefix:   "",
			testArgs: []string{"--host", "localhost", "--port", "3306"},
			want: map[string]string{
				"host": "localhost",
				"port": "3306",
			},
		},
		{
			name:     "boolean flags",
			prefix:   "",
			testArgs: []string{"--debug", "--verbose"},
			want: map[string]string{
				"debug":   "true",
				"verbose": "true",
			},
		},
		{
			name:     "mixed formats",
			prefix:   "",
			testArgs: []string{"--host=localhost", "--port", "3306", "--debug", "--name", "test app"},
			want: map[string]string{
				"host":  "localhost",
				"port":  "3306",
				"debug": "true",
				"name":  "test app",
			},
		},
		{
			name:     "with prefix filter",
			prefix:   "app-",
			testArgs: []string{"--app-host=localhost", "--app-port=3306", "--debug", "--other=ignored"},
			want: map[string]string{
				"host": "localhost",
				"port": "3306",
			},
		},
		{
			name:     "values with spaces and special chars",
			prefix:   "",
			testArgs: []string{"--message=hello world", "--key", "secret key with spaces", "--json={\"test\":true}"},
			want: map[string]string{
				"message": "hello world",
				"key":     "secret key with spaces",
				"json":    "{\"test\":true}",
			},
		},
		{
			name:     "compound structure keys",
			prefix:   "",
			testArgs: []string{"--redis-password", "123456", "--database-url=postgres://localhost:5432/db", "--jwt-secret=mysecret", "--api-timeout", "30s", "--cache-redis-addr=localhost:6379"},
			want: map[string]string{
				"redis-password":   "123456",
				"database-url":     "postgres://localhost:5432/db",
				"jwt-secret":       "mysecret",
				"api-timeout":      "30s",
				"cache-redis-addr": "localhost:6379",
			},
		},
		{
			name:     "nested compound keys with prefix",
			prefix:   "app-",
			testArgs: []string{"--app-redis-host=localhost", "--app-redis-port", "6379", "--app-db-mysql-host=127.0.0.1", "--other-key=ignored", "--app-log-level=debug"},
			want: map[string]string{
				"redis-host":    "localhost",
				"redis-port":    "6379",
				"db-mysql-host": "127.0.0.1",
				"log-level":     "debug",
			},
		},
		{
			name:     "ignore non-long options",
			prefix:   "",
			testArgs: []string{"-h", "help", "--host=localhost", "ignored", "--port", "3306"},
			want: map[string]string{
				"host": "localhost",
				"port": "3306",
			},
		},
		{
			name:     "empty values",
			prefix:   "",
			testArgs: []string{"--empty=", "--host", ""},
			want: map[string]string{
				"empty": "",
				"host":  "",
			},
		},
		{
			name:   "complex nested configuration keys",
			prefix: "",
			testArgs: []string{
				"--server-http-port=8080",
				"--server-grpc-port", "9090",
				"--database-mysql-master-host=db1.example.com",
				"--database-mysql-slave-host", "db2.example.com",
				"--redis-cluster-node-1-addr=redis1:6379",
				"--redis-cluster-node-2-addr=redis2:6379",
				"--oauth2-google-client-id=12345",
				"--feature-flag-new-ui-enabled",
			},
			want: map[string]string{
				"server-http-port":            "8080",
				"server-grpc-port":            "9090",
				"database-mysql-master-host":  "db1.example.com",
				"database-mysql-slave-host":   "db2.example.com",
				"redis-cluster-node-1-addr":   "redis1:6379",
				"redis-cluster-node-2-addr":   "redis2:6379",
				"oauth2-google-client-id":     "12345",
				"feature-flag-new-ui-enabled": "true",
			},
		},
		{
			name:   "kubernetes style configuration",
			prefix: "k8s-",
			testArgs: []string{
				"--k8s-namespace=default",
				"--k8s-service-account", "myapp",
				"--k8s-config-map-name=app-config",
				"--k8s-secret-tls-cert-path=/etc/ssl/certs/tls.crt",
				"--other-flag=ignored",
				"--k8s-ingress-class-name", "nginx",
			},
			want: map[string]string{
				"namespace":            "default",
				"service-account":      "myapp",
				"config-map-name":      "app-config",
				"secret-tls-cert-path": "/etc/ssl/certs/tls.crt",
				"ingress-class-name":   "nginx",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &CmdProvider{
				prefix:   tt.prefix,
				testArgs: tt.testArgs,
			}

			data, err := provider.Load()
			if err != nil {
				t.Errorf("CmdProvider.Load() error = %v", err)
				return
			}

			// 解析返回的 .env 格式数据
			got := parseCmdData(string(data))

			// 检查期望的键值对
			for key, expectedValue := range tt.want {
				if gotValue, exists := got[key]; !exists {
					t.Errorf("CmdProvider.Load() missing key %s", key)
				} else if gotValue != expectedValue {
					t.Errorf("CmdProvider.Load() key %s = %v, want %v", key, gotValue, expectedValue)
				}
			}

			// 检查是否有多余的键
			for key := range got {
				if _, expected := tt.want[key]; !expected {
					t.Errorf("CmdProvider.Load() unexpected key %s = %v", key, got[key])
				}
			}
		})
	}
}

func TestCmdProvider_Save(t *testing.T) {
	provider, err := NewCmdProviderWithOptions(nil)
	if err != nil {
		t.Fatalf("NewCmdProviderWithOptions() error = %v", err)
	}

	err = provider.Save([]byte("test data"))
	if err == nil {
		t.Errorf("CmdProvider.Save() expected error, got nil")
	}

	// 检查错误信息
	if !strings.Contains(err.Error(), "does not support save") {
		t.Errorf("CmdProvider.Save() error message = %v, want contains 'does not support save'", err.Error())
	}
}

func TestCmdProvider_OnChange(t *testing.T) {
	provider, err := NewCmdProviderWithOptions(nil)
	if err != nil {
		t.Fatalf("NewCmdProviderWithOptions() error = %v", err)
	}

	// OnChange 应该直接返回，不做任何操作
	called := false
	provider.OnChange(func(data []byte) error {
		called = true
		return nil
	})

	// 验证回调没有被调用（因为不支持变更监听）
	if called {
		t.Errorf("CmdProvider.OnChange() callback was called, expected no call")
	}
}

func TestCmdProvider_Close(t *testing.T) {
	provider, err := NewCmdProviderWithOptions(nil)
	if err != nil {
		t.Fatalf("NewCmdProviderWithOptions() error = %v", err)
	}

	err = provider.Close()
	if err != nil {
		t.Errorf("CmdProvider.Close() error = %v, want nil", err)
	}
}

func TestCmdProvider_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		testArgs []string
		want     map[string]string
	}{
		{
			name:     "empty key after --",
			testArgs: []string{"--"},
			want:     map[string]string{},
		},
		{
			name:     "key with equals but empty value",
			testArgs: []string{"--key="},
			want: map[string]string{
				"key": "",
			},
		},
		{
			name:     "multiple equals signs",
			testArgs: []string{"--url=http://localhost:3000/path?param=value"},
			want: map[string]string{
				"url": "http://localhost:3000/path?param=value",
			},
		},
		{
			name:     "prefix edge cases",
			testArgs: []string{"--app-", "--app-host=localhost"},
			want: map[string]string{
				"host": "localhost", // --app-host 移除前缀后为 host
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := ""
			if strings.Contains(tt.name, "prefix") {
				prefix = "app-"
			}
			provider := &CmdProvider{
				prefix:   prefix,
				testArgs: tt.testArgs,
			}

			data, err := provider.Load()
			if err != nil {
				t.Errorf("CmdProvider.Load() error = %v", err)
				return
			}

			got := parseCmdData(string(data))

			// 检查期望的键值对
			for key, expectedValue := range tt.want {
				if gotValue, exists := got[key]; !exists {
					t.Errorf("CmdProvider.Load() missing key %s", key)
				} else if gotValue != expectedValue {
					t.Errorf("CmdProvider.Load() key %s = %v, want %v", key, gotValue, expectedValue)
				}
			}

			// 检查是否有多余的键
			for key := range got {
				if _, expected := tt.want[key]; !expected {
					t.Errorf("CmdProvider.Load() unexpected key %s = %v", key, got[key])
				}
			}
		})
	}
}

// parseCmdData 解析 .env 格式数据为 map
func parseCmdData(data string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
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
