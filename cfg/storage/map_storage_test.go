package storage

import (
	"reflect"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

// 测试数据集
var testData = map[string]interface{}{
	"database": map[string]interface{}{
		"host": "localhost",
		"port": 3306,
		"connections": []interface{}{
			map[string]interface{}{
				"name": "primary",
				"user": "admin",
			},
			map[string]interface{}{
				"name": "secondary",
				"user": "readonly",
			},
		},
	},
	"servers": []interface{}{"server1", "server2"},
	"config": map[string]interface{}{
		"timeout":    "30s",
		"created_at": "2023-12-25T15:30:45Z",
		"enabled":    true,
	},
}

// TestMapStorage_Creation 测试 MapStorage 的各种创建方式
func TestMapStorage_Creation(t *testing.T) {
	Convey("MapStorage 创建测试", t, func() {
		Convey("创建带默认值的 MapStorage", func() {
			storage := NewMapStorage(testData)
			So(storage, ShouldNotBeNil)
			So(storage.Data(), ShouldResemble, testData)
			So(storage.enableDefaults, ShouldBeTrue)
		})

		Convey("创建不带默认值的 MapStorage", func() {
			storage := NewMapStorage(testData).WithDefaults(false)
			So(storage, ShouldNotBeNil)
			So(storage.Data(), ShouldResemble, testData)
			So(storage.enableDefaults, ShouldBeFalse)
		})

		Convey("创建空数据的 MapStorage", func() {
			storage := NewMapStorage(nil)
			So(storage, ShouldNotBeNil)
			So(storage.Data(), ShouldBeNil)
			So(storage.enableDefaults, ShouldBeTrue)
		})

		Convey("创建空 map 的 MapStorage", func() {
			emptyMap := map[string]interface{}{}
			storage := NewMapStorage(emptyMap)
			So(storage, ShouldNotBeNil)
			So(storage.Data(), ShouldResemble, emptyMap)
			So(storage.enableDefaults, ShouldBeTrue)
		})

		Convey("创建简单类型的 MapStorage", func() {
			simpleData := "simple string"
			storage := NewMapStorage(simpleData)
			So(storage, ShouldNotBeNil)
			So(storage.Data(), ShouldEqual, simpleData)
			So(storage.enableDefaults, ShouldBeTrue)
		})
	})
}

// TestMapStorage_WithDefaults 测试默认值开关功能
func TestMapStorage_WithDefaults(t *testing.T) {
	Convey("MapStorage 默认值开关测试", t, func() {
		storage := NewMapStorage(testData)

		Convey("开启默认值", func() {
			result := storage.WithDefaults(true)
			So(result.enableDefaults, ShouldBeTrue)
		})

		Convey("关闭默认值", func() {
			result := storage.WithDefaults(false)
			So(result.enableDefaults, ShouldBeFalse)
		})

		Convey("nil storage 的处理", func() {
			var nilStorage *MapStorage = nil
			result := nilStorage.WithDefaults(true)
			So(result, ShouldBeNil)
		})
	})
}

// TestMapStorage_Data 测试数据获取功能
func TestMapStorage_Data(t *testing.T) {
	Convey("MapStorage 数据获取测试", t, func() {
		Convey("map 数据", func() {
			storage := NewMapStorage(testData)
			So(storage.Data(), ShouldResemble, testData)
		})

		Convey("nil 数据", func() {
			storage := NewMapStorage(nil)
			So(storage.Data(), ShouldBeNil)
		})

		Convey("字符串数据", func() {
			data := "test"
			storage := NewMapStorage(data)
			So(storage.Data(), ShouldEqual, data)
		})

		Convey("数字数据", func() {
			data := 123
			storage := NewMapStorage(data)
			So(storage.Data(), ShouldEqual, data)
		})

		Convey("切片数据", func() {
			data := []string{"a", "b", "c"}
			storage := NewMapStorage(data)
			So(storage.Data(), ShouldResemble, data)
		})
	})
}

// TestMapStorage_Sub_Basic 测试基本路径访问功能
func TestMapStorage_Sub_Basic(t *testing.T) {
	Convey("MapStorage 基本路径访问测试", t, func() {
		storage := NewMapStorage(testData)

		Convey("空key返回自身", func() {
			result := storage.Sub("")
			So(result, ShouldNotBeNil)

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			So(err, ShouldBeNil)
			So(actualData, ShouldResemble, testData)
		})

		Convey("简单字段访问", func() {
			result := storage.Sub("servers")
			So(result, ShouldNotBeNil)

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			So(err, ShouldBeNil)
			So(actualData, ShouldResemble, []interface{}{"server1", "server2"})
		})

		Convey("嵌套map访问", func() {
			result := storage.Sub("database")
			So(result, ShouldNotBeNil)

			expected := map[string]interface{}{
				"host": "localhost",
				"port": 3306,
				"connections": []interface{}{
					map[string]interface{}{"name": "primary", "user": "admin"},
					map[string]interface{}{"name": "secondary", "user": "readonly"},
				},
			}

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			So(err, ShouldBeNil)
			So(actualData, ShouldResemble, expected)
		})

		Convey("不存在的key", func() {
			result := storage.Sub("nonexistent")

			// 检查返回的是否是 nil MapStorage
			ms, ok := result.(*MapStorage)
			So(ok, ShouldBeTrue)
			So(ms, ShouldBeNil)
		})
	})
}

// TestMapStorage_Sub_NestedPath 测试嵌套路径访问
func TestMapStorage_Sub_NestedPath(t *testing.T) {
	Convey("MapStorage 嵌套路径访问测试", t, func() {
		storage := NewMapStorage(testData)

		Convey("两级嵌套访问", func() {
			result := storage.Sub("database.host")
			So(result, ShouldNotBeNil)

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			So(err, ShouldBeNil)
			So(actualData, ShouldEqual, "localhost")
		})

		Convey("两级嵌套数字访问", func() {
			result := storage.Sub("database.port")
			So(result, ShouldNotBeNil)

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			So(err, ShouldBeNil)
			So(actualData, ShouldEqual, 3306)
		})

		Convey("三级嵌套访问", func() {
			result := storage.Sub("config.timeout")
			So(result, ShouldNotBeNil)

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			So(err, ShouldBeNil)
			So(actualData, ShouldEqual, "30s")
		})

		Convey("不存在的嵌套路径", func() {
			result := storage.Sub("database.nonexistent")

			ms, ok := result.(*MapStorage)
			So(ok, ShouldBeTrue)
			So(ms, ShouldBeNil)
		})

		Convey("部分存在的路径", func() {
			result := storage.Sub("nonexistent.field")

			ms, ok := result.(*MapStorage)
			So(ok, ShouldBeTrue)
			So(ms, ShouldBeNil)
		})
	})
}

// TestMapStorage_Sub_ArrayIndex 测试数组索引访问
func TestMapStorage_Sub_ArrayIndex(t *testing.T) {
	Convey("MapStorage 数组索引访问测试", t, func() {
		storage := NewMapStorage(testData)

		Convey("数组第一个元素", func() {
			result := storage.Sub("servers[0]")
			So(result, ShouldNotBeNil)

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			So(err, ShouldBeNil)
			So(actualData, ShouldEqual, "server1")
		})

		Convey("数组第二个元素", func() {
			result := storage.Sub("servers[1]")
			So(result, ShouldNotBeNil)

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			So(err, ShouldBeNil)
			So(actualData, ShouldEqual, "server2")
		})

		Convey("数组越界访问", func() {
			result := storage.Sub("servers[2]")

			ms, ok := result.(*MapStorage)
			So(ok, ShouldBeTrue)
			So(ms, ShouldBeNil)
		})

		Convey("负数索引", func() {
			result := storage.Sub("servers[-1]")

			ms, ok := result.(*MapStorage)
			So(ok, ShouldBeTrue)
			So(ms, ShouldBeNil)
		})

		Convey("非数字索引", func() {
			result := storage.Sub("servers[abc]")

			ms, ok := result.(*MapStorage)
			So(ok, ShouldBeTrue)
			So(ms, ShouldBeNil)
		})
	})
}

// TestMapStorage_Sub_ComplexPath 测试复杂路径访问
func TestMapStorage_Sub_ComplexPath(t *testing.T) {
	Convey("MapStorage 复杂路径访问测试", t, func() {
		storage := NewMapStorage(testData)

		Convey("数组元素的字段", func() {
			result := storage.Sub("database.connections[0].name")
			So(result, ShouldNotBeNil)

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			So(err, ShouldBeNil)
			So(actualData, ShouldEqual, "primary")
		})

		Convey("数组第二个元素的字段", func() {
			result := storage.Sub("database.connections[1].user")
			So(result, ShouldNotBeNil)

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			So(err, ShouldBeNil)
			So(actualData, ShouldEqual, "readonly")
		})

		Convey("数组越界的字段访问", func() {
			result := storage.Sub("database.connections[2].name")

			ms, ok := result.(*MapStorage)
			So(ok, ShouldBeTrue)
			So(ms, ShouldBeNil)
		})

		Convey("数组元素不存在的字段", func() {
			result := storage.Sub("database.connections[0].nonexistent")

			ms, ok := result.(*MapStorage)
			So(ok, ShouldBeTrue)
			So(ms, ShouldBeNil)
		})
	})
}

// TestMapStorage_Sub_DefaultsInheritance 测试子Storage的默认值继承
func TestMapStorage_Sub_DefaultsInheritance(t *testing.T) {
	Convey("MapStorage 子Storage默认值继承测试", t, func() {
		Convey("带默认值的Storage", func() {
			storageWithDefaults := NewMapStorage(testData)
			sub1 := storageWithDefaults.Sub("database")

			So(sub1, ShouldNotBeNil)

			subMS := sub1.(*MapStorage)
			So(subMS.enableDefaults, ShouldBeTrue)
		})

		Convey("不带默认值的Storage", func() {
			storageWithoutDefaults := NewMapStorage(testData).WithDefaults(false)
			sub2 := storageWithoutDefaults.Sub("database")

			So(sub2, ShouldNotBeNil)

			subMS2 := sub2.(*MapStorage)
			So(subMS2.enableDefaults, ShouldBeFalse)
		})
	})
}

// deepEqual 深度比较两个值是否相等
func deepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

// TestMapStorage_ConvertTo_BasicTypes 测试基本类型转换
func TestMapStorage_ConvertTo_BasicTypes(t *testing.T) {
	Convey("MapStorage 基本类型转换测试", t, func() {
		Convey("字符串转换", func() {
			storage := NewMapStorage("hello world")
			target := new(string)
			err := storage.ConvertTo(target)

			So(err, ShouldBeNil)
			actual := reflect.ValueOf(target).Elem().Interface()
			So(actual, ShouldEqual, "hello world")
		})

		Convey("整数转换", func() {
			storage := NewMapStorage(42)
			target := new(int)
			err := storage.ConvertTo(target)

			So(err, ShouldBeNil)
			actual := reflect.ValueOf(target).Elem().Interface()
			So(actual, ShouldEqual, 42)
		})

		Convey("浮点数转换", func() {
			storage := NewMapStorage(3.14)
			target := new(float64)
			err := storage.ConvertTo(target)

			So(err, ShouldBeNil)
			actual := reflect.ValueOf(target).Elem().Interface()
			So(actual, ShouldEqual, 3.14)
		})

		Convey("布尔值转换", func() {
			storage := NewMapStorage(true)
			target := new(bool)
			err := storage.ConvertTo(target)

			So(err, ShouldBeNil)
			actual := reflect.ValueOf(target).Elem().Interface()
			So(actual, ShouldEqual, true)
		})
	})
}

// TestMapStorage_ConvertTo_Struct 测试结构体转换
func TestMapStorage_ConvertTo_Struct(t *testing.T) {
	Convey("MapStorage 结构体转换测试", t, func() {
		type ServerConfig struct {
			Name    string `json:"name"`
			Port    int    `json:"port"`
			Enabled bool   `json:"enabled"`
		}

		Convey("结构体转换", func() {
			data := map[string]interface{}{
				"name":    "test-server",
				"port":    8080,
				"enabled": true,
			}

			storage := NewMapStorage(data)
			var config ServerConfig
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)
			So(config.Name, ShouldEqual, "test-server")
			So(config.Port, ShouldEqual, 8080)
			So(config.Enabled, ShouldBeTrue)
		})
	})
}

