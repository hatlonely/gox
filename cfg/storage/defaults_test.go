package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试用的结构体
type ConfigWithDefaults struct {
	Name        string        `json:"name" def:"default_name"`
	Age         int           `json:"age" def:"25"`
	Height      float64       `json:"height" def:"175.5"`
	IsActive    bool          `json:"is_active" def:"true"`
	Tags        []string      `json:"tags" def:"tag1,tag2,tag3"`
	Timeout     time.Duration `json:"timeout" def:"30s"`
	Description *string       `json:"description" def:"default description"`

	// 嵌套结构体
	Database DatabaseConfigWithDefaults `json:"database"`

	// 指针类型的嵌套结构体
	Cache *CacheConfigWithDefaults `json:"cache,omitempty"`

	// Map 字段，用于测试 map 中新创建的结构体对象
	Services map[string]ServiceConfig `json:"services"`
}

type DatabaseConfigWithDefaults struct {
	Host     string `json:"host" def:"localhost"`
	Port     int    `json:"port" def:"3306"`
	Username string `json:"username" def:"root"`
	Password string `json:"password" def:""`
}

type CacheConfigWithDefaults struct {
	Redis RedisConfigWithDefaults `json:"redis"`
}

type RedisConfigWithDefaults struct {
	Host string `json:"host" def:"redis-host"`
	Port int    `json:"port" def:"6379"`
}

type ServiceConfig struct {
	Host    string `json:"host" def:"service-host"`
	Port    int    `json:"port" def:"8080"`
	Enabled bool   `json:"enabled" def:"true"`
}

func TestMapStorage_WithDefaults(t *testing.T) {
	data := map[string]interface{}{
		"age":    30, // 覆盖默认值
		"height": 180.0,
		"database": map[string]interface{}{
			"username": "admin", // 覆盖默认值
		},
		"services": map[string]interface{}{
			"api": map[string]interface{}{
				"port": 9000, // 覆盖默认值
			},
			"web": map[string]interface{}{}, // 空配置，应该使用默认值
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &ConfigWithDefaults{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证基本类型默认值
	assert.Equal(t, "default_name", config.Name) // 使用默认值
	assert.Equal(t, 30, config.Age)              // 使用配置值
	assert.Equal(t, 180.0, config.Height)       // 使用配置值
	assert.Equal(t, true, config.IsActive)      // 使用默认值
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, config.Tags) // 使用默认值
	assert.Equal(t, 30*time.Second, config.Timeout)                // 使用默认值

	// 验证指针类型默认值
	assert.NotNil(t, config.Description)
	assert.Equal(t, "default description", *config.Description) // 使用默认值

	// 验证嵌套结构体默认值
	assert.Equal(t, "localhost", config.Database.Host)     // 使用默认值
	assert.Equal(t, 3306, config.Database.Port)            // 使用默认值
	assert.Equal(t, "admin", config.Database.Username)     // 使用配置值
	assert.Equal(t, "", config.Database.Password)          // 使用默认值

	// 验证 Map 中新创建的结构体对象默认值
	assert.NotNil(t, config.Services)
	assert.Contains(t, config.Services, "api")
	assert.Contains(t, config.Services, "web")

	// api 服务：port 使用配置值，其他使用默认值
	apiService := config.Services["api"]
	assert.Equal(t, "service-host", apiService.Host) // 默认值
	assert.Equal(t, 9000, apiService.Port)           // 配置值
	assert.Equal(t, true, apiService.Enabled)        // 默认值

	// web 服务：全部使用默认值
	webService := config.Services["web"]
	assert.Equal(t, "service-host", webService.Host) // 默认值
	assert.Equal(t, 8080, webService.Port)           // 默认值
	assert.Equal(t, true, webService.Enabled)        // 默认值
}

func TestMapStorage_WithoutDefaults(t *testing.T) {
	data := map[string]interface{}{
		"age": 30,
	}

	ms := NewMapStorageWithoutDefaults(data)
	config := &ConfigWithDefaults{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证不设置默认值
	assert.Equal(t, "", config.Name)       // 零值
	assert.Equal(t, 30, config.Age)        // 配置值
	assert.Equal(t, 0.0, config.Height)    // 零值
	assert.Equal(t, false, config.IsActive) // 零值
	assert.Nil(t, config.Tags)             // 零值
	assert.Equal(t, time.Duration(0), config.Timeout) // 零值
	assert.Nil(t, config.Description)      // 零值

	// 验证嵌套结构体不设置默认值
	assert.Equal(t, "", config.Database.Host)     // 零值
	assert.Equal(t, 0, config.Database.Port)      // 零值
	assert.Equal(t, "", config.Database.Username) // 零值
	assert.Equal(t, "", config.Database.Password) // 零值
}

func TestFlatStorage_WithDefaults(t *testing.T) {
	data := map[string]interface{}{
		"age":               30,
		"height":            180.0,
		"database.username": "admin",
	}

	fs := NewFlatStorage(data).WithDefaults(true)
	config := &ConfigWithDefaults{}

	err := fs.ConvertTo(config)
	assert.NoError(t, err)

	// 验证基本类型默认值
	assert.Equal(t, "default_name", config.Name) // 使用默认值
	assert.Equal(t, 30, config.Age)              // 使用配置值
	assert.Equal(t, 180.0, config.Height)       // 使用配置值
	assert.Equal(t, true, config.IsActive)      // 使用默认值
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, config.Tags) // 使用默认值
	assert.Equal(t, 30*time.Second, config.Timeout)                // 使用默认值

	// 验证指针类型默认值
	assert.NotNil(t, config.Description)
	assert.Equal(t, "default description", *config.Description) // 使用默认值

	// 验证嵌套结构体默认值
	assert.Equal(t, "localhost", config.Database.Host)     // 使用默认值
	assert.Equal(t, 3306, config.Database.Port)            // 使用默认值
	assert.Equal(t, "admin", config.Database.Username)     // 使用配置值
	assert.Equal(t, "", config.Database.Password)          // 使用默认值
}

func TestFlatStorage_WithoutDefaults(t *testing.T) {
	data := map[string]interface{}{
		"age": 30,
	}

	fs := NewFlatStorageWithoutDefaults(data)
	config := &ConfigWithDefaults{}

	err := fs.ConvertTo(config)
	assert.NoError(t, err)

	// 验证不设置默认值
	assert.Equal(t, "", config.Name)       // 零值
	assert.Equal(t, 30, config.Age)        // 配置值
	assert.Equal(t, 0.0, config.Height)    // 零值
	assert.Equal(t, false, config.IsActive) // 零值
}

func TestMapStorage_Sub_InheritsDefaults(t *testing.T) {
	data := map[string]interface{}{
		"database": map[string]interface{}{
			"username": "admin",
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	subStorage := ms.Sub("database")
	
	dbConfig := &DatabaseConfigWithDefaults{}
	err := subStorage.ConvertTo(dbConfig)
	assert.NoError(t, err)

	// 验证子存储继承了默认值设置
	assert.Equal(t, "localhost", dbConfig.Host)     // 使用默认值
	assert.Equal(t, 3306, dbConfig.Port)            // 使用默认值
	assert.Equal(t, "admin", dbConfig.Username)     // 使用配置值
	assert.Equal(t, "", dbConfig.Password)          // 使用默认值
}

func TestFlatStorage_Sub_InheritsDefaults(t *testing.T) {
	data := map[string]interface{}{
		"database.username": "admin",
	}

	fs := NewFlatStorage(data).WithDefaults(true)
	subStorage := fs.Sub("database")
	
	dbConfig := &DatabaseConfigWithDefaults{}
	err := subStorage.ConvertTo(dbConfig)
	assert.NoError(t, err)

	// 验证子存储继承了默认值设置
	assert.Equal(t, "localhost", dbConfig.Host)     // 使用默认值
	assert.Equal(t, 3306, dbConfig.Port)            // 使用默认值
	assert.Equal(t, "admin", dbConfig.Username)     // 使用配置值
	assert.Equal(t, "", dbConfig.Password)          // 使用默认值
}

func TestPointerStructDefaults(t *testing.T) {
	data := map[string]interface{}{
		"cache": map[string]interface{}{
			"redis": map[string]interface{}{
				"port": 7000, // 覆盖默认值
			},
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &ConfigWithDefaults{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证指针结构体的默认值设置
	assert.NotNil(t, config.Cache)
	assert.Equal(t, "redis-host", config.Cache.Redis.Host) // 使用默认值
	assert.Equal(t, 7000, config.Cache.Redis.Port)         // 使用配置值
}