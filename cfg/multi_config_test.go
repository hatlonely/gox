package cfg

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hatlonely/gox/cfg/decoder"
	"github.com/hatlonely/gox/cfg/provider"
	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/ref"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMultiConfigWithOptions(t *testing.T) {
	t.Run("创建多配置源", func(t *testing.T) {
		options := &MultiConfigOptions{
			Sources: []*ConfigSourceOptions{
				{
					Provider: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "EnvProvider",
						Options:   &provider.EnvProviderOptions{EnvFiles: []string{}},
					},
					Decoder: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/decoder",
						Type:      "EnvDecoder",
					},
				},
			},
		}

		config, err := NewMultiConfigWithOptions(options)
		require.NoError(t, err)
		require.NotNil(t, config)

		// 验证配置源数量
		assert.Equal(t, 1, len(config.sources))

		// 验证基本功能
		var result map[string]interface{}
		err = config.ConvertTo(&result)
		assert.NoError(t, err)

		// 清理
		err = config.Close()
		assert.NoError(t, err)
	})

	t.Run("空配置源应该失败", func(t *testing.T) {
		options := &MultiConfigOptions{
			Sources: []*ConfigSourceOptions{},
		}
		_, err := NewMultiConfigWithOptions(options)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one configuration source is required")
	})

	t.Run("nil选项应该失败", func(t *testing.T) {
		_, err := NewMultiConfigWithOptions(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "options cannot be nil")
	})
}

func TestMultiConfig_ConvertTo(t *testing.T) {
	t.Run("多个配置源合并", func(t *testing.T) {
		// 创建测试数据，模拟从不同源加载的配置
		// 这里我们直接创建 MultiConfig 来测试核心功能

		// 基础配置
		baseStorage := storage.NewMapStorage(map[string]interface{}{
			"name":    "app",
			"port":    8080,
			"debug":   false,
			"feature": "base-feature",
		})

		// 环境配置
		envStorage := storage.NewMapStorage(map[string]interface{}{
			"port":  9090,
			"debug": true,
			"env":   "production",
		})

		// 创建 MultiStorage
		multiStorage := storage.NewMultiStorage([]storage.Storage{baseStorage, envStorage})

		config := &MultiConfig{
			multiStorage:        multiStorage,
			onKeyChangeHandlers: make(map[string][]func(storage.Storage) error),
		}

		// 测试转换到结构体
		type AppConfig struct {
			Name    string `cfg:"name"`
			Port    int    `cfg:"port"`
			Debug   bool   `cfg:"debug"`
			Feature string `cfg:"feature"`
			Env     string `cfg:"env"`
		}

		var appConfig AppConfig
		err := config.ConvertTo(&appConfig)
		require.NoError(t, err)

		// 验证合并结果
		assert.Equal(t, "app", appConfig.Name)             // 来自 base
		assert.Equal(t, 9090, appConfig.Port)              // 被 env 覆盖
		assert.Equal(t, true, appConfig.Debug)             // 被 env 覆盖
		assert.Equal(t, "base-feature", appConfig.Feature) // 来自 base
		assert.Equal(t, "production", appConfig.Env)       // 来自 env

		// 测试转换到 map
		var mapConfig map[string]interface{}
		err = config.ConvertTo(&mapConfig)
		require.NoError(t, err)

		// 验证 map 的增量合并
		assert.Equal(t, "app", mapConfig["name"])
		assert.Equal(t, 9090, mapConfig["port"])
		assert.Equal(t, true, mapConfig["debug"])
		assert.Equal(t, "base-feature", mapConfig["feature"])
		assert.Equal(t, "production", mapConfig["env"])
	})
}

func TestMultiConfig_Sub(t *testing.T) {
	t.Run("获取子配置", func(t *testing.T) {
		// 创建嵌套配置
		baseStorage := storage.NewMapStorage(map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 5432,
				"ssl":  false,
			},
			"redis": map[string]interface{}{
				"host": "127.0.0.1",
				"port": 6379,
			},
		})

		envStorage := storage.NewMapStorage(map[string]interface{}{
			"database": map[string]interface{}{
				"port": 3306,
				"ssl":  true,
				"name": "production_db",
			},
		})

		multiStorage := storage.NewMultiStorage([]storage.Storage{baseStorage, envStorage})
		config := &MultiConfig{
			multiStorage:        multiStorage,
			onKeyChangeHandlers: make(map[string][]func(storage.Storage) error),
		}

		// 获取数据库子配置
		dbConfig := config.Sub("database")
		require.NotNil(t, dbConfig)

		var dbResult map[string]interface{}
		err := dbConfig.ConvertTo(&dbResult)
		require.NoError(t, err)

		// 验证子配置的合并
		assert.Equal(t, "localhost", dbResult["host"])     // 来自 base
		assert.Equal(t, 3306, dbResult["port"])            // 被 env 覆盖
		assert.Equal(t, true, dbResult["ssl"])             // 被 env 覆盖
		assert.Equal(t, "production_db", dbResult["name"]) // 来自 env

		// 获取 Redis 子配置
		redisConfig := config.Sub("redis")
		var redisResult map[string]interface{}
		err = redisConfig.ConvertTo(&redisResult)
		require.NoError(t, err)

		assert.Equal(t, "127.0.0.1", redisResult["host"])
		assert.Equal(t, 6379, redisResult["port"])
	})

	t.Run("空键返回自身", func(t *testing.T) {
		mapStorage := storage.NewMapStorage(map[string]interface{}{
			"key": "value",
		})

		multiStorage := storage.NewMultiStorage([]storage.Storage{mapStorage})
		config := &MultiConfig{
			multiStorage:        multiStorage,
			onKeyChangeHandlers: make(map[string][]func(storage.Storage) error),
		}

		sub := config.Sub("")
		assert.Equal(t, config, sub)
	})
}

func TestMultiConfig_OnChange(t *testing.T) {
	t.Run("注册变更监听器", func(t *testing.T) {
		config := &MultiConfig{
			onKeyChangeHandlers: make(map[string][]func(storage.Storage) error),
		}

		config.OnChange(func(storage storage.Storage) error {
			return nil
		})

		// 验证处理器已注册
		assert.Len(t, config.onKeyChangeHandlers[""], 1)

		// 模拟触发变更
		if len(config.onKeyChangeHandlers[""]) > 0 {
			handler := config.onKeyChangeHandlers[""][0]
			err := handler(nil)
			assert.NoError(t, err)
		}
	})

	t.Run("注册特定键的变更监听器", func(t *testing.T) {
		config := &MultiConfig{
			onKeyChangeHandlers: make(map[string][]func(storage.Storage) error),
		}

		config.OnKeyChange("database", func(storage storage.Storage) error {
			return nil
		})

		// 验证处理器已注册到正确的键
		assert.Len(t, config.onKeyChangeHandlers["database"], 1)
		assert.Len(t, config.onKeyChangeHandlers[""], 0)
	})
}

func TestMultiConfig_SetLogger(t *testing.T) {
	t.Run("设置日志记录器", func(t *testing.T) {
		config := &MultiConfig{}

		// 创建测试 logger
		logger, err := log.NewSLogWithOptions(&log.SLogOptions{
			Level:  "debug",
			Format: "text",
		})
		require.NoError(t, err)

		config.SetLogger(logger)
		assert.Equal(t, logger, config.logger)
	})
}

func TestMultiConfig_Close(t *testing.T) {
	t.Run("关闭配置对象", func(t *testing.T) {
		// 创建一个简单的 MultiConfig 用于测试
		options := &MultiConfigOptions{
			Sources: []*ConfigSourceOptions{
				{
					Provider: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "EnvProvider",
						Options:   &provider.EnvProviderOptions{EnvFiles: []string{}},
					},
					Decoder: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/decoder",
						Type:      "EnvDecoder",
					},
				},
			},
		}

		config, err := NewMultiConfigWithOptions(options)
		require.NoError(t, err)

		// 关闭配置
		err = config.Close()
		assert.NoError(t, err)

		// 再次关闭应该返回相同的结果
		err = config.Close()
		assert.NoError(t, err)
	})
}

func TestMultiConfig_HandlerExecution(t *testing.T) {
	t.Run("处理器执行配置", func(t *testing.T) {
		options := &MultiConfigOptions{
			Sources: []*ConfigSourceOptions{
				{
					Provider: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "EnvProvider",
						Options:   &provider.EnvProviderOptions{EnvFiles: []string{}},
					},
					Decoder: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/decoder",
						Type:      "EnvDecoder",
					},
				},
			},
			HandlerExecution: &HandlerExecutionOptions{
				Timeout:     10 * time.Second,
				Async:       false,
				ErrorPolicy: "stop",
			},
		}

		config, err := NewMultiConfigWithOptions(options)
		require.NoError(t, err)
		defer config.Close()

		// 验证处理器执行配置
		assert.Equal(t, 10*time.Second, config.handlerExecution.Timeout)
		assert.Equal(t, false, config.handlerExecution.Async)
		assert.Equal(t, "stop", config.handlerExecution.ErrorPolicy)
	})

	t.Run("默认处理器执行配置", func(t *testing.T) {
		options := &MultiConfigOptions{
			Sources: []*ConfigSourceOptions{
				{
					Provider: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "EnvProvider",
						Options:   &provider.EnvProviderOptions{EnvFiles: []string{}},
					},
					Decoder: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/decoder",
						Type:      "EnvDecoder",
					},
				},
			},
		}

		config, err := NewMultiConfigWithOptions(options)
		require.NoError(t, err)
		defer config.Close()

		// 验证默认配置
		assert.Equal(t, 5*time.Second, config.handlerExecution.Timeout)
		assert.Equal(t, true, config.handlerExecution.Async)
		assert.Equal(t, "continue", config.handlerExecution.ErrorPolicy)
	})
}

// TestMultiConfig_ValidateStorageIntegration 测试 MultiConfig 的 ValidateStorage 集成功能
func TestMultiConfig_ValidateStorageIntegration(t *testing.T) {
	t.Run("ConvertTo with validation success", func(t *testing.T) {
		// 创建配置源：第一个基础配置，第二个覆盖配置
		baseConfig := map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "testuser",
				"email": "test@example.com",
				"age":   25,
			},
		}

		overrideConfig := map[string]interface{}{
			"user": map[string]interface{}{
				"age": 30, // 覆盖年龄
			},
		}

		// 创建临时 JSON 文件用于测试
		baseConfigData, _ := json.Marshal(baseConfig)
		os.WriteFile("/tmp/test_config_base.json", baseConfigData, 0644)
		overrideConfigData, _ := json.Marshal(overrideConfig)
		os.WriteFile("/tmp/test_config_override.json", overrideConfigData, 0644)

		options := &MultiConfigOptions{
			Sources: []*ConfigSourceOptions{
				{
					Provider: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "FileProvider",
						Options: &provider.FileProviderOptions{
							FilePath: "/tmp/test_config_base.json",
						},
					},
					Decoder: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/decoder",
						Type:      "JsonDecoder",
						Options:   &decoder.JsonDecoderOptions{},
					},
				},
				{
					Provider: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "FileProvider",
						Options: &provider.FileProviderOptions{
							FilePath: "/tmp/test_config_override.json",
						},
					},
					Decoder: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/decoder",
						Type:      "JsonDecoder",
						Options:   &decoder.JsonDecoderOptions{},
					},
				},
			},
		}

		config, err := NewMultiConfigWithOptions(options)
		require.NoError(t, err)
		defer config.Close()

		// 定义用于测试的结构体，包含校验标签
		type User struct {
			Name  string `cfg:"name" validate:"required,min=3,max=20"`
			Email string `cfg:"email" validate:"required,email"`
			Age   int    `cfg:"age" validate:"min=18,max=120"`
		}

		var user User
		userConfig := config.Sub("user")

		// 测试 ConvertTo 方法的自动校验
		err = userConfig.ConvertTo(&user)
		require.NoError(t, err, "ConvertTo should succeed with valid data")

		// 验证数据正确合并和校验
		assert.Equal(t, "testuser", user.Name)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, 30, user.Age) // 应该是覆盖后的值
	})

	t.Run("ConvertTo with validation failure", func(t *testing.T) {
		// 创建包含无效数据的配置
		invalidConfig := map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "ab",            // 太短，不满足 min=3
				"email": "invalid-email", // 无效邮箱
				"age":   15,              // 太小，不满足 min=18
			},
		}

		// 创建临时 JSON 文件用于测试
		invalidConfigData, _ := json.Marshal(invalidConfig)
		os.WriteFile("/tmp/test_config_invalid.json", invalidConfigData, 0644)

		options := &MultiConfigOptions{
			Sources: []*ConfigSourceOptions{
				{
					Provider: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "FileProvider",
						Options: &provider.FileProviderOptions{
							FilePath: "/tmp/test_config_invalid.json",
						},
					},
					Decoder: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/decoder",
						Type:      "JsonDecoder",
						Options:   &decoder.JsonDecoderOptions{},
					},
				},
			},
		}

		config, err := NewMultiConfigWithOptions(options)
		require.NoError(t, err)
		defer config.Close()

		type User struct {
			Name  string `cfg:"name" validate:"required,min=3,max=20"`
			Email string `cfg:"email" validate:"required,email"`
			Age   int    `cfg:"age" validate:"min=18,max=120"`
		}

		var user User
		userConfig := config.Sub("user")

		// 测试 ConvertTo 方法的自动校验失败
		err = userConfig.ConvertTo(&user)
		require.Error(t, err, "ConvertTo should fail with invalid data")
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("Root config validation", func(t *testing.T) {
		// 测试根配置的校验
		appConfig := map[string]interface{}{
			"app": map[string]interface{}{
				"name":    "testapp",
				"version": "1.0.0",
				"debug":   true,
			},
		}

		// 创建临时 JSON 文件用于测试
		appConfigData, _ := json.Marshal(appConfig)
		os.WriteFile("/tmp/test_config_app.json", appConfigData, 0644)

		options := &MultiConfigOptions{
			Sources: []*ConfigSourceOptions{
				{
					Provider: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "FileProvider",
						Options: &provider.FileProviderOptions{
							FilePath: "/tmp/test_config_app.json",
						},
					},
					Decoder: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/decoder",
						Type:      "JsonDecoder",
						Options:   &decoder.JsonDecoderOptions{},
					},
				},
			},
		}

		config, err := NewMultiConfigWithOptions(options)
		require.NoError(t, err)
		defer config.Close()

		type AppConfig struct {
			App struct {
				Name    string `cfg:"name" validate:"required,min=3"`
				Version string `cfg:"version" validate:"required"`
				Debug   bool   `cfg:"debug"`
			} `cfg:"app"`
		}

		var appConfigStruct AppConfig
		// 测试根配置的 ConvertTo 自动校验
		err = config.ConvertTo(&appConfigStruct)
		require.NoError(t, err, "Root config ConvertTo should succeed")

		assert.Equal(t, "testapp", appConfigStruct.App.Name)
		assert.Equal(t, "1.0.0", appConfigStruct.App.Version)
		assert.Equal(t, true, appConfigStruct.App.Debug)
	})

	t.Run("Multiple source priority and validation", func(t *testing.T) {
		// 测试多配置源的优先级和校验
		baseConfig := map[string]interface{}{
			"database": map[string]interface{}{
				"host":     "localhost",
				"port":     3306,
				"username": "testuser",
				"password": "testpass123",
			},
		}

		// 高优先级配置覆盖一些字段
		highPriorityConfig := map[string]interface{}{
			"database": map[string]interface{}{
				"host": "production.db.com",
				"port": 5432,
			},
		}

		// 创建临时 JSON 文件用于测试
		baseConfigData, _ := json.Marshal(baseConfig)
		os.WriteFile("/tmp/test_config_base_db.json", baseConfigData, 0644)
		highPriorityConfigData, _ := json.Marshal(highPriorityConfig)
		os.WriteFile("/tmp/test_config_high_priority.json", highPriorityConfigData, 0644)

		options := &MultiConfigOptions{
			Sources: []*ConfigSourceOptions{
				{
					Provider: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "FileProvider",
						Options: &provider.FileProviderOptions{
							FilePath: "/tmp/test_config_base_db.json",
						},
					},
					Decoder: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/decoder",
						Type:      "JsonDecoder",
						Options:   &decoder.JsonDecoderOptions{},
					},
				},
				{
					Provider: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "FileProvider",
						Options: &provider.FileProviderOptions{
							FilePath: "/tmp/test_config_high_priority.json",
						},
					},
					Decoder: ref.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/decoder",
						Type:      "JsonDecoder",
						Options:   &decoder.JsonDecoderOptions{},
					},
				},
			},
		}

		config, err := NewMultiConfigWithOptions(options)
		require.NoError(t, err)
		defer config.Close()

		type DatabaseConfig struct {
			Host     string `cfg:"host" validate:"required"`
			Port     int    `cfg:"port" validate:"min=1,max=65535"`
			Username string `cfg:"username" validate:"required,min=3"`
			Password string `cfg:"password" validate:"required,min=8"`
		}

		var dbConfig DatabaseConfig
		// 测试嵌套配置的自动校验和优先级合并
		dbSubConfig := config.Sub("database")
		err = dbSubConfig.ConvertTo(&dbConfig)
		require.NoError(t, err, "Database config ConvertTo should succeed")

		// 验证高优先级配置覆盖了低优先级配置
		assert.Equal(t, "production.db.com", dbConfig.Host) // 来自高优先级配置
		assert.Equal(t, 5432, dbConfig.Port)                // 来自高优先级配置
		assert.Equal(t, "testuser", dbConfig.Username)      // 来自基础配置
		assert.Equal(t, "testpass123", dbConfig.Password)   // 来自基础配置
	})
}

func TestMain(m *testing.M) {
	// 运行测试
	code := m.Run()

	// 清理临时文件
	os.Remove("/tmp/test_config_base.json")
	os.Remove("/tmp/test_config_override.json")
	os.Remove("/tmp/test_config_invalid.json")
	os.Remove("/tmp/test_config_app.json")
	os.Remove("/tmp/test_config_base_db.json")
	os.Remove("/tmp/test_config_high_priority.json")

	os.Exit(code)
}
