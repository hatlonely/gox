package def

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
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
	Database DefDatabaseConfig
	
	// 指针类型的嵌套结构体
	Cache *CacheConfig
}

type DefDatabaseConfig struct {
	Host     string `def:"localhost"`
	Port     int    `def:"3306"`
	Username string `def:"root"`
	Password string `def:""`
}

type CacheConfig struct {
	Redis RedisConfig
}

type RedisConfig struct {
	Host string `def:"redis-host"`
	Port int    `def:"6379"`
}

func TestSetDefaults_BasicTypes(t *testing.T) {
	Convey("设置基本类型默认值", t, func() {
		config := &TestConfig{}
		
		err := SetDefaults(config)
		So(err, ShouldBeNil)
		
		Convey("验证基本类型默认值", func() {
			So(config.Name, ShouldEqual, "default_name")
			So(config.Age, ShouldEqual, 25)
			So(config.Height, ShouldEqual, 175.5)
			So(config.IsActive, ShouldBeTrue)
			So(config.Tags, ShouldResemble, []string{"tag1", "tag2", "tag3"})
		})
		
		Convey("验证时间类型默认值", func() {
			So(config.Timeout, ShouldEqual, 30*time.Second)
			expectedTime, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
			So(config.CreatedAt, ShouldEqual, expectedTime)
		})
		
		Convey("验证指针类型默认值", func() {
			So(config.Description, ShouldNotBeNil)
			So(*config.Description, ShouldEqual, "default description")
		})
	})
}

func TestSetDefaults_NestedStruct(t *testing.T) {
	Convey("设置嵌套结构体默认值", t, func() {
		config := &TestConfig{}
		
		err := SetDefaults(config)
		So(err, ShouldBeNil)
		
		Convey("验证嵌套结构体默认值", func() {
			So(config.Database.Host, ShouldEqual, "localhost")
			So(config.Database.Port, ShouldEqual, 3306)
			So(config.Database.Username, ShouldEqual, "root")
			So(config.Database.Password, ShouldEqual, "")
		})
	})
}

func TestSetDefaults_PointerNestedStruct(t *testing.T) {
	Convey("处理指针类型的嵌套结构体", t, func() {
		config := &TestConfig{}
		
		err := SetDefaults(config)
		So(err, ShouldBeNil)
		
		Convey("Cache是指针类型且为空，应该保持为nil（不会自动创建）", func() {
			So(config.Cache, ShouldBeNil)
		})
	})
}

func TestSetDefaults_PointerNestedStructNotNil(t *testing.T) {
	Convey("处理非空指针类型的嵌套结构体", t, func() {
		config := &TestConfig{
			Cache: &CacheConfig{}, // 手动初始化指针
		}
		
		err := SetDefaults(config)
		So(err, ShouldBeNil)
		
		Convey("验证指针类型的嵌套结构体，当指针不为空时会递归处理", func() {
			So(config.Cache, ShouldNotBeNil)
			So(config.Cache.Redis.Host, ShouldEqual, "redis-host")
			So(config.Cache.Redis.Port, ShouldEqual, 6379)
		})
	})
}

func TestSetDefaults_NonZeroValues(t *testing.T) {
	Convey("处理非零值字段", t, func() {
		config := &TestConfig{
			Name: "existing_name",
			Age:  30,
		}
		
		err := SetDefaults(config)
		So(err, ShouldBeNil)
		
		Convey("已有值不应该被覆盖", func() {
			So(config.Name, ShouldEqual, "existing_name")
			So(config.Age, ShouldEqual, 30)
		})
		
		Convey("零值字段应该被设置默认值", func() {
			So(config.Height, ShouldEqual, 175.5)
			So(config.IsActive, ShouldBeTrue)
		})
	})
}

func TestSetDefaults_InvalidInput(t *testing.T) {
	Convey("测试无效输入", t, func() {
		Convey("测试nil指针", func() {
			err := SetDefaults(nil)
			So(err, ShouldNotBeNil)
		})
		
		Convey("测试非指针类型", func() {
			config := TestConfig{}
			err := SetDefaults(config)
			So(err, ShouldNotBeNil)
		})
		
		Convey("测试nil对象", func() {
			var nilConfig *TestConfig
			err := SetDefaults(nilConfig)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSetDefaults_TimeFormats(t *testing.T) {
	Convey("测试时间格式", t, func() {
		type TimeConfig struct {
			Time1 time.Time `def:"2023-01-01"`
			Time2 time.Time `def:"2023-01-01 15:04:05"`
			Time3 time.Time `def:"1672531200"` // Unix timestamp
			Time4 time.Time `def:"1672531200.5"` // Float timestamp
		}
		
		config := &TimeConfig{}
		err := SetDefaults(config)
		So(err, ShouldBeNil)
		
		Convey("验证各种时间格式解析", func() {
			expectedTime1, _ := time.Parse("2006-01-02", "2023-01-01")
			So(config.Time1, ShouldEqual, expectedTime1)
			
			expectedTime2, _ := time.Parse("2006-01-02 15:04:05", "2023-01-01 15:04:05")
			So(config.Time2, ShouldEqual, expectedTime2)
			
			expectedTime3 := time.Unix(1672531200, 0)
			So(config.Time3, ShouldEqual, expectedTime3)
			
			expectedTime4 := time.Unix(1672531200, 500000000)
			So(config.Time4, ShouldEqual, expectedTime4)
		})
	})
}

func TestSetDefaults_DurationFormats(t *testing.T) {
	Convey("测试时间间隔格式", t, func() {
		type DurationConfig struct {
			Duration1 time.Duration `def:"1h30m"`
			Duration2 time.Duration `def:"5000000000"` // 纳秒
		}
		
		config := &DurationConfig{}
		err := SetDefaults(config)
		So(err, ShouldBeNil)
		
		Convey("验证各种时间间隔格式解析", func() {
			So(config.Duration1, ShouldEqual, time.Hour+30*time.Minute)
			So(config.Duration2, ShouldEqual, 5*time.Second)
		})
	})
}

func TestSetDefaults_SliceTypes(t *testing.T) {
	Convey("测试切片类型", t, func() {
		type SliceConfig struct {
			StringSlice []string `def:"a,b,c"`
			IntSlice    []int    `def:"1,2,3"`
			FloatSlice  []float64 `def:"1.1,2.2,3.3"`
		}
		
		config := &SliceConfig{}
		err := SetDefaults(config)
		So(err, ShouldBeNil)
		
		Convey("验证各种切片类型默认值", func() {
			So(config.StringSlice, ShouldResemble, []string{"a", "b", "c"})
			So(config.IntSlice, ShouldResemble, []int{1, 2, 3})
			So(config.FloatSlice, ShouldResemble, []float64{1.1, 2.2, 3.3})
		})
	})
}