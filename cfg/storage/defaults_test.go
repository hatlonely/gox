package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============ 测试结构体定义 ============

// 基础配置结构体，包含各种类型的字段
type BasicConfig struct {
	Name        string        `json:"name" def:"default_name"`
	Age         int           `json:"age" def:"25"`
	Height      float64       `json:"height" def:"175.5"`
	IsActive    bool          `json:"is_active" def:"true"`
	Tags        []string      `json:"tags" def:"tag1,tag2,tag3"`
	Timeout     time.Duration `json:"timeout" def:"30s"`
	Description *string       `json:"description" def:"default description"`
}

// 带有嵌套结构体的配置
type ConfigWithNested struct {
	Name        string        `json:"name" def:"default_name"`
	Age         int           `json:"age" def:"25"`
	Height      float64       `json:"height" def:"175.5"`
	IsActive    bool          `json:"is_active" def:"true"`
	Tags        []string      `json:"tags" def:"tag1,tag2,tag3"`
	Timeout     time.Duration `json:"timeout" def:"30s"`
	Description *string       `json:"description" def:"default description"`
	// 嵌套结构体
	Database DatabaseConfig `json:"database"`
	// 指针类型的嵌套结构体
	Cache *CacheConfig `json:"cache,omitempty"`
}

// 带有 map 字段的配置
type ConfigWithMap struct {
	Name        string        `json:"name" def:"default_name"`
	Age         int           `json:"age" def:"25"`
	Height      float64       `json:"height" def:"175.5"`
	IsActive    bool          `json:"is_active" def:"true"`
	Tags        []string      `json:"tags" def:"tag1,tag2,tag3"`
	Timeout     time.Duration `json:"timeout" def:"30s"`
	Description *string       `json:"description" def:"default description"`
	// Map 字段，用于测试 map 中新创建的结构体对象
	Services map[string]ServiceConfig `json:"services"`
}

// 带有切片字段的配置
type ConfigWithSlice struct {
	Name        string        `json:"name" def:"default_name"`
	Age         int           `json:"age" def:"25"`
	Height      float64       `json:"height" def:"175.5"`
	IsActive    bool          `json:"is_active" def:"true"`
	Tags        []string      `json:"tags" def:"tag1,tag2,tag3"`
	Timeout     time.Duration `json:"timeout" def:"30s"`
	Description *string       `json:"description" def:"default description"`
	Services []ServiceConfig `json:"services"`
}

// 完整的配置结构体，包含所有类型的字段
type FullConfig struct {
	Name        string        `json:"name" def:"default_name"`
	Age         int           `json:"age" def:"25"`
	Height      float64       `json:"height" def:"175.5"`
	IsActive    bool          `json:"is_active" def:"true"`
	Tags        []string      `json:"tags" def:"tag1,tag2,tag3"`
	Timeout     time.Duration `json:"timeout" def:"30s"`
	Description *string       `json:"description" def:"default description"`
	Database DatabaseConfig            `json:"database"`
	Cache    *CacheConfig              `json:"cache,omitempty"`
	Services map[string]ServiceConfig  `json:"services"`
	Workers  []WorkerConfig            `json:"workers"`
}

type DatabaseConfig struct {
	Host     string `json:"host" def:"localhost"`
	Port     int    `json:"port" def:"3306"`
	Username string `json:"username" def:"root"`
	Password string `json:"password" def:""`
}

type CacheConfig struct {
	Redis RedisConfig `json:"redis"`
}

type RedisConfig struct {
	Host string `json:"host" def:"redis-host"`
	Port int    `json:"port" def:"6379"`
}

type ServiceConfig struct {
	Host    string `json:"host" def:"service-host"`
	Port    int    `json:"port" def:"8080"`
	Enabled bool   `json:"enabled" def:"true"`
}

type WorkerConfig struct {
	Name        string `json:"name" def:"worker"`
	Concurrency int    `json:"concurrency" def:"10"`
	Enabled     bool   `json:"enabled" def:"true"`
}

// ============ MapStorage 测试 ============

// 测试 MapStorage 基本默认值功能
func TestMapStorage_BasicDefaults(t *testing.T) {
	data := map[string]interface{}{
		"age":    30, // 覆盖默认值
		"height": 180.0,
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &BasicConfig{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证基本类型默认值
	assert.Equal(t, "default_name", config.Name)                            // 使用默认值
	assert.Equal(t, 30, config.Age)                                         // 使用配置值
	assert.Equal(t, 180.0, config.Height)                                   // 使用配置值
	assert.Equal(t, true, config.IsActive)                                  // 使用默认值
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, config.Tags)          // 使用默认值
	assert.Equal(t, 30*time.Second, config.Timeout)                         // 使用默认值
	assert.NotNil(t, config.Description)                                    // 指针已分配
	assert.Equal(t, "default description", *config.Description)             // 使用默认值
}

// 测试 MapStorage 嵌套结构体默认值
func TestMapStorage_NestedDefaults(t *testing.T) {
	data := map[string]interface{}{
		"name": "custom_name", // 覆盖默认值
		"database": map[string]interface{}{
			"username": "admin", // 覆盖默认值，其他字段使用默认值
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &ConfigWithNested{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证基本字段
	assert.Equal(t, "custom_name", config.Name) // 配置值
	assert.Equal(t, 25, config.Age)             // 默认值

	// 验证嵌套结构体默认值
	assert.Equal(t, "localhost", config.Database.Host)     // 默认值
	assert.Equal(t, 3306, config.Database.Port)            // 默认值
	assert.Equal(t, "admin", config.Database.Username)     // 配置值
	assert.Equal(t, "", config.Database.Password)          // 默认值
}

// 测试 MapStorage 指针结构体默认值
func TestMapStorage_PointerDefaults(t *testing.T) {
	data := map[string]interface{}{
		"cache": map[string]interface{}{
			"redis": map[string]interface{}{
				"port": 7000, // 覆盖默认值
			},
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &ConfigWithNested{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证指针结构体的默认值设置
	assert.NotNil(t, config.Cache)                                    // 指针已分配
	assert.Equal(t, "redis-host", config.Cache.Redis.Host)            // 默认值
	assert.Equal(t, 7000, config.Cache.Redis.Port)                   // 配置值
}

// 测试 MapStorage map 字段中新创建结构体的默认值
func TestMapStorage_MapDefaults(t *testing.T) {
	data := map[string]interface{}{
		"services": map[string]interface{}{
			"api": map[string]interface{}{
				"port": 9000, // 覆盖默认值，其他使用默认值
			},
			"web": map[string]interface{}{}, // 空配置，全部使用默认值
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &ConfigWithMap{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

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

// 测试 MapStorage 切片字段中新创建结构体的默认值
func TestMapStorage_SliceDefaults(t *testing.T) {
	data := map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{
				"port": 9000, // 覆盖默认值，其他字段使用默认值
			},
			map[string]interface{}{
				"host":    "custom-service", // 覆盖默认值
				"enabled": false,            // 覆盖默认值
			},
			map[string]interface{}{}, // 空配置，全部使用默认值
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &ConfigWithSlice{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证切片长度
	assert.Len(t, config.Services, 3)

	// 第一个服务：port 使用配置值，其他使用默认值
	assert.Equal(t, "service-host", config.Services[0].Host) // 默认值
	assert.Equal(t, 9000, config.Services[0].Port)           // 配置值
	assert.Equal(t, true, config.Services[0].Enabled)        // 默认值

	// 第二个服务：host 和 enabled 使用配置值，port 使用默认值
	assert.Equal(t, "custom-service", config.Services[1].Host) // 配置值
	assert.Equal(t, 8080, config.Services[1].Port)             // 默认值
	assert.Equal(t, false, config.Services[1].Enabled)         // 配置值

	// 第三个服务：全部使用默认值
	assert.Equal(t, "service-host", config.Services[2].Host) // 默认值
	assert.Equal(t, 8080, config.Services[2].Port)           // 默认值
	assert.Equal(t, true, config.Services[2].Enabled)        // 默认值
}

// 测试 MapStorage 不启用默认值功能
func TestMapStorage_WithoutDefaults(t *testing.T) {
	data := map[string]interface{}{
		"age": 30,
	}

	ms := NewMapStorageWithoutDefaults(data)
	config := &BasicConfig{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证不设置默认值，所有字段保持零值
	assert.Equal(t, "", config.Name)                          // 零值
	assert.Equal(t, 30, config.Age)                           // 配置值
	assert.Equal(t, 0.0, config.Height)                       // 零值
	assert.Equal(t, false, config.IsActive)                   // 零值
	assert.Nil(t, config.Tags)                                // 零值
	assert.Equal(t, time.Duration(0), config.Timeout)         // 零值
	assert.Nil(t, config.Description)                         // 零值
}

// 测试 MapStorage 子存储继承默认值设置
func TestMapStorage_SubInheritsDefaults(t *testing.T) {
	data := map[string]interface{}{
		"database": map[string]interface{}{
			"username": "admin",
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	subStorage := ms.Sub("database")

	dbConfig := &DatabaseConfig{}
	err := subStorage.ConvertTo(dbConfig)
	assert.NoError(t, err)

	// 验证子存储继承了默认值设置
	assert.Equal(t, "localhost", dbConfig.Host)     // 默认值
	assert.Equal(t, 3306, dbConfig.Port)            // 默认值
	assert.Equal(t, "admin", dbConfig.Username)     // 配置值
	assert.Equal(t, "", dbConfig.Password)          // 默认值
}

// ============ FlatStorage 测试 ============

// 测试 FlatStorage 基本默认值功能
func TestFlatStorage_BasicDefaults(t *testing.T) {
	data := map[string]interface{}{
		"age":    30,
		"height": 180.0,
	}

	fs := NewFlatStorage(data).WithDefaults(true)
	config := &BasicConfig{}

	err := fs.ConvertTo(config)
	assert.NoError(t, err)

	// 验证基本类型默认值
	assert.Equal(t, "default_name", config.Name)                            // 使用默认值
	assert.Equal(t, 30, config.Age)                                         // 使用配置值
	assert.Equal(t, 180.0, config.Height)                                   // 使用配置值
	assert.Equal(t, true, config.IsActive)                                  // 使用默认值
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, config.Tags)          // 使用默认值
	assert.Equal(t, 30*time.Second, config.Timeout)                         // 使用默认值
	assert.NotNil(t, config.Description)                                    // 指针已分配
	assert.Equal(t, "default description", *config.Description)             // 使用默认值
}

// 测试 FlatStorage 嵌套结构体默认值
func TestFlatStorage_NestedDefaults(t *testing.T) {
	data := map[string]interface{}{
		"name":               "custom_name",
		"database.username":  "admin",
	}

	fs := NewFlatStorage(data).WithDefaults(true)
	config := &ConfigWithNested{}

	err := fs.ConvertTo(config)
	assert.NoError(t, err)

	// 验证基本字段
	assert.Equal(t, "custom_name", config.Name) // 配置值
	assert.Equal(t, 25, config.Age)             // 默认值

	// 验证嵌套结构体默认值
	assert.Equal(t, "localhost", config.Database.Host)     // 默认值
	assert.Equal(t, 3306, config.Database.Port)            // 默认值
	assert.Equal(t, "admin", config.Database.Username)     // 配置值
	assert.Equal(t, "", config.Database.Password)          // 默认值
}

// 测试 FlatStorage 指针结构体默认值
func TestFlatStorage_PointerDefaults(t *testing.T) {
	data := map[string]interface{}{
		"cache.redis.port": 7000, // 覆盖默认值
	}

	fs := NewFlatStorage(data).WithDefaults(true)
	config := &ConfigWithNested{}

	err := fs.ConvertTo(config)
	assert.NoError(t, err)

	// 验证指针结构体的默认值设置
	assert.NotNil(t, config.Cache)                                    // 指针已分配
	assert.Equal(t, "redis-host", config.Cache.Redis.Host)            // 默认值
	assert.Equal(t, 7000, config.Cache.Redis.Port)                   // 配置值
}

// 测试 FlatStorage 不启用默认值功能
func TestFlatStorage_WithoutDefaults(t *testing.T) {
	data := map[string]interface{}{
		"age": 30,
	}

	fs := NewFlatStorageWithoutDefaults(data)
	config := &BasicConfig{}

	err := fs.ConvertTo(config)
	assert.NoError(t, err)

	// 验证不设置默认值，所有字段保持零值
	assert.Equal(t, "", config.Name)                          // 零值
	assert.Equal(t, 30, config.Age)                           // 配置值
	assert.Equal(t, 0.0, config.Height)                       // 零值
	assert.Equal(t, false, config.IsActive)                   // 零值
}

// 测试 FlatStorage 子存储继承默认值设置
func TestFlatStorage_SubInheritsDefaults(t *testing.T) {
	data := map[string]interface{}{
		"database.username": "admin",
	}

	fs := NewFlatStorage(data).WithDefaults(true)
	subStorage := fs.Sub("database")

	dbConfig := &DatabaseConfig{}
	err := subStorage.ConvertTo(dbConfig)
	assert.NoError(t, err)

	// 验证子存储继承了默认值设置
	assert.Equal(t, "localhost", dbConfig.Host)     // 默认值
	assert.Equal(t, 3306, dbConfig.Port)            // 默认值
	assert.Equal(t, "admin", dbConfig.Username)     // 配置值
	assert.Equal(t, "", dbConfig.Password)          // 默认值
}

// ============ 一致性测试 ============

// 测试 MapStorage 和 FlatStorage 行为一致性
func TestStorageConsistency_BasicDefaults(t *testing.T) {
	// 相同的基础数据
	data := map[string]interface{}{
		"age":    35,
		"height": 170.0,
	}

	// MapStorage 测试
	ms := NewMapStorage(data).WithDefaults(true)
	mapConfig := &BasicConfig{}
	err := ms.ConvertTo(mapConfig)
	assert.NoError(t, err)

	// FlatStorage 测试
	fs := NewFlatStorage(data).WithDefaults(true)
	flatConfig := &BasicConfig{}
	err = fs.ConvertTo(flatConfig)
	assert.NoError(t, err)

	// 验证两者结果一致
	assert.Equal(t, mapConfig.Name, flatConfig.Name)
	assert.Equal(t, mapConfig.Age, flatConfig.Age)
	assert.Equal(t, mapConfig.Height, flatConfig.Height)
	assert.Equal(t, mapConfig.IsActive, flatConfig.IsActive)
	assert.Equal(t, mapConfig.Tags, flatConfig.Tags)
	assert.Equal(t, mapConfig.Timeout, flatConfig.Timeout)
	assert.Equal(t, *mapConfig.Description, *flatConfig.Description)
}

// 测试嵌套结构体的一致性
func TestStorageConsistency_NestedDefaults(t *testing.T) {
	// MapStorage 数据格式
	mapData := map[string]interface{}{
		"database": map[string]interface{}{
			"username": "admin",
			"port":     5432,
		},
	}

	// FlatStorage 数据格式
	flatData := map[string]interface{}{
		"database.username": "admin",
		"database.port":     5432,
	}

	// MapStorage 测试
	ms := NewMapStorage(mapData).WithDefaults(true)
	mapConfig := &ConfigWithNested{}
	err := ms.ConvertTo(mapConfig)
	assert.NoError(t, err)

	// FlatStorage 测试
	fs := NewFlatStorage(flatData).WithDefaults(true)
	flatConfig := &ConfigWithNested{}
	err = fs.ConvertTo(flatConfig)
	assert.NoError(t, err)

	// 验证嵌套结构体结果一致
	assert.Equal(t, mapConfig.Database.Host, flatConfig.Database.Host)
	assert.Equal(t, mapConfig.Database.Port, flatConfig.Database.Port)
	assert.Equal(t, mapConfig.Database.Username, flatConfig.Database.Username)
	assert.Equal(t, mapConfig.Database.Password, flatConfig.Database.Password)
}

// ============ 边界情况测试 ============

// 测试配置优先级：明确配置的零值应该保留，不被默认值覆盖
func TestConfigurationPriority(t *testing.T) {
	data := map[string]interface{}{
		"age":       0,     // 明确配置零值，应该保留
		"height":    0.0,   // 明确配置零值，应该保留
		"name":      "",    // 明确配置空字符串，应该保留
		"is_active": false, // 明确配置false，应该保留
		// timeout 字段不在配置中，应该使用默认值
	}

	// MapStorage 测试
	ms := NewMapStorage(data).WithDefaults(true)
	mapConfig := &BasicConfig{}
	err := ms.ConvertTo(mapConfig)
	assert.NoError(t, err)

	// 验证明确配置的零值被保留（配置优先于默认值）
	assert.Equal(t, "", mapConfig.Name)        // 配置值（空字符串）
	assert.Equal(t, 0, mapConfig.Age)          // 配置值（0）
	assert.Equal(t, 0.0, mapConfig.Height)     // 配置值（0.0）
	assert.Equal(t, false, mapConfig.IsActive) // 配置值（false）

	// 验证未在配置中的字段使用默认值
	assert.Equal(t, 30*time.Second, mapConfig.Timeout) // 默认值

	// FlatStorage 测试（验证一致性）
	fs := NewFlatStorage(data).WithDefaults(true)
	flatConfig := &BasicConfig{}
	err = fs.ConvertTo(flatConfig)
	assert.NoError(t, err)

	// 验证与 MapStorage 结果一致
	assert.Equal(t, mapConfig.Name, flatConfig.Name)
	assert.Equal(t, mapConfig.Age, flatConfig.Age)
	assert.Equal(t, mapConfig.Height, flatConfig.Height)
	assert.Equal(t, mapConfig.IsActive, flatConfig.IsActive)
	assert.Equal(t, mapConfig.Timeout, flatConfig.Timeout)
}

// 测试深层嵌套默认值
func TestDeepNestedDefaults(t *testing.T) {
	// 完全空的配置，验证多层嵌套的默认值设置
	data := map[string]interface{}{
		"cache": map[string]interface{}{
			// redis 字段为空，应该设置默认值
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &ConfigWithNested{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证所有基本字段都使用默认值
	assert.Equal(t, "default_name", config.Name)
	assert.Equal(t, 25, config.Age)
	assert.Equal(t, true, config.IsActive)
	assert.NotNil(t, config.Description)
	assert.Equal(t, "default description", *config.Description)

	// 验证嵌套结构体默认值
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 3306, config.Database.Port)
	assert.Equal(t, "root", config.Database.Username)
	assert.Equal(t, "", config.Database.Password)

	// 验证指针类型嵌套结构体的多层默认值
	assert.NotNil(t, config.Cache)
	assert.Equal(t, "redis-host", config.Cache.Redis.Host) // 深层嵌套默认值
	assert.Equal(t, 6379, config.Cache.Redis.Port)         // 深层嵌套默认值
}

// ============ 综合测试 ============

// 测试完整配置场景（包含所有类型的字段）
func TestFullConfiguration(t *testing.T) {
	data := map[string]interface{}{
		"name":   "production-app",
		"age":    100,
		"height": 200.0,
		"database": map[string]interface{}{
			"host":     "prod-db",
			"port":     5432,
			"username": "prod_user",
		},
		"cache": map[string]interface{}{
			"redis": map[string]interface{}{
				"host": "prod-redis",
			},
		},
		"services": map[string]interface{}{
			"api": map[string]interface{}{
				"host": "api-server",
				"port": 8080,
			},
			"worker": map[string]interface{}{
				"enabled": false,
			},
		},
		"workers": []interface{}{
			map[string]interface{}{
				"name": "worker1",
				"concurrency": 20,
			},
			map[string]interface{}{
				"enabled": false,
			},
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &FullConfig{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证基本配置
	assert.Equal(t, "production-app", config.Name)
	assert.Equal(t, 100, config.Age)
	assert.Equal(t, 200.0, config.Height)
	assert.Equal(t, true, config.IsActive) // 默认值

	// 验证数据库配置
	assert.Equal(t, "prod-db", config.Database.Host)
	assert.Equal(t, 5432, config.Database.Port)
	assert.Equal(t, "prod_user", config.Database.Username)
	assert.Equal(t, "", config.Database.Password) // 默认值

	// 验证缓存配置
	assert.NotNil(t, config.Cache)
	assert.Equal(t, "prod-redis", config.Cache.Redis.Host)
	assert.Equal(t, 6379, config.Cache.Redis.Port) // 默认值

	// 验证服务配置
	assert.NotNil(t, config.Services)
	
	apiService := config.Services["api"]
	assert.Equal(t, "api-server", apiService.Host)
	assert.Equal(t, 8080, apiService.Port)
	assert.Equal(t, true, apiService.Enabled) // 默认值

	workerService := config.Services["worker"]
	assert.Equal(t, "service-host", workerService.Host) // 默认值
	assert.Equal(t, 8080, workerService.Port)           // 默认值
	assert.Equal(t, false, workerService.Enabled)       // 配置值

	// 验证工作器配置
	assert.Len(t, config.Workers, 2)
	
	assert.Equal(t, "worker1", config.Workers[0].Name)
	assert.Equal(t, 20, config.Workers[0].Concurrency)
	assert.Equal(t, true, config.Workers[0].Enabled) // 默认值

	assert.Equal(t, "worker", config.Workers[1].Name)        // 默认值
	assert.Equal(t, 10, config.Workers[1].Concurrency)      // 默认值
	assert.Equal(t, false, config.Workers[1].Enabled)       // 配置值
}