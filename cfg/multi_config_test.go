package cfg

import (
	"testing"
	"time"

	"github.com/hatlonely/gox/cfg/provider"
	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/refx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMultiConfigWithOptions(t *testing.T) {
	t.Run("创建多配置源", func(t *testing.T) {
		options := &MultiConfigOptions{
			Sources: []*ConfigSourceOptions{
				{
					Provider: refx.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "EnvProvider",
						Options:   &provider.EnvProviderOptions{EnvFiles: []string{}},
					},
					Decoder: refx.TypeOptions{
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
			"port":    9090,
			"debug":   true,
			"env":     "production",
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
		assert.Equal(t, "app", appConfig.Name)         // 来自 base
		assert.Equal(t, 9090, appConfig.Port)          // 被 env 覆盖
		assert.Equal(t, true, appConfig.Debug)         // 被 env 覆盖
		assert.Equal(t, "base-feature", appConfig.Feature) // 来自 base
		assert.Equal(t, "production", appConfig.Env)   // 来自 env

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
		assert.Equal(t, "localhost", dbResult["host"]) // 来自 base
		assert.Equal(t, 3306, dbResult["port"])        // 被 env 覆盖
		assert.Equal(t, true, dbResult["ssl"])         // 被 env 覆盖
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
		logger, err := log.NewLogWithOptions(&log.Options{
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
					Provider: refx.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "EnvProvider",
						Options:   &provider.EnvProviderOptions{EnvFiles: []string{}},
					},
					Decoder: refx.TypeOptions{
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
					Provider: refx.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "EnvProvider",
						Options:   &provider.EnvProviderOptions{EnvFiles: []string{}},
					},
					Decoder: refx.TypeOptions{
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
					Provider: refx.TypeOptions{
						Namespace: "github.com/hatlonely/gox/cfg/provider",
						Type:      "EnvProvider",
						Options:   &provider.EnvProviderOptions{EnvFiles: []string{}},
					},
					Decoder: refx.TypeOptions{
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