// TestMapStorage_ConvertTo_Slice 测试切片转换
func TestMapStorage_ConvertTo_Slice(t *testing.T) {
	Convey("MapStorage 切片转换测试", t, func() {
		Convey("字符串切片", func() {
			data := []interface{}{"item1", "item2", "item3"}
			target := &[]string{}
			storage := NewMapStorage(data)
			err := storage.ConvertTo(target)

			So(err, ShouldBeNil)
			actual := reflect.ValueOf(target).Elem().Interface()
			So(deepEqual(actual, []string{"item1", "item2", "item3"}), ShouldBeTrue)
		})

		Convey("整数切片", func() {
			data := []interface{}{1, 2, 3}
			target := &[]int{}
			storage := NewMapStorage(data)
			err := storage.ConvertTo(target)

			So(err, ShouldBeNil)
			actual := reflect.ValueOf(target).Elem().Interface()
			So(deepEqual(actual, []int{1, 2, 3}), ShouldBeTrue)
		})

		Convey("空切片", func() {
			data := []interface{}{}
			target := &[]string{}
			storage := NewMapStorage(data)
			err := storage.ConvertTo(target)

			So(err, ShouldBeNil)
			actual := reflect.ValueOf(target).Elem().Interface()
			So(deepEqual(actual, []string{}), ShouldBeTrue)
		})
	})
}

// TestMapStorage_ConvertTo_Map 测试Map转换
func TestMapStorage_ConvertTo_Map(t *testing.T) {
	Convey("MapStorage Map转换测试", t, func() {
		Convey("转换为 map[string]interface{}", func() {
			data := map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
				"key3": 123,
			}

			storage := NewMapStorage(data)
			var result1 map[string]interface{}
			err := storage.ConvertTo(&result1)

			So(err, ShouldBeNil)
			So(deepEqual(result1, data), ShouldBeTrue)
		})

		Convey("转换为 map[string]string", func() {
			stringData := map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			}
			storage2 := NewMapStorage(stringData)
			var result2 map[string]string
			err := storage2.ConvertTo(&result2)

			So(err, ShouldBeNil)
			expected := map[string]string{
				"key1": "value1",
				"key2": "value2",
			}
			So(deepEqual(result2, expected), ShouldBeTrue)
		})
	})
}

// TestMapStorage_ConvertTo_Time 测试时间类型转换
func TestMapStorage_ConvertTo_Time(t *testing.T) {
	Convey("MapStorage 时间类型转换测试", t, func() {
		Convey("RFC3339 字符串", func() {
			storage := NewMapStorage("2023-12-25T15:30:45Z")
			var timeValue time.Time
			err := storage.ConvertTo(&timeValue)

			So(err, ShouldBeNil)
			expected := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
			So(timeValue.Equal(expected), ShouldBeTrue)
		})

		Convey("日期字符串", func() {
			storage := NewMapStorage("2023-12-25")
			var timeValue time.Time
			err := storage.ConvertTo(&timeValue)

			So(err, ShouldBeNil)
			expected := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
			So(timeValue.Equal(expected), ShouldBeTrue)
		})

		Convey("日期时间字符串", func() {
			storage := NewMapStorage("2023-12-25 15:30:45")
			var timeValue time.Time
			err := storage.ConvertTo(&timeValue)

			So(err, ShouldBeNil)
			expected := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
			So(timeValue.Equal(expected), ShouldBeTrue)
		})

		Convey("Unix时间戳整数", func() {
			storage := NewMapStorage(int64(1703517045))
			var timeValue time.Time
			err := storage.ConvertTo(&timeValue)

			So(err, ShouldBeNil)
			expected := time.Unix(1703517045, 0)
			So(timeValue.Equal(expected), ShouldBeTrue)
		})

		Convey("Unix时间戳浮点数", func() {
			storage := NewMapStorage(1703517045.5)
			var timeValue time.Time
			err := storage.ConvertTo(&timeValue)

			So(err, ShouldBeNil)
			expected := time.Unix(1703517045, 500000000)
			So(timeValue.Equal(expected), ShouldBeTrue)
		})
	})
}

// TestMapStorage_ConvertTo_Duration 测试Duration类型转换
func TestMapStorage_ConvertTo_Duration(t *testing.T) {
	Convey("MapStorage Duration类型转换测试", t, func() {
		Convey("字符串Duration", func() {
			storage := NewMapStorage("5m30s")
			var duration time.Duration
			err := storage.ConvertTo(&duration)

			So(err, ShouldBeNil)
			So(duration, ShouldEqual, 5*time.Minute+30*time.Second)
		})

		Convey("小时Duration", func() {
			storage := NewMapStorage("2h15m")
			var duration time.Duration
			err := storage.ConvertTo(&duration)

			So(err, ShouldBeNil)
			So(duration, ShouldEqual, 2*time.Hour+15*time.Minute)
		})

		Convey("纳秒整数", func() {
			storage := NewMapStorage(int64(1000000000))
			var duration time.Duration
			err := storage.ConvertTo(&duration)

			So(err, ShouldBeNil)
			So(duration, ShouldEqual, time.Second)
		})

		Convey("秒浮点数", func() {
			storage := NewMapStorage(2.5)
			var duration time.Duration
			err := storage.ConvertTo(&duration)

			So(err, ShouldBeNil)
			So(duration, ShouldEqual, 2*time.Second+500*time.Millisecond)
		})
	})
}

// TestMapStorage_ConvertTo_StructWithTags 测试带标签的结构体转换
func TestMapStorage_ConvertTo_StructWithTags(t *testing.T) {
	Convey("MapStorage 带标签的结构体转换测试", t, func() {
		type TestConfig struct {
			Field1 string `cfg:"custom_name" json:"json_name"`
			Field2 string `json:"json_field"`
			Field3 string `yaml:"yaml_field"`
			Field4 string `toml:"toml_field"`
			Field5 string `ini:"ini_field"`
			Field6 string // 无标签，使用字段名
		}

		Convey("各种标签的字段映射", func() {
			data := map[string]interface{}{
				"custom_name": "value1",
				"json_field":  "value2",
				"yaml_field":  "value3",
				"toml_field":  "value4",
				"ini_field":   "value5",
				"Field6":      "value6",
			}

			storage := NewMapStorage(data)
			var config TestConfig
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)
			So(config.Field1, ShouldEqual, "value1")
			So(config.Field2, ShouldEqual, "value2")
			So(config.Field3, ShouldEqual, "value3")
			So(config.Field4, ShouldEqual, "value4")
			So(config.Field5, ShouldEqual, "value5")
			So(config.Field6, ShouldEqual, "value6")
		})
	})
}

// TestMapStorage_ConvertTo_NestedStruct 测试嵌套结构体转换
func TestMapStorage_ConvertTo_NestedStruct(t *testing.T) {
	Convey("MapStorage 嵌套结构体转换测试", t, func() {
		type DatabaseConfig struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		}

		type AppConfig struct {
			Name     string         `json:"name"`
			Database DatabaseConfig `json:"database"`
			Servers  []string       `json:"servers"`
		}

		Convey("嵌套结构体和数组转换", func() {
			data := map[string]interface{}{
				"name": "test-app",
				"database": map[string]interface{}{
					"host": "localhost",
					"port": 3306,
				},
				"servers": []interface{}{"server1", "server2"},
			}

			storage := NewMapStorage(data)
			var config AppConfig
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)
			So(config.Name, ShouldEqual, "test-app")
			So(config.Database.Host, ShouldEqual, "localhost")
			So(config.Database.Port, ShouldEqual, 3306)
			So(len(config.Servers), ShouldEqual, 2)
			So(config.Servers[0], ShouldEqual, "server1")
			So(config.Servers[1], ShouldEqual, "server2")
		})
	})
}

// TestMapStorage_ConvertTo_ComplexNestedStructure 测试复杂嵌套结构转换
// 包含结构体中有map和slice，而slice/map中也包含结构体的情况
func TestMapStorage_ConvertTo_ComplexNestedStructure(t *testing.T) {
	Convey("MapStorage 复杂嵌套结构转换测试", t, func() {
		// 定义嵌套的结构体类型
		type Endpoint struct {
			URL     string `json:"url"`
			Timeout string `json:"timeout"`
			Retries int    `json:"retries"`
		}

		type ServiceConfig struct {
			Name      string              `json:"name"`
			Enabled   bool                `json:"enabled"`
			Endpoints []Endpoint          `json:"endpoints"`
			Metadata  map[string]string   `json:"metadata"`
			Advanced  map[string]Endpoint `json:"advanced"`
		}

		type DatabaseConnection struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Database string `json:"database"`
			Pool     struct {
				MinSize int `json:"min_size"`
				MaxSize int `json:"max_size"`
			} `json:"pool"`
		}

		type ComplexConfig struct {
			Application struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"application"`
			Services    []ServiceConfig               `json:"services"`
			Databases   map[string]DatabaseConnection `json:"databases"`
			Environment map[string]interface{}        `json:"environment"`
			Features    map[string][]string           `json:"features"`
		}

		// 构造复杂的测试数据
		data := map[string]interface{}{
			"application": map[string]interface{}{
				"name":    "complex-app",
				"version": "1.0.0",
			},
			"services": []interface{}{
				map[string]interface{}{
					"name":    "auth-service",
					"enabled": true,
					"endpoints": []interface{}{
						map[string]interface{}{
							"url":     "https://auth.example.com/login",
							"timeout": "30s",
							"retries": 3,
						},
						map[string]interface{}{
							"url":     "https://auth.example.com/logout",
							"timeout": "15s",
							"retries": 1,
						},
					},
					"metadata": map[string]interface{}{
						"team":        "security",
						"environment": "production",
					},
					"advanced": map[string]interface{}{
						"health_check": map[string]interface{}{
							"url":     "https://auth.example.com/health",
							"timeout": "5s",
							"retries": 2,
						},
						"metrics": map[string]interface{}{
							"url":     "https://auth.example.com/metrics",
							"timeout": "10s",
							"retries": 1,
						},
					},
				},
				map[string]interface{}{
					"name":    "notification-service",
					"enabled": false,
					"endpoints": []interface{}{
						map[string]interface{}{
							"url":     "https://notify.example.com/send",
							"timeout": "60s",
							"retries": 5,
						},
					},
					"metadata": map[string]interface{}{
						"team": "messaging",
					},
					"advanced": map[string]interface{}{},
				},
			},
			"databases": map[string]interface{}{
				"primary": map[string]interface{}{
					"host":     "primary-db.example.com",
					"port":     5432,
					"database": "app_production",
					"pool": map[string]interface{}{
						"min_size": 5,
						"max_size": 20,
					},
				},
				"cache": map[string]interface{}{
					"host":     "cache.example.com",
					"port":     6379,
					"database": "0",
					"pool": map[string]interface{}{
						"min_size": 2,
						"max_size": 10,
					},
				},
			},
			"environment": map[string]interface{}{
				"stage":      "production",
				"debug":      false,
				"log_level":  "info",
				"max_memory": "2GB",
			},
			"features": map[string]interface{}{
				"experimental": []interface{}{"feature-a", "feature-b"},
				"stable":       []interface{}{"feature-x", "feature-y", "feature-z"},
				"deprecated":   []interface{}{},
			},
		}

		Convey("完整复杂结构转换", func() {
			storage := NewMapStorage(data)
			var config ComplexConfig
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)

			Convey("验证应用程序信息", func() {
				So(config.Application.Name, ShouldEqual, "complex-app")
				So(config.Application.Version, ShouldEqual, "1.0.0")
			})

			Convey("验证服务配置", func() {
				So(len(config.Services), ShouldEqual, 2)

				Convey("第一个服务（auth-service）", func() {
					authService := config.Services[0]
					So(authService.Name, ShouldEqual, "auth-service")
					So(authService.Enabled, ShouldBeTrue)

					Convey("验证endpoints", func() {
						So(len(authService.Endpoints), ShouldEqual, 2)

						loginEndpoint := authService.Endpoints[0]
						So(loginEndpoint.URL, ShouldEqual, "https://auth.example.com/login")
						So(loginEndpoint.Timeout, ShouldEqual, "30s")
						So(loginEndpoint.Retries, ShouldEqual, 3)
					})

					Convey("验证metadata", func() {
						So(authService.Metadata["team"], ShouldEqual, "security")
						So(authService.Metadata["environment"], ShouldEqual, "production")
					})

					Convey("验证advanced配置", func() {
						So(len(authService.Advanced), ShouldEqual, 2)

						healthCheck, exists := authService.Advanced["health_check"]
						So(exists, ShouldBeTrue)
						So(healthCheck.URL, ShouldEqual, "https://auth.example.com/health")
						So(healthCheck.Timeout, ShouldEqual, "5s")
						So(healthCheck.Retries, ShouldEqual, 2)
					})
				})

				Convey("第二个服务（notification-service）", func() {
					notifyService := config.Services[1]
					So(notifyService.Name, ShouldEqual, "notification-service")
					So(notifyService.Enabled, ShouldBeFalse)
					So(len(notifyService.Advanced), ShouldEqual, 0)
				})
			})

			Convey("验证数据库配置", func() {
				So(len(config.Databases), ShouldEqual, 2)

				Convey("主数据库配置", func() {
					primaryDB, exists := config.Databases["primary"]
					So(exists, ShouldBeTrue)
					So(primaryDB.Host, ShouldEqual, "primary-db.example.com")
					So(primaryDB.Port, ShouldEqual, 5432)
					So(primaryDB.Database, ShouldEqual, "app_production")
					So(primaryDB.Pool.MinSize, ShouldEqual, 5)
					So(primaryDB.Pool.MaxSize, ShouldEqual, 20)
				})

				Convey("缓存数据库配置", func() {
					cacheDB, exists := config.Databases["cache"]
					So(exists, ShouldBeTrue)
					So(cacheDB.Port, ShouldEqual, 6379)
				})
			})

			Convey("验证环境变量", func() {
				So(len(config.Environment), ShouldEqual, 4)
				So(config.Environment["stage"], ShouldEqual, "production")
				So(config.Environment["debug"], ShouldEqual, false)
			})

			Convey("验证特性配置", func() {
				So(len(config.Features), ShouldEqual, 3)

				experimental, exists := config.Features["experimental"]
				So(exists, ShouldBeTrue)
				So(len(experimental), ShouldEqual, 2)
				So(experimental[0], ShouldEqual, "feature-a")
				So(experimental[1], ShouldEqual, "feature-b")

				stable, exists := config.Features["stable"]
				So(exists, ShouldBeTrue)
				So(len(stable), ShouldEqual, 3)

				deprecated, exists := config.Features["deprecated"]
				So(exists, ShouldBeTrue)
				So(len(deprecated), ShouldEqual, 0)
			})
		})
	})
}

// TestMapStorage_ConvertTo_WithDefaults 测试使用def tag的默认值功能
func TestMapStorage_ConvertTo_WithDefaults(t *testing.T) {
	Convey("MapStorage 使用def tag的默认值功能测试", t, func() {
		// 定义带有默认值的结构体
		type ServerConfig struct {
			Host     string        `json:"host" def:"localhost"`
			Port     int           `json:"port" def:"8080"`
			Enabled  bool          `json:"enabled" def:"true"`
			Timeout  time.Duration `json:"timeout" def:"30s"`
			MaxConns int           `json:"max_conns" def:"100"`
			Tags     []string      `json:"tags" def:"web,api,service"`
		}

		type DatabaseConfig struct {
			Host     string `json:"host" def:"db.example.com"`
			Port     int    `json:"port" def:"5432"`
			Database string `json:"database" def:"myapp"`
			Pool     struct {
				MinSize int `json:"min_size" def:"5"`
				MaxSize int `json:"max_size" def:"20"`
			} `json:"pool"`
		}

		type AppConfig struct {
			Name     string         `json:"name" def:"MyApp"`
			Version  string         `json:"version" def:"1.0.0"`
			Debug    bool           `json:"debug" def:"false"`
			Server   ServerConfig   `json:"server"`
			Database DatabaseConfig `json:"database"`
		}

		Convey("完全空配置使用默认值", func() {
			// 空的配置数据
			data := map[string]interface{}{}
			storage := NewMapStorage(data)

			var config AppConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)

			// 验证顶级字段的默认值
			So(config.Name, ShouldEqual, "MyApp")
			So(config.Version, ShouldEqual, "1.0.0")
			So(config.Debug, ShouldBeFalse)

			// 验证嵌套结构体的默认值
			So(config.Server.Host, ShouldEqual, "localhost")
			So(config.Server.Port, ShouldEqual, 8080)
			So(config.Server.Enabled, ShouldBeTrue)
			So(config.Server.Timeout, ShouldEqual, 30*time.Second)
			So(config.Server.MaxConns, ShouldEqual, 100)

			// 验证切片默认值
			expectedTags := []string{"web", "api", "service"}
			So(len(config.Server.Tags), ShouldEqual, 3)
			for i, tag := range expectedTags {
				So(config.Server.Tags[i], ShouldEqual, tag)
			}

			// 验证数据库配置的默认值
			So(config.Database.Host, ShouldEqual, "db.example.com")
			So(config.Database.Port, ShouldEqual, 5432)
			So(config.Database.Database, ShouldEqual, "myapp")

			// 验证嵌套结构体字段的默认值
			So(config.Database.Pool.MinSize, ShouldEqual, 5)
			So(config.Database.Pool.MaxSize, ShouldEqual, 20)
		})

		Convey("部分配置覆盖默认值", func() {
			// 部分配置数据
			data := map[string]interface{}{
				"name": "CustomApp",
				"server": map[string]interface{}{
					"host": "custom.example.com",
					"port": 9090,
					// enabled 和 timeout 使用默认值
				},
				"database": map[string]interface{}{
					"host": "custom-db.example.com",
					// port 和 database 使用默认值
					"pool": map[string]interface{}{
						"max_size": 50,
						// min_size 使用默认值
					},
				},
			}
			storage := NewMapStorage(data)

			var config AppConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)

			// 验证被覆盖的值
			So(config.Name, ShouldEqual, "CustomApp")
			So(config.Server.Host, ShouldEqual, "custom.example.com")
			So(config.Server.Port, ShouldEqual, 9090)
			So(config.Database.Host, ShouldEqual, "custom-db.example.com")
			So(config.Database.Pool.MaxSize, ShouldEqual, 50)

			// 验证使用默认值的字段
			So(config.Version, ShouldEqual, "1.0.0")
			So(config.Server.Enabled, ShouldBeTrue)
			So(config.Server.Timeout, ShouldEqual, 30*time.Second)
			So(config.Database.Port, ShouldEqual, 5432)
			So(config.Database.Database, ShouldEqual, "myapp")
			So(config.Database.Pool.MinSize, ShouldEqual, 5)
		})

		Convey("禁用默认值功能", func() {
			// 空配置数据
			data := map[string]interface{}{}
			storage := NewMapStorage(data).WithDefaults(false)

			var config AppConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)

			// 验证所有字段都是零值（没有应用默认值）
			So(config.Name, ShouldEqual, "")
			So(config.Version, ShouldEqual, "")
			So(config.Debug, ShouldBeFalse) // bool 的零值是 false
			So(config.Server.Host, ShouldEqual, "")
			So(config.Server.Port, ShouldEqual, 0)
			So(config.Server.Enabled, ShouldBeFalse)
			So(config.Server.Timeout, ShouldEqual, 0)
			So(len(config.Server.Tags), ShouldEqual, 0)
		})
	})
}

// TestMapStorage_ConvertTo_DefaultsWithPointers 测试指针字段的默认值处理
func TestMapStorage_ConvertTo_DefaultsWithPointers(t *testing.T) {
	Convey("MapStorage 指针字段的默认值处理测试", t, func() {
		type DatabaseConfig struct {
			Host     string `json:"host" def:"localhost"`
			Port     int    `json:"port" def:"5432"`
			Username string `json:"username" def:"admin"`
		}

		type AppConfig struct {
			Name     string          `json:"name" def:"TestApp"`
			Database *DatabaseConfig `json:"database"`
			Optional *DatabaseConfig `json:"optional"`
		}

		Convey("指针字段在配置中存在时应用默认值", func() {
			data := map[string]interface{}{
				"database": map[string]interface{}{
					"host": "custom.db.com",
					// port 和 username 使用默认值
				},
			}
			storage := NewMapStorage(data)

			var config AppConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)

			// 验证顶级默认值
			So(config.Name, ShouldEqual, "TestApp")

			// 验证指针字段不为空且应用了默认值
			So(config.Database, ShouldNotBeNil)
			So(config.Database.Host, ShouldEqual, "custom.db.com")
			So(config.Database.Port, ShouldEqual, 5432)
			So(config.Database.Username, ShouldEqual, "admin")

			// 验证可选字段保持nil
			So(config.Optional, ShouldBeNil)
		})

		Convey("指针字段在配置中不存在时保持nil", func() {
			data := map[string]interface{}{
				"name": "OnlyName",
			}
			storage := NewMapStorage(data)

			var config AppConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)

			So(config.Name, ShouldEqual, "OnlyName")

			// 指针字段应该保持nil，因为配置中没有对应的数据
			So(config.Database, ShouldBeNil)
			So(config.Optional, ShouldBeNil)
		})
	})
}

// TestMapStorage_Equals_Basic 测试基本比较功能
func TestMapStorage_Equals_Basic(t *testing.T) {
	Convey("MapStorage 基本比较功能测试", t, func() {
		data1 := map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		}

		data2 := map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		}

		data3 := map[string]interface{}{
			"host": "localhost",
			"port": 3307,
		}

		storage1 := NewMapStorage(data1)
		storage2 := NewMapStorage(data2)
		storage3 := NewMapStorage(data3)

		Convey("相同数据的storage应该相等", func() {
			So(storage1.Equals(storage2), ShouldBeTrue)
		})

		Convey("不同数据的storage应该不相等", func() {
			So(storage1.Equals(storage3), ShouldBeFalse)
		})

		Convey("storage与自身应该相等", func() {
			So(storage1.Equals(storage1), ShouldBeTrue)
		})
	})
}

// TestMapStorage_Equals_ComplexData 测试复杂数据比较
func TestMapStorage_Equals_ComplexData(t *testing.T) {
	Convey("MapStorage 复杂数据比较测试", t, func() {
		complexData1 := map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 3306,
			},
			"servers": []interface{}{"server1", "server2"},
		}

		complexData2 := map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 3306,
			},
			"servers": []interface{}{"server1", "server2"},
		}

		complexData3 := map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 3306,
			},
			"servers": []interface{}{"server1", "server3"}, // 不同的server
		}

		storage1 := NewMapStorage(complexData1)
		storage2 := NewMapStorage(complexData2)
		storage3 := NewMapStorage(complexData3)

		Convey("相同复杂数据的storage应该相等", func() {
			So(storage1.Equals(storage2), ShouldBeTrue)
		})

		Convey("不同复杂数据的storage应该不相等", func() {
			So(storage1.Equals(storage3), ShouldBeFalse)
		})
	})
}

// TestMapStorage_Equals_NilHandling 测试nil处理
func TestMapStorage_Equals_NilHandling(t *testing.T) {
	Convey("MapStorage nil处理测试", t, func() {
		data := map[string]interface{}{
			"key": "value",
		}

		normalStorage := NewMapStorage(data)
		var nilStorage1 *MapStorage = nil
		var nilStorage2 *MapStorage = nil

		// 获取通过Sub方法返回的nil storage
		nilFromSub := normalStorage.Sub("nonexistent")

		Convey("nil storage与nil storage应该相等", func() {
			So(nilStorage1.Equals(nilStorage2), ShouldBeTrue)
		})

		Convey("nil storage与从Sub返回的nil storage应该相等", func() {
			So(nilStorage1.Equals(nilFromSub), ShouldBeTrue)
		})

		Convey("从Sub返回的nil storage之间应该相等", func() {
			nilFromSub2 := normalStorage.Sub("another_nonexistent")
			So(nilFromSub.Equals(nilFromSub2), ShouldBeTrue)
		})

		Convey("nil storage与正常storage应该不相等", func() {
			So(nilStorage1.Equals(normalStorage), ShouldBeFalse)
		})

		Convey("正常storage与nil storage应该不相等", func() {
			So(normalStorage.Equals(nilStorage1), ShouldBeFalse)
		})

		Convey("nil storage与nil接口应该不相等", func() {
			So(nilStorage1.Equals(nil), ShouldBeFalse)
		})
	})
}

// TestMapStorage_Equals_EmptyData 测试空数据比较
func TestMapStorage_Equals_EmptyData(t *testing.T) {
	Convey("MapStorage 空数据比较测试", t, func() {
		empty1 := NewMapStorage(nil)
		empty2 := NewMapStorage(nil)
		emptyMap1 := NewMapStorage(map[string]interface{}{})
		emptyMap2 := NewMapStorage(map[string]interface{}{})

		Convey("nil数据应该相等", func() {
			So(empty1.Equals(empty2), ShouldBeTrue)
		})

		Convey("空map应该相等", func() {
			So(emptyMap1.Equals(emptyMap2), ShouldBeTrue)
		})

		Convey("nil和空map在reflect.DeepEqual中不相等，这是预期行为", func() {
			So(empty1.Equals(emptyMap1), ShouldBeFalse)
		})
	})
}

// TestMapStorage_Equals_SubStorage 测试子Storage的比较
func TestMapStorage_Equals_SubStorage(t *testing.T) {
	Convey("MapStorage 子Storage比较测试", t, func() {
		data1 := map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 3306,
			},
		}

		data2 := map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 3306,
			},
		}

		data3 := map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 3307,
			},
		}

		storage1 := NewMapStorage(data1)
		storage2 := NewMapStorage(data2)
		storage3 := NewMapStorage(data3)

		sub1 := storage1.Sub("database")
		sub2 := storage2.Sub("database")
		sub3 := storage3.Sub("database")

		Convey("相同数据的子Storage应该相等", func() {
			So(sub1.Equals(sub2), ShouldBeTrue)
		})

		Convey("不同数据的子Storage应该不相等", func() {
			So(sub1.Equals(sub3), ShouldBeFalse)
		})
	})
}

// MockStorage 用于测试的模拟Storage实现
type MockStorage struct{}

func (ms *MockStorage) Sub(key string) Storage             { return nil }
func (ms *MockStorage) ConvertTo(object interface{}) error { return nil }
func (ms *MockStorage) Equals(other Storage) bool          { return false }

// TestMapStorage_Equals_DifferentTypes 测试不同类型的比较
func TestMapStorage_Equals_DifferentTypes(t *testing.T) {
	Convey("MapStorage 不同类型比较测试", t, func() {
		storage := NewMapStorage(testData)
		mockStorage := &MockStorage{}

		Convey("MapStorage与其他类型的Storage比较应该返回false", func() {
			So(storage.Equals(mockStorage), ShouldBeFalse)
		})
	})
}

// TestMapStorage_ConvertTo_NilStorage 测试nil storage的ConvertTo行为
func TestMapStorage_ConvertTo_NilStorage(t *testing.T) {
	Convey("MapStorage nil storage的ConvertTo行为测试", t, func() {
		// 获取一个nil storage
		normalStorage := NewMapStorage(testData)
		nilStorage := normalStorage.Sub("nonexistent")

		// 测试对空指针的处理
		type TestConfig struct {
			Name string `json:"name"`
			Port int    `json:"port"`
		}

		Convey("nil指针应该保持nil", func() {
			var nilConfig *TestConfig = nil
			err := nilStorage.ConvertTo(&nilConfig)
			So(err, ShouldBeNil)
			So(nilConfig, ShouldBeNil)
		})

		Convey("非空指针的值应该保持不变", func() {
			existingConfig := &TestConfig{Name: "existing", Port: 5432}
			err := nilStorage.ConvertTo(&existingConfig)
			So(err, ShouldBeNil)
			So(existingConfig.Name, ShouldEqual, "existing")
			So(existingConfig.Port, ShouldEqual, 5432)
		})
	})
}

// TestMapStorage_ConvertTo_PointerFields 测试指针字段的智能处理
func TestMapStorage_ConvertTo_PointerFields(t *testing.T) {
	Convey("MapStorage 指针字段智能处理测试", t, func() {
		type InnerConfig struct {
			Value string `json:"value"`
			Count int    `json:"count"`
		}

		type OuterConfig struct {
			Name     string       `json:"name"`
			Inner    *InnerConfig `json:"inner"`
			Optional *InnerConfig `json:"optional"`
		}

		Convey("配置中没有指针字段", func() {
			data := map[string]interface{}{
				"name": "test",
			}
			storage := NewMapStorage(data)

			Convey("目标结构体的指针字段为nil", func() {
				var config1 OuterConfig
				config1.Inner = nil
				config1.Optional = nil

				err := storage.ConvertTo(&config1)
				So(err, ShouldBeNil)
				So(config1.Name, ShouldEqual, "test")
				So(config1.Inner, ShouldBeNil)
				So(config1.Optional, ShouldBeNil)
			})

			Convey("目标结构体的指针字段已有值", func() {
				existingInner := &InnerConfig{Value: "existing", Count: 999}
				var config2 OuterConfig
				config2.Inner = existingInner
				config2.Optional = nil

				err := storage.ConvertTo(&config2)
				So(err, ShouldBeNil)
				So(config2.Inner, ShouldEqual, existingInner)
				So(config2.Inner.Value, ShouldEqual, "existing")
				So(config2.Inner.Count, ShouldEqual, 999)
			})
		})

		Convey("配置中有指针字段", func() {
			data := map[string]interface{}{
				"name": "test",
				"inner": map[string]interface{}{
					"value": "configured",
					"count": 42,
				},
			}
			storage := NewMapStorage(data)

			Convey("Inner应该被创建并赋值", func() {
				var config OuterConfig
				config.Inner = nil
				config.Optional = nil

				err := storage.ConvertTo(&config)
				So(err, ShouldBeNil)
				So(config.Inner, ShouldNotBeNil)
				So(config.Inner.Value, ShouldEqual, "configured")
				So(config.Inner.Count, ShouldEqual, 42)
				So(config.Optional, ShouldBeNil)
			})
		})
	})
}

// TestMapStorage_ErrorHandling 测试错误处理
func TestMapStorage_ErrorHandling(t *testing.T) {
	Convey("MapStorage 错误处理测试", t, func() {
		Convey("类型转换错误", func() {
			// 尝试将非数组数据转换为切片
			stringStorage := NewMapStorage("not a slice")
			var slice []string
			err := stringStorage.ConvertTo(&slice)
			So(err, ShouldNotBeNil)
		})

		Convey("时间解析错误", func() {
			// 无效的时间格式
			invalidTimeStorage := NewMapStorage("invalid-time-format")
			var timeValue time.Time
			err := invalidTimeStorage.ConvertTo(&timeValue)
			So(err, ShouldNotBeNil)
		})

		Convey("Duration解析错误", func() {
			// 无效的duration格式
			invalidDurationStorage := NewMapStorage("invalid-duration")
			var duration time.Duration
			err := invalidDurationStorage.ConvertTo(&duration)
			So(err, ShouldNotBeNil)
		})
	})
}

// TestMapStorage_EdgeCases 测试边界情况
func TestMapStorage_EdgeCases(t *testing.T) {
	Convey("MapStorage 边界情况测试", t, func() {
		Convey("空字符串路径", func() {
			storage := NewMapStorage(testData)
			result := storage.Sub("")

			// 空路径应该返回自身
			So(result, ShouldNotBeNil)

			var data interface{}
			err := result.ConvertTo(&data)
			So(err, ShouldBeNil)
			So(deepEqual(data, testData), ShouldBeTrue)
		})

		Convey("路径中包含特殊字符", func() {
			specialData := map[string]interface{}{
				"key.with.dots":     "value1",
				"key[with]brackets": "value2",
				"normal_key":        "value3",
			}

			storage := NewMapStorage(specialData)

			// 正常的key应该能访问
			result := storage.Sub("normal_key")
			So(result, ShouldNotBeNil)

			var value string
			err := result.ConvertTo(&value)
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "value3")
		})

		Convey("深层嵌套null值", func() {
			nullData := map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": nil,
				},
			}

			storage := NewMapStorage(nullData)
			result := storage.Sub("level1.level2")

			// 应该返回nil storage（类型是*MapStorage但值是nil）
			ms, ok := result.(*MapStorage)
			So(ok, ShouldBeTrue)
			So(ms, ShouldBeNil)
		})
	})
}

// TestMapStorage_ValidateStruct 测试结构体校验功能
func TestMapStorage_ValidateStruct(t *testing.T) {
	Convey("MapStorage 结构体校验测试", t, func() {
		
		// 定义测试用的结构体
		type User struct {
			Name  string `json:"name" validate:"required,min=2,max=50"`
			Email string `json:"email" validate:"required,email"`
			Age   int    `json:"age" validate:"min=0,max=150"`
		}

		type Address struct {
			Street string `json:"street" validate:"required"`
			City   string `json:"city" validate:"required"`
		}

		type UserWithAddress struct {
			Name    string  `json:"name" validate:"required"`
			Email   string  `json:"email" validate:"required,email"`
			Address Address `json:"address" validate:"required"`
		}

		type UserWithPointer struct {
			Name     string   `json:"name" validate:"required"`
			Email    string   `json:"email" validate:"required,email"`
			Address  *Address `json:"address,omitempty" validate:"omitempty"`
		}

		Convey("有效的结构体校验", func() {
			data := map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"age":   30,
			}
			storage := NewMapStorage(data)
			var user User
			err := storage.ConvertTo(&user)
			
			So(err, ShouldBeNil)
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Email, ShouldEqual, "john@example.com")
			So(user.Age, ShouldEqual, 30)
		})

		Convey("校验失败 - 必填字段为空", func() {
			data := map[string]interface{}{
				"name":  "",
				"email": "john@example.com",
				"age":   30,
			}
			storage := NewMapStorage(data)
			var user User
			err := storage.ConvertTo(&user)
			
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "validation failed")
		})

		Convey("校验失败 - 邮箱格式错误", func() {
			data := map[string]interface{}{
				"name":  "John Doe",
				"email": "invalid-email",
				"age":   30,
			}
			storage := NewMapStorage(data)
			var user User
			err := storage.ConvertTo(&user)
			
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "validation failed")
		})

		Convey("校验失败 - 年龄超出范围", func() {
			data := map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"age":   200,
			}
			storage := NewMapStorage(data)
			var user User
			err := storage.ConvertTo(&user)
			
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "validation failed")
		})

		Convey("嵌套结构体校验 - 有效", func() {
			data := map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"address": map[string]interface{}{
					"street": "123 Main St",
					"city":   "New York",
				},
			}
			storage := NewMapStorage(data)
			var user UserWithAddress
			err := storage.ConvertTo(&user)
			
			So(err, ShouldBeNil)
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Address.Street, ShouldEqual, "123 Main St")
			So(user.Address.City, ShouldEqual, "New York")
		})

		Convey("嵌套结构体校验 - 失败", func() {
			data := map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"address": map[string]interface{}{
					"street": "",  // 必填字段为空
					"city":   "New York",
				},
			}
			storage := NewMapStorage(data)
			var user UserWithAddress
			err := storage.ConvertTo(&user)
			
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "validation failed")
		})

		Convey("指针结构体校验 - nil 指针", func() {
			data := map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			}
			storage := NewMapStorage(data)
			var user UserWithPointer
			err := storage.ConvertTo(&user)
			
			So(err, ShouldBeNil)
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Address, ShouldBeNil)
		})

		Convey("指针结构体校验 - 有效指针", func() {
			data := map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"address": map[string]interface{}{
					"street": "123 Main St",
					"city":   "New York",
				},
			}
			storage := NewMapStorage(data)
			var user UserWithPointer
			err := storage.ConvertTo(&user)
			
			So(err, ShouldBeNil)
			So(user.Name, ShouldEqual, "John Doe")
			So(user.Address, ShouldNotBeNil)
			So(user.Address.Street, ShouldEqual, "123 Main St")
		})

		Convey("time.Time 类型跳过校验", func() {
			storage := NewMapStorage("2023-12-25T15:30:45Z")
			var timeValue time.Time
			err := storage.ConvertTo(&timeValue)
			
			So(err, ShouldBeNil)
			expected := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
			So(timeValue.Equal(expected), ShouldBeTrue)
		})

		Convey("基本类型跳过校验", func() {
			storage := NewMapStorage(42)
			var intValue int
			err := storage.ConvertTo(&intValue)
			
			So(err, ShouldBeNil)
			So(intValue, ShouldEqual, 42)
		})

		Convey("nil storage 跳过校验", func() {
			// 获取一个 nil storage（通过不存在的 key）
			normalStorage := NewMapStorage(testData)
			nilStorage := normalStorage.Sub("nonexistent")
			var user *User
			err := nilStorage.ConvertTo(&user)
			
			So(err, ShouldBeNil)
			So(user, ShouldBeNil)
		})
		
		Convey("nil 数据跳过校验", func() {
			storage := NewMapStorage(nil)
			var user *User
			err := storage.ConvertTo(&user)
			
			So(err, ShouldBeNil)
			// 当数据为 nil 时，会创建零值结构体但跳过校验
			So(user, ShouldNotBeNil)
			So(user.Name, ShouldEqual, "")
			So(user.Email, ShouldEqual, "")
		})

		Convey("map 类型跳过校验", func() {
			data := map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			}
			storage := NewMapStorage(data)
			var result map[string]string
			err := storage.ConvertTo(&result)
			
			So(err, ShouldBeNil)
			So(result["key1"], ShouldEqual, "value1")
			So(result["key2"], ShouldEqual, "value2")
		})

		Convey("slice 类型跳过校验", func() {
			data := []interface{}{"item1", "item2", "item3"}
			storage := NewMapStorage(data)
			var result []string
			err := storage.ConvertTo(&result)
			
			So(err, ShouldBeNil)
			So(len(result), ShouldEqual, 3)
			So(result[0], ShouldEqual, "item1")
		})
	})
}
