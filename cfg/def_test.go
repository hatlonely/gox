package cfg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试用的结构体
type TestConfig struct {
	Name        string        `def:"default_name"`
	Age         int           `def:"25"`
	Height      float64       `def:"175.5"`
	IsActive    bool          `def:"true"`
	Tags        []string      `def:"tag1,tag2,tag3"`
	Timeout     time.Duration `def:"30s"`
	CreatedAt   time.Time     `def:"2023-01-01T00:00:00Z"`
	Description *string       `def:"default description"`
	
	// 嵌套结构体
	Database DefDatabaseConfig `def:""`
	
	// 指针类型的嵌套结构体
	Cache *CacheConfig `def:""`
}

type DefDatabaseConfig struct {
	Host     string `def:"localhost"`
	Port     int    `def:"3306"`
	Username string `def:"root"`
	Password string `def:""`
}

type CacheConfig struct {
	Redis RedisConfig `def:""`
}

type RedisConfig struct {
	Host string `def:"redis-host"`
	Port int    `def:"6379"`
}

func TestSetDefaults_BasicTypes(t *testing.T) {
	config := &TestConfig{}
	
	err := SetDefaults(config)
	assert.NoError(t, err)
	
	// 验证基本类型默认值
	assert.Equal(t, "default_name", config.Name)
	assert.Equal(t, 25, config.Age)
	assert.Equal(t, 175.5, config.Height)
	assert.Equal(t, true, config.IsActive)
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, config.Tags)
	
	// 验证时间类型默认值
	assert.Equal(t, 30*time.Second, config.Timeout)
	expectedTime, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	assert.Equal(t, expectedTime, config.CreatedAt)
	
	// 验证指针类型默认值
	assert.NotNil(t, config.Description)
	assert.Equal(t, "default description", *config.Description)
}

func TestSetDefaults_NestedStruct(t *testing.T) {
	config := &TestConfig{}
	
	err := SetDefaults(config)
	assert.NoError(t, err)
	
	// 验证嵌套结构体默认值
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 3306, config.Database.Port)
	assert.Equal(t, "root", config.Database.Username)
	assert.Equal(t, "", config.Database.Password)
}

func TestSetDefaults_PointerNestedStruct(t *testing.T) {
	config := &TestConfig{}
	
	err := SetDefaults(config)
	assert.NoError(t, err)
	
	// 验证指针类型的嵌套结构体
	assert.NotNil(t, config.Cache)
	assert.Equal(t, "redis-host", config.Cache.Redis.Host)
	assert.Equal(t, 6379, config.Cache.Redis.Port)
}

func TestSetDefaults_NonZeroValues(t *testing.T) {
	config := &TestConfig{
		Name: "existing_name",
		Age:  30,
	}
	
	err := SetDefaults(config)
	assert.NoError(t, err)
	
	// 已有值不应该被覆盖
	assert.Equal(t, "existing_name", config.Name)
	assert.Equal(t, 30, config.Age)
	
	// 零值字段应该被设置默认值
	assert.Equal(t, 175.5, config.Height)
	assert.Equal(t, true, config.IsActive)
}

func TestSetDefaults_InvalidInput(t *testing.T) {
	// 测试 nil 指针
	err := SetDefaults(nil)
	assert.Error(t, err)
	
	// 测试非指针类型
	config := TestConfig{}
	err = SetDefaults(config)
	assert.Error(t, err)
	
	// 测试 nil 对象
	var nilConfig *TestConfig
	err = SetDefaults(nilConfig)
	assert.Error(t, err)
}

func TestSetDefaults_TimeFormats(t *testing.T) {
	type TimeConfig struct {
		Time1 time.Time `def:"2023-01-01"`
		Time2 time.Time `def:"2023-01-01 15:04:05"`
		Time3 time.Time `def:"1672531200"` // Unix timestamp
		Time4 time.Time `def:"1672531200.5"` // Float timestamp
	}
	
	config := &TimeConfig{}
	err := SetDefaults(config)
	assert.NoError(t, err)
	
	expectedTime1, _ := time.Parse("2006-01-02", "2023-01-01")
	assert.Equal(t, expectedTime1, config.Time1)
	
	expectedTime2, _ := time.Parse("2006-01-02 15:04:05", "2023-01-01 15:04:05")
	assert.Equal(t, expectedTime2, config.Time2)
	
	expectedTime3 := time.Unix(1672531200, 0)
	assert.Equal(t, expectedTime3, config.Time3)
	
	expectedTime4 := time.Unix(1672531200, 500000000)
	assert.Equal(t, expectedTime4, config.Time4)
}

func TestSetDefaults_DurationFormats(t *testing.T) {
	type DurationConfig struct {
		Duration1 time.Duration `def:"1h30m"`
		Duration2 time.Duration `def:"5000000000"` // 纳秒
	}
	
	config := &DurationConfig{}
	err := SetDefaults(config)
	assert.NoError(t, err)
	
	assert.Equal(t, time.Hour+30*time.Minute, config.Duration1)
	assert.Equal(t, 5*time.Second, config.Duration2)
}

func TestSetDefaults_SliceTypes(t *testing.T) {
	type SliceConfig struct {
		StringSlice []string `def:"a,b,c"`
		IntSlice    []int    `def:"1,2,3"`
		FloatSlice  []float64 `def:"1.1,2.2,3.3"`
	}
	
	config := &SliceConfig{}
	err := SetDefaults(config)
	assert.NoError(t, err)
	
	assert.Equal(t, []string{"a", "b", "c"}, config.StringSlice)
	assert.Equal(t, []int{1, 2, 3}, config.IntSlice)
	assert.Equal(t, []float64{1.1, 2.2, 3.3}, config.FloatSlice)
}