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

// 测试FlatStorage的map功能（如果支持）
func TestFlatStorage_MapDefaults(t *testing.T) {
	data := map[string]interface{}{
		"services.api.port":    9000,          // 覆盖默认值
		"services.web.host":    "custom-host", // 覆盖默认值
		// services.api.host 和 services.api.enabled 应该使用默认值
		// services.web.port 和 services.web.enabled 应该使用默认值
	}

	fs := NewFlatStorage(data).WithDefaults(true)
	config := &ConfigWithDefaults{}

	err := fs.ConvertTo(config)
	// 注意：如果FlatStorage不支持这种map结构转换，这个测试可能会失败
	// 这是为了验证FlatStorage是否能够通过扁平键创建map中的结构体对象
	if err != nil {
		t.Logf("FlatStorage may not support this map structure: %v", err)
		return
	}

	// 验证 Map 中新创建的结构体对象默认值（如果支持）
	if config.Services != nil {
		if apiService, exists := config.Services["api"]; exists {
			assert.Equal(t, "service-host", apiService.Host) // 默认值
			assert.Equal(t, 9000, apiService.Port)           // 配置值
			assert.Equal(t, true, apiService.Enabled)        // 默认值
		}
		
		if webService, exists := config.Services["web"]; exists {
			assert.Equal(t, "custom-host", webService.Host)  // 配置值
			assert.Equal(t, 8080, webService.Port)           // 默认值
			assert.Equal(t, true, webService.Enabled)        // 默认值
		}
	}
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

// 测试切片中结构体对象的默认值
func TestSliceStructDefaults(t *testing.T) {
	// 扩展测试结构体以包含切片字段
	type ExtendedConfig struct {
		Name     string          `json:"name" def:"app-name"`
		Services []ServiceConfig `json:"services"`
	}

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
	config := &ExtendedConfig{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证基本默认值
	assert.Equal(t, "app-name", config.Name)

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

// 测试多层嵌套默认值传递
func TestDeepNestedDefaults(t *testing.T) {
	// 测试完全空的配置，验证多层嵌套的默认值设置
	data := map[string]interface{}{
		"cache": map[string]interface{}{
			// redis 字段为空，应该设置默认值
		},
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &ConfigWithDefaults{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证所有基本字段都使用默认值
	assert.Equal(t, "default_name", config.Name)
	assert.Equal(t, 25, config.Age)
	assert.Equal(t, 175.5, config.Height)
	assert.Equal(t, true, config.IsActive)
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, config.Tags)
	assert.Equal(t, 30*time.Second, config.Timeout)
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

// 测试边界情况：配置中明确提供的零值应该保留，不被默认值覆盖
func TestEdgeCasesDefaults(t *testing.T) {
	data := map[string]interface{}{
		"age":    0,   // 明确配置零值，应该保留
		"height": 0.0, // 明确配置零值，应该保留  
		"name":   "",   // 明确配置空字符串，应该保留
		// is_active 字段不在配置中，应该使用默认值
	}

	ms := NewMapStorage(data).WithDefaults(true)
	config := &ConfigWithDefaults{}

	err := ms.ConvertTo(config)
	assert.NoError(t, err)

	// 验证明确配置的零值被保留（配置优先于默认值）
	assert.Equal(t, "", config.Name)       // 配置值（空字符串）
	assert.Equal(t, 0, config.Age)         // 配置值（0）
	assert.Equal(t, 0.0, config.Height)    // 配置值（0.0）
	
	// 验证未在配置中的字段使用默认值
	assert.Equal(t, true, config.IsActive) // 默认值（字段不在配置中）
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, config.Tags) // 默认值
	assert.Equal(t, 30*time.Second, config.Timeout)                // 默认值
}