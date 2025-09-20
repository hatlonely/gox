package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewEnvProviderWithOptions(t *testing.T) {
	Convey("测试NewEnvProviderWithOptions函数", t, func() {
		testNewEnvProvider := func(name string, options *EnvProviderOptions, wantErr bool) {
			Convey(name, func() {
				provider, err := NewEnvProviderWithOptions(options)
				if wantErr {
					So(err, ShouldNotBeNil)
				} else {
					So(err, ShouldBeNil)
					So(provider, ShouldNotBeNil)
				}
			})
		}

		testNewEnvProvider("使用nil选项", nil, false)
		testNewEnvProvider("使用空选项", &EnvProviderOptions{}, false)
		testNewEnvProvider("使用有效的环境文件", &EnvProviderOptions{
			EnvFiles: []string{"test1.env", "test2.env"},
		}, false)
		testNewEnvProvider("混合有效和空文件", &EnvProviderOptions{
			EnvFiles: []string{"test.env", "", "test2.env"},
		}, false)
	})
}

func TestEnvProvider_Load(t *testing.T) {
	Convey("测试EnvProvider的Load功能", t, func() {
		// 创建临时目录
		tempDir, err := os.MkdirTemp("", "env_provider_test")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)

		// 创建测试 .env 文件
		env1File := filepath.Join(tempDir, "test1.env")
		env1Content := `# Test env file 1
APP_NAME=TestApp
DB_HOST=localhost
DB_PORT=3306
DEBUG=true`

		err = os.WriteFile(env1File, []byte(env1Content), 0644)
		So(err, ShouldBeNil)

		env2File := filepath.Join(tempDir, "test2.env")
		env2Content := `# Test env file 2
DB_PORT=5432
DB_USER=admin
API_KEY="secret key with spaces"`

		err = os.WriteFile(env2File, []byte(env2Content), 0644)
		So(err, ShouldBeNil)

		testLoad := func(name string, envFiles []string, envVars map[string]string, want map[string]string) {
			Convey(name, func() {
				// 设置测试环境变量
				for key, value := range envVars {
					os.Setenv(key, value)
					defer os.Unsetenv(key)
				}

				provider, err := NewEnvProviderWithOptions(&EnvProviderOptions{
					EnvFiles: envFiles,
				})
				So(err, ShouldBeNil)

				data, err := provider.Load()
				So(err, ShouldBeNil)

				// 解析返回的 .env 格式数据
				got := parseEnvData(string(data))

				// 检查期望的键值对
				for key, expectedValue := range want {
					So(got, ShouldContainKey, key)
					So(got[key], ShouldEqual, expectedValue)
				}

				// 检查是否有多余的键
				for key := range got {
					if _, expected := want[key]; !expected {
						// 允许系统环境变量存在
						if !strings.HasPrefix(key, "TEST_") && !strings.HasPrefix(key, "EXISTING_") && !strings.HasPrefix(key, "SYSTEM_") {
							continue
						}
						So(want, ShouldContainKey, key)
					}
				}
			})
		}

		testLoad("仅系统环境变量，无文件", []string{}, 
			map[string]string{"TEST_VAR": "test_value"}, 
			map[string]string{"TEST_VAR": "test_value"})

		testLoad("单个环境文件", []string{env1File}, 
			map[string]string{"EXISTING_VAR": "existing"}, 
			map[string]string{
				"EXISTING_VAR": "existing",
				"APP_NAME":     "TestApp",
				"DB_HOST":      "localhost",
				"DB_PORT":      "3306",
				"DEBUG":        "true",
			})

		testLoad("多个环境文件及优先级", []string{env1File, env2File}, 
			map[string]string{"SYSTEM_VAR": "system"}, 
			map[string]string{
				"SYSTEM_VAR": "system",
				"APP_NAME":   "TestApp",
				"DB_HOST":    "localhost",
				"DB_PORT":    "5432", // env2 覆盖 env1
				"DEBUG":      "true",
				"DB_USER":    "admin",
				"API_KEY":    `"secret key with spaces"`, // 保持原始格式
			})

		testLoad("不存在的文件不应该导致错误", []string{env1File, "/nonexistent/file.env", env2File}, 
			map[string]string{}, 
			map[string]string{
				"APP_NAME": "TestApp",
				"DB_HOST":  "localhost",
				"DB_PORT":  "5432", // env2 覆盖 env1
				"DEBUG":    "true",
				"DB_USER":  "admin",
				"API_KEY":  `"secret key with spaces"`,
			})
	})
}

func TestEnvProvider_Save(t *testing.T) {
	Convey("测试EnvProvider的Save功能", t, func() {
		provider, err := NewEnvProviderWithOptions(nil)
		So(err, ShouldBeNil)

		Convey("Save操作应该返回错误（不支持保存）", func() {
			err = provider.Save([]byte("test data"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "does not support save")
		})
	})
}

func TestEnvProvider_OnChange(t *testing.T) {
	Convey("测试EnvProvider的OnChange功能", t, func() {
		provider, err := NewEnvProviderWithOptions(nil)
		So(err, ShouldBeNil)

		Convey("OnChange应该直接返回，不做任何操作", func() {
			called := false
			provider.OnChange(func(data []byte) error {
				called = true
				return nil
			})

			// 验证回调没有被调用（因为不支持变更监听）
			So(called, ShouldBeFalse)
		})
	})
}

func TestEnvProvider_Close(t *testing.T) {
	Convey("测试EnvProvider的Close功能", t, func() {
		provider, err := NewEnvProviderWithOptions(nil)
		So(err, ShouldBeNil)

		Convey("Close操作应该成功", func() {
			err = provider.Close()
			So(err, ShouldBeNil)
		})
	})
}

func TestEnvProvider_loadEnvFile(t *testing.T) {
	Convey("测试EnvProvider的loadEnvFile功能", t, func() {
		// 创建临时文件
		tempFile, err := os.CreateTemp("", "test.env")
		So(err, ShouldBeNil)
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

		_, err = tempFile.WriteString(content)
		So(err, ShouldBeNil)
		tempFile.Close()

		Convey("加载环境文件应该成功解析各种格式", func() {
			provider := &EnvProvider{}
			envVars := make(map[string]string)

			err = provider.loadEnvFile(tempFile.Name(), envVars)
			So(err, ShouldBeNil)

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
				So(envVars, ShouldContainKey, key)
				So(envVars[key], ShouldEqual, expectedValue)
			}
		})
	})
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

func TestEnvProvider_Watch(t *testing.T) {
	Convey("测试EnvProvider的Watch功能", t, func() {
		provider, err := NewEnvProviderWithOptions(nil)
		So(err, ShouldBeNil)
		defer provider.Close()

		Convey("EnvProvider不支持Watch，但不应该返回错误", func() {
			err = provider.Watch()
			So(err, ShouldBeNil)
		})
	})
}

func TestEnvProvider_LoadWithPrefix(t *testing.T) {
	Convey("测试EnvProvider的前缀过滤功能", t, func() {
		// 设置测试环境变量
		os.Setenv("APP_DATABASE_HOST", "localhost")
		os.Setenv("APP_DATABASE_PORT", "3306")
		os.Setenv("APP_DEBUG", "true")
		os.Setenv("OTHER_KEY", "should_be_ignored")
		os.Setenv("APP_", "empty_after_prefix") // 应该被忽略
		os.Setenv("APPOTHER", "not_matching")   // 不匹配前缀
		defer func() {
			os.Unsetenv("APP_DATABASE_HOST")
			os.Unsetenv("APP_DATABASE_PORT") 
			os.Unsetenv("APP_DEBUG")
			os.Unsetenv("OTHER_KEY")
			os.Unsetenv("APP_")
			os.Unsetenv("APPOTHER")
		}()

		testWithPrefix := func(name, prefix string, want map[string]string) {
			Convey(name, func() {
				provider, err := NewEnvProviderWithOptions(&EnvProviderOptions{
					Prefix: prefix,
				})
				So(err, ShouldBeNil)

				data, err := provider.Load()
				So(err, ShouldBeNil)

				// 解析返回的数据
				got := parseEnvData(string(data))

				// 检查期望的键值对
				for key, expectedValue := range want {
					So(got, ShouldContainKey, key)
					So(got[key], ShouldEqual, expectedValue)
				}

				// 检查不应该包含的键（仅检查我们的测试键）
				testKeys := []string{"APP_DATABASE_HOST", "APP_DATABASE_PORT", "APP_DEBUG", "OTHER_KEY", "APP_", "APPOTHER"}
				for _, testKey := range testKeys {
					if _, exists := got[testKey]; exists {
						// 如果设置了前缀，原始键不应该存在
						if prefix != "" {
							So(false, ShouldBeTrue) // 强制失败，因为不应该包含原始键
						}
					}
				}
			})
		}

		testWithPrefix("无前缀", "", map[string]string{
			"APP_DATABASE_HOST": "localhost",
			"APP_DATABASE_PORT": "3306", 
			"APP_DEBUG":         "true",
			"OTHER_KEY":         "should_be_ignored",
			"APP_":              "empty_after_prefix",
			"APPOTHER":          "not_matching",
		})

		testWithPrefix("使用APP_前缀", "APP_", map[string]string{
			"DATABASE_HOST": "localhost",
			"DATABASE_PORT": "3306",
			"DEBUG":         "true",
		})

		testWithPrefix("使用OTHER_前缀", "OTHER_", map[string]string{
			"KEY": "should_be_ignored",
		})

		testWithPrefix("不匹配的前缀", "NONEXIST_", map[string]string{})
	})
}

func TestEnvProvider_LoadWithPrefixFromFile(t *testing.T) {
	Convey("测试EnvProvider从文件加载时的前缀过滤功能", t, func() {
		// 创建临时目录和文件
		tempDir, err := os.MkdirTemp("", "env_provider_prefix_test")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)

		// 创建测试 .env 文件
		envFile := filepath.Join(tempDir, "test.env")
		envContent := `# Test env file with various prefixes
APP_DATABASE_HOST=localhost
APP_DATABASE_PORT=3306
OTHER_SERVER_HOST=example.com
OTHER_SERVER_PORT=8080
APP_=empty_key
STANDALONE_KEY=standalone_value`

		err = os.WriteFile(envFile, []byte(envContent), 0644)
		So(err, ShouldBeNil)

		testFilePrefix := func(name, prefix string, want map[string]string) {
			Convey(name, func() {
				provider, err := NewEnvProviderWithOptions(&EnvProviderOptions{
					EnvFiles: []string{envFile},
					Prefix:   prefix,
				})
				So(err, ShouldBeNil)

				data, err := provider.Load()
				So(err, ShouldBeNil)

				// 解析返回的数据
				got := parseEnvData(string(data))

				// 检查期望的键值对
				for key, expectedValue := range want {
					So(got, ShouldContainKey, key)
					So(got[key], ShouldEqual, expectedValue)
				}
			})
		}

		testFilePrefix("文件中APP_前缀", "APP_", map[string]string{
			"DATABASE_HOST": "localhost",
			"DATABASE_PORT": "3306",
		})

		testFilePrefix("文件中OTHER_前缀", "OTHER_", map[string]string{
			"SERVER_HOST": "example.com",
			"SERVER_PORT": "8080",
		})

		testFilePrefix("文件中无前缀", "", map[string]string{
			"APP_DATABASE_HOST":  "localhost",
			"APP_DATABASE_PORT":  "3306",
			"OTHER_SERVER_HOST":  "example.com",
			"OTHER_SERVER_PORT":  "8080",
			"STANDALONE_KEY":     "standalone_value",
		})
	})
}
