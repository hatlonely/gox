package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试用的结构体
type TestConfigWithDefaults struct {
	Name        string        `json:"name" def:"default_name"`
	Age         int           `json:"age" def:"25"`
	Height      float64       `json:"height" def:"175.5"`
	IsActive    bool          `json:"isActive" def:"true"`
	Tags        []string      `json:"tags" def:"tag1,tag2,tag3"`
	Timeout     time.Duration `json:"timeout" def:"30s"`
	Description *string       `json:"description" def:"default description"`
	
	// 嵌套结构体
	Database DatabaseConfigWithDefaults `json:"database"`
	
	// 指针类型的嵌套结构体
	Cache *CacheConfigWithDefaults `json:"cache"`
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

func TestMapStorage_WithDefaults_BasicUsage(t *testing.T) {
	// 配置数据
	data := map[string]interface{}{
		"name": "configured_name",  // 这个值应该覆盖默认值
		"age":  30,                 // 这个值应该覆盖默认值
		// height 没有配置，应该使用默认值 175.5
		// isActive 没有配置，应该使用默认值 true
	}
	
	storage := NewMapStorage(data)
	config := &TestConfigWithDefaults{}
	
	err := storage.ConvertTo(config)
	assert.NoError(t, err)
	
	// 验证配置值覆盖了默认值
	assert.Equal(t, "configured_name", config.Name)
	assert.Equal(t, 30, config.Age)
	
	// 验证默认值被设置
	assert.Equal(t, 175.5, config.Height)
	assert.Equal(t, true, config.IsActive)
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, config.Tags)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.NotNil(t, config.Description)
	assert.Equal(t, "default description", *config.Description)
	
	// 验证嵌套结构体的默认值
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 3306, config.Database.Port)
	assert.Equal(t, "root", config.Database.Username)
	assert.Equal(t, "", config.Database.Password)
	
	// Cache 是指针类型且没有配置，应该保持为 nil
	assert.Nil(t, config.Cache)
}

func TestMapStorage_WithDefaults_NestedStructOverride(t *testing.T) {
	// 配置数据包含嵌套结构体的部分配置
	data := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "configured_host",
			"port": 5432,
			// username 和 password 没有配置，应该使用默认值
		},
	}
	
	storage := NewMapStorage(data)
	config := &TestConfigWithDefaults{}
	
	err := storage.ConvertTo(config)
	assert.NoError(t, err)
	
	// 验证嵌套结构体中配置值覆盖了默认值
	assert.Equal(t, "configured_host", config.Database.Host)
	assert.Equal(t, 5432, config.Database.Port)
	
	// 验证嵌套结构体中的默认值
	assert.Equal(t, "root", config.Database.Username)
	assert.Equal(t, "", config.Database.Password)
	
	// 验证顶层的默认值
	assert.Equal(t, "default_name", config.Name)
	assert.Equal(t, 25, config.Age)
}

func TestMapStorage_WithDefaults_PointerStructCreation(t *testing.T) {
	// 配置数据包含指针结构体的配置
	data := map[string]interface{}{
		"cache": map[string]interface{}{
			"redis": map[string]interface{}{
				"host": "configured_redis_host",
				// port 没有配置，应该使用默认值
			},
		},
	}
	
	storage := NewMapStorage(data)
	config := &TestConfigWithDefaults{}
	
	err := storage.ConvertTo(config)
	assert.NoError(t, err)
	
	// 验证指针结构体被创建
	assert.NotNil(t, config.Cache)
	
	// 验证指针结构体中配置值覆盖了默认值
	assert.Equal(t, "configured_redis_host", config.Cache.Redis.Host)
	
	// 验证指针结构体中的默认值
	assert.Equal(t, 6379, config.Cache.Redis.Port)
	
	// 验证顶层的默认值
	assert.Equal(t, "default_name", config.Name)
	assert.Equal(t, 25, config.Age)
}

func TestMapStorage_WithDefaults_Disabled(t *testing.T) {
	// 配置数据
	data := map[string]interface{}{
		"name": "configured_name",
	}
	
	storage := NewMapStorage(data).WithDefaults(false)
	config := &TestConfigWithDefaults{}
	
	err := storage.ConvertTo(config)
	assert.NoError(t, err)
	
	// 验证配置值被设置
	assert.Equal(t, "configured_name", config.Name)
	
	// 验证默认值没有被设置（保持零值）
	assert.Equal(t, 0, config.Age)
	assert.Equal(t, 0.0, config.Height)
	assert.Equal(t, false, config.IsActive)
	assert.Nil(t, config.Tags)
	assert.Equal(t, time.Duration(0), config.Timeout)
	assert.Nil(t, config.Description)
	
	// 验证嵌套结构体的默认值没有被设置
	assert.Equal(t, "", config.Database.Host)
	assert.Equal(t, 0, config.Database.Port)
}

func TestMapStorage_WithDefaults_SubStorageInheritance(t *testing.T) {
	// 配置数据
	data := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "configured_host",
		},
	}
	
	storage := NewMapStorage(data)
	subStorage := storage.Sub("database")
	
	// 子存储应该继承父存储的默认值设置
	subMapStorage, ok := subStorage.(*MapStorage)
	assert.True(t, ok)
	assert.True(t, subMapStorage.enableDefaults)
	
	// 测试子存储的转换
	dbConfig := &DatabaseConfigWithDefaults{}
	err := subStorage.ConvertTo(dbConfig)
	assert.NoError(t, err)
	
	// 验证配置值和默认值
	assert.Equal(t, "configured_host", dbConfig.Host)
	assert.Equal(t, 3306, dbConfig.Port)  // 默认值
	assert.Equal(t, "root", dbConfig.Username)  // 默认值
	assert.Equal(t, "", dbConfig.Password)  // 默认值
}

func TestMapStorage_WithDefaults_PriorityOrder(t *testing.T) {
	// 这个测试验证配置数据优先级高于默认值
	data := map[string]interface{}{
		"name":        "config_name",
		"age":         35,
		"height":      180.0,
		"isActive":    false,
		"tags":        []string{"config_tag"},
		"timeout":     "60s",
		"description": "config description",
	}
	
	storage := NewMapStorage(data)
	config := &TestConfigWithDefaults{}
	
	err := storage.ConvertTo(config)
	assert.NoError(t, err)
	
	// 所有字段都应该使用配置值，而不是默认值
	assert.Equal(t, "config_name", config.Name)
	assert.Equal(t, 35, config.Age)
	assert.Equal(t, 180.0, config.Height)
	assert.Equal(t, false, config.IsActive)
	assert.Equal(t, []string{"config_tag"}, config.Tags)
	assert.Equal(t, 60*time.Second, config.Timeout)
	assert.NotNil(t, config.Description)
	assert.Equal(t, "config description", *config.Description)
}