package storage

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

// 测试用的扁平化数据集
var testFlatData = map[string]interface{}{
	"database.host":               "localhost",
	"database.port":               3306,
	"database.connections.0.name": "primary",
	"database.connections.0.user": "admin",
	"database.connections.1.name": "secondary",
	"database.connections.1.user": "readonly",
	"servers.0":                   "server1",
	"servers.1":                   "server2",
	"config.timeout":              "30s",
	"config.created_at":           "2023-12-25T15:30:45Z",
	"config.enabled":              true,
}

// TestFlatStorage_Creation 测试 FlatStorage 的各种创建方式
func TestFlatStorage_Creation(t *testing.T) {
	Convey("FlatStorage 创建测试", t, func() {
		Convey("创建基本的 FlatStorage", func() {
			storage := NewFlatStorage(testFlatData)
			So(storage, ShouldNotBeNil)
			So(storage.Data(), ShouldResemble, testFlatData)
			So(storage.separator, ShouldEqual, ".")
			So(storage.enableDefaults, ShouldBeFalse)
			So(storage.uppercase, ShouldBeFalse)
			So(storage.lowercase, ShouldBeFalse)
		})

		Convey("创建空数据的 FlatStorage", func() {
			storage := NewFlatStorage(nil)
			So(storage, ShouldNotBeNil)
			So(storage.Data(), ShouldBeNil)
		})

		Convey("创建空 map 的 FlatStorage", func() {
			emptyMap := map[string]interface{}{}
			storage := NewFlatStorage(emptyMap)
			So(storage, ShouldNotBeNil)
			So(storage.Data(), ShouldResemble, emptyMap)
		})
	})
}

// TestFlatStorage_WithOptions 测试 FlatStorage 的配置选项
func TestFlatStorage_WithOptions(t *testing.T) {
	Convey("FlatStorage 配置选项测试", t, func() {
		storage := NewFlatStorage(testFlatData)

		Convey("开启默认值", func() {
			result := storage.WithDefaults(true)
			So(result.enableDefaults, ShouldBeTrue)
			So(result, ShouldEqual, storage) // 返回自身
		})

		Convey("关闭默认值", func() {
			result := storage.WithDefaults(false)
			So(result.enableDefaults, ShouldBeFalse)
		})

		Convey("设置分隔符", func() {
			result := storage.WithSeparator("-")
			So(result.separator, ShouldEqual, "-")
		})

		Convey("开启大写转换", func() {
			result := storage.WithUppercase(true)
			So(result.uppercase, ShouldBeTrue)
			So(result.lowercase, ShouldBeFalse) // 确保不冲突
		})

		Convey("开启小写转换", func() {
			result := storage.WithLowercase(true)
			So(result.lowercase, ShouldBeTrue)
			So(result.uppercase, ShouldBeFalse) // 确保不冲突
		})

		Convey("链式调用配置", func() {
			result := storage.WithDefaults(true).WithSeparator("_").WithUppercase(true)
			So(result.enableDefaults, ShouldBeTrue)
			So(result.separator, ShouldEqual, "_")
			So(result.uppercase, ShouldBeTrue)
		})
	})
}

// TestFlatStorage_Data 测试数据获取功能
func TestFlatStorage_Data(t *testing.T) {
	Convey("FlatStorage 数据获取测试", t, func() {
		Convey("获取 map 数据", func() {
			storage := NewFlatStorage(testFlatData)
			So(storage.Data(), ShouldResemble, testFlatData)
		})

		Convey("获取 nil 数据", func() {
			storage := NewFlatStorage(nil)
			So(storage.Data(), ShouldBeNil)
		})

		Convey("获取空 map 数据", func() {
			emptyMap := map[string]interface{}{}
			storage := NewFlatStorage(emptyMap)
			So(storage.Data(), ShouldResemble, emptyMap)
		})
	})
}

// TestFlatStorage_ConvertTo_BasicTypes 测试基本类型转换
func TestFlatStorage_ConvertTo_BasicTypes(t *testing.T) {
	Convey("FlatStorage 基本类型转换测试", t, func() {
		Convey("字符串转换", func() {
			data := map[string]interface{}{
				"name": "hello world",
			}
			storage := NewFlatStorage(data)

			var result string
			err := storage.Sub("name").ConvertTo(&result)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "hello world")
		})

		Convey("整数转换", func() {
			data := map[string]interface{}{
				"port": 42,
			}
			storage := NewFlatStorage(data)

			var result int
			err := storage.Sub("port").ConvertTo(&result)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, 42)
		})

		Convey("浮点数转换", func() {
			data := map[string]interface{}{
				"ratio": 3.14,
			}
			storage := NewFlatStorage(data)

			var result float64
			err := storage.Sub("ratio").ConvertTo(&result)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, 3.14)
		})

		Convey("布尔值转换", func() {
			data := map[string]interface{}{
				"enabled": true,
			}
			storage := NewFlatStorage(data)

			var result bool
			err := storage.Sub("enabled").ConvertTo(&result)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, true)
		})

		Convey("直接在根级别转换", func() {
			storage := NewFlatStorage(testFlatData)

			var host string
			err := storage.Sub("database.host").ConvertTo(&host)
			So(err, ShouldBeNil)
			So(host, ShouldEqual, "localhost")

			var port int
			err = storage.Sub("database.port").ConvertTo(&port)
			So(err, ShouldBeNil)
			So(port, ShouldEqual, 3306)
		})
	})
}

// TestFlatStorage_ConvertTo_Struct 测试结构体转换
func TestFlatStorage_ConvertTo_Struct(t *testing.T) {
	Convey("FlatStorage 结构体转换测试", t, func() {
		type ServerConfig struct {
			Host    string `json:"host"`
			Port    int    `json:"port"`
			Enabled bool   `json:"enabled"`
		}

		Convey("简单结构体转换", func() {
			data := map[string]interface{}{
				"host":    "test-server",
				"port":    8080,
				"enabled": true,
			}

			storage := NewFlatStorage(data)
			var config ServerConfig
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)
			So(config.Host, ShouldEqual, "test-server")
			So(config.Port, ShouldEqual, 8080)
			So(config.Enabled, ShouldBeTrue)
		})

		Convey("嵌套结构体转换", func() {
			type DatabaseConfig struct {
				Host string `json:"host"`
				Port int    `json:"port"`
			}

			type AppConfig struct {
				Name     string         `json:"name"`
				Database DatabaseConfig `json:"database"`
			}

			flatData := map[string]interface{}{
				"name":          "test-app",
				"database.host": "localhost",
				"database.port": 3306,
			}

			storage := NewFlatStorage(flatData)
			var config AppConfig
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)
			So(config.Name, ShouldEqual, "test-app")
			So(config.Database.Host, ShouldEqual, "localhost")
			So(config.Database.Port, ShouldEqual, 3306)
		})

		Convey("带标签的结构体转换", func() {
			type TestConfig struct {
				Field1 string `cfg:"custom_name"`
				Field2 string `json:"json_field"`
				Field3 string `yaml:"yaml_field"`
				Field4 string // 无标签，使用字段名
			}

			data := map[string]interface{}{
				"custom_name": "value1",
				"json_field":  "value2",
				"yaml_field":  "value3",
				"Field4":      "value4",
			}

			storage := NewFlatStorage(data)
			var config TestConfig
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)
			So(config.Field1, ShouldEqual, "value1")
			So(config.Field2, ShouldEqual, "value2")
			So(config.Field3, ShouldEqual, "value3")
			So(config.Field4, ShouldEqual, "value4")
		})
	})
}

// TestFlatStorage_ConvertTo_Slice 测试切片转换
func TestFlatStorage_ConvertTo_Slice(t *testing.T) {
	Convey("FlatStorage 切片转换测试", t, func() {
		Convey("字符串切片", func() {
			data := map[string]interface{}{
				"servers.0": "server1",
				"servers.1": "server2",
				"servers.2": "server3",
			}

			storage := NewFlatStorage(data)
			var servers []string
			err := storage.Sub("servers").ConvertTo(&servers)

			So(err, ShouldBeNil)
			So(len(servers), ShouldEqual, 3)
			So(servers[0], ShouldEqual, "server1")
			So(servers[1], ShouldEqual, "server2")
			So(servers[2], ShouldEqual, "server3")
		})

		Convey("整数切片", func() {
			data := map[string]interface{}{
				"ports.0": 8080,
				"ports.1": 8081,
				"ports.2": 8082,
			}

			storage := NewFlatStorage(data)
			var ports []int
			err := storage.Sub("ports").ConvertTo(&ports)

			So(err, ShouldBeNil)
			So(len(ports), ShouldEqual, 3)
			So(ports[0], ShouldEqual, 8080)
			So(ports[1], ShouldEqual, 8081)
			So(ports[2], ShouldEqual, 8082)
		})

		Convey("结构体切片", func() {
			type Server struct {
				Name string `json:"name"`
				Port int    `json:"port"`
			}

			data := map[string]interface{}{
				"servers.0.name": "web1",
				"servers.0.port": 8080,
				"servers.1.name": "web2",
				"servers.1.port": 8081,
			}

			storage := NewFlatStorage(data)
			var servers []Server
			err := storage.Sub("servers").ConvertTo(&servers)

			So(err, ShouldBeNil)
			So(len(servers), ShouldEqual, 2)
			So(servers[0].Name, ShouldEqual, "web1")
			So(servers[0].Port, ShouldEqual, 8080)
			So(servers[1].Name, ShouldEqual, "web2")
			So(servers[1].Port, ShouldEqual, 8081)
		})

		Convey("空切片", func() {
			data := map[string]interface{}{
				"other": "value",
			}

			storage := NewFlatStorage(data)
			var servers []string
			err := storage.Sub("servers").ConvertTo(&servers)

			So(err, ShouldBeNil)
			So(len(servers), ShouldEqual, 0)
		})
	})
}

// TestFlatStorage_ConvertTo_Map 测试Map转换
func TestFlatStorage_ConvertTo_Map(t *testing.T) {
	Convey("FlatStorage Map转换测试", t, func() {
		Convey("简单map转换", func() {
			data := map[string]interface{}{
				"config.timeout": "30s",
				"config.retries": 3,
				"config.enabled": true,
			}

			storage := NewFlatStorage(data)
			var config map[string]interface{}
			err := storage.Sub("config").ConvertTo(&config)

			So(err, ShouldBeNil)
			So(len(config), ShouldEqual, 3)
			So(config["timeout"], ShouldEqual, "30s")
			So(config["retries"], ShouldEqual, 3)
			So(config["enabled"], ShouldEqual, true)
		})

		Convey("字符串map转换", func() {
			data := map[string]interface{}{
				"labels.env":  "production",
				"labels.team": "backend",
				"labels.app":  "api",
			}

			storage := NewFlatStorage(data)
			var labels map[string]string
			err := storage.Sub("labels").ConvertTo(&labels)

			So(err, ShouldBeNil)
			So(len(labels), ShouldEqual, 3)
			So(labels["env"], ShouldEqual, "production")
			So(labels["team"], ShouldEqual, "backend")
			So(labels["app"], ShouldEqual, "api")
		})

		Convey("嵌套map转换", func() {
			type DatabaseConfig struct {
				Host string `json:"host"`
				Port int    `json:"port"`
			}

			data := map[string]interface{}{
				"databases.primary.host":   "db1.example.com",
				"databases.primary.port":   5432,
				"databases.secondary.host": "db2.example.com",
				"databases.secondary.port": 5433,
			}

			storage := NewFlatStorage(data)
			var databases map[string]DatabaseConfig
			err := storage.Sub("databases").ConvertTo(&databases)

			So(err, ShouldBeNil)
			So(len(databases), ShouldEqual, 2)
			So(databases["primary"].Host, ShouldEqual, "db1.example.com")
			So(databases["primary"].Port, ShouldEqual, 5432)
			So(databases["secondary"].Host, ShouldEqual, "db2.example.com")
			So(databases["secondary"].Port, ShouldEqual, 5433)
		})

		Convey("直接转换flat map", func() {
			data := map[string]interface{}{
				"config.timeout": "30s",
				"config.retries": 3,
				"config.enabled": true,
			}

			storage := NewFlatStorage(data)
			var config map[string]interface{}
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)
			So(len(config), ShouldEqual, 3)
			So(config["config.timeout"], ShouldEqual, "30s")
			So(config["config.retries"], ShouldEqual, 3)
			So(config["config.enabled"], ShouldEqual, true)
		})

		Convey("直接多层转换flat map", func() {
			data := map[string]interface{}{
				"server.config.timeout": "30s",
				"server.config.retries": 3,
				"server.config.enabled": true,
			}

			storage := NewFlatStorage(data)
			var config map[string]interface{}
			err := storage.Sub("server").ConvertTo(&config)

			So(err, ShouldBeNil)
			So(len(config), ShouldEqual, 3)
			So(config["config.timeout"], ShouldEqual, "30s")
			So(config["config.retries"], ShouldEqual, 3)
			So(config["config.enabled"], ShouldEqual, true)
		})
	})
}

// TestFlatStorage_ConvertTo_Time 测试时间类型转换
func TestFlatStorage_ConvertTo_Time(t *testing.T) {
	Convey("FlatStorage 时间类型转换测试", t, func() {
		Convey("RFC3339 字符串转时间", func() {
			data := map[string]interface{}{
				"created_at": "2023-12-25T15:30:45Z",
			}
			storage := NewFlatStorage(data)

			var timeValue time.Time
			err := storage.Sub("created_at").ConvertTo(&timeValue)

			So(err, ShouldBeNil)
			expected := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
			So(timeValue.Equal(expected), ShouldBeTrue)
		})

		Convey("日期字符串转时间", func() {
			data := map[string]interface{}{
				"birth_date": "2023-12-25",
			}
			storage := NewFlatStorage(data)

			var timeValue time.Time
			err := storage.Sub("birth_date").ConvertTo(&timeValue)

			So(err, ShouldBeNil)
			expected := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
			So(timeValue.Equal(expected), ShouldBeTrue)
		})

		Convey("Unix时间戳转时间", func() {
			data := map[string]interface{}{
				"timestamp": int64(1703517045),
			}
			storage := NewFlatStorage(data)

			var timeValue time.Time
			err := storage.Sub("timestamp").ConvertTo(&timeValue)

			So(err, ShouldBeNil)
			expected := time.Unix(1703517045, 0)
			So(timeValue.Equal(expected), ShouldBeTrue)
		})

		Convey("浮点时间戳转时间", func() {
			data := map[string]interface{}{
				"timestamp": 1703517045.5,
			}
			storage := NewFlatStorage(data)

			var timeValue time.Time
			err := storage.Sub("timestamp").ConvertTo(&timeValue)

			So(err, ShouldBeNil)
			expected := time.Unix(1703517045, 500000000)
			So(timeValue.Equal(expected), ShouldBeTrue)
		})
	})
}

// TestFlatStorage_ConvertTo_Duration 测试Duration类型转换
func TestFlatStorage_ConvertTo_Duration(t *testing.T) {
	Convey("FlatStorage Duration类型转换测试", t, func() {
		Convey("字符串Duration", func() {
			data := map[string]interface{}{
				"timeout": "5m30s",
			}
			storage := NewFlatStorage(data)

			var duration time.Duration
			err := storage.Sub("timeout").ConvertTo(&duration)

			So(err, ShouldBeNil)
			So(duration, ShouldEqual, 5*time.Minute+30*time.Second)
		})

		Convey("小时Duration", func() {
			data := map[string]interface{}{
				"cache_ttl": "2h15m",
			}
			storage := NewFlatStorage(data)

			var duration time.Duration
			err := storage.Sub("cache_ttl").ConvertTo(&duration)

			So(err, ShouldBeNil)
			So(duration, ShouldEqual, 2*time.Hour+15*time.Minute)
		})

		Convey("纳秒整数", func() {
			data := map[string]interface{}{
				"delay": int64(1000000000),
			}
			storage := NewFlatStorage(data)

			var duration time.Duration
			err := storage.Sub("delay").ConvertTo(&duration)

			So(err, ShouldBeNil)
			So(duration, ShouldEqual, time.Second)
		})

		Convey("秒浮点数", func() {
			data := map[string]interface{}{
				"interval": 2.5,
			}
			storage := NewFlatStorage(data)

			var duration time.Duration
			err := storage.Sub("interval").ConvertTo(&duration)

			So(err, ShouldBeNil)
			So(duration, ShouldEqual, 2*time.Second+500*time.Millisecond)
		})
	})
}

// TestFlatStorage_ConvertTo_WithDefaults 测试默认值功能
func TestFlatStorage_ConvertTo_WithDefaults(t *testing.T) {
	Convey("FlatStorage 默认值功能测试", t, func() {
		type ServerConfig struct {
			Host     string        `json:"host" def:"localhost"`
			Port     int           `json:"port" def:"8080"`
			Enabled  bool          `json:"enabled" def:"true"`
			Timeout  time.Duration `json:"timeout" def:"30s"`
			MaxConns int           `json:"max_conns" def:"100"`
		}

		type AppConfig struct {
			Name   string       `json:"name" def:"MyApp"`
			Debug  bool         `json:"debug" def:"false"`
			Server ServerConfig `json:"server"`
		}

		Convey("完全空配置使用默认值", func() {
			data := map[string]interface{}{}
			storage := NewFlatStorage(data).WithDefaults(true)

			var config AppConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)

			So(config.Name, ShouldEqual, "MyApp")
			So(config.Debug, ShouldBeFalse)
			So(config.Server.Host, ShouldEqual, "localhost")
			So(config.Server.Port, ShouldEqual, 8080)
			So(config.Server.Enabled, ShouldBeTrue)
			So(config.Server.Timeout, ShouldEqual, 30*time.Second)
			So(config.Server.MaxConns, ShouldEqual, 100)
		})

		Convey("部分配置覆盖默认值", func() {
			data := map[string]interface{}{
				"name":        "CustomApp",
				"server.host": "custom.example.com",
				"server.port": 9090,
			}
			storage := NewFlatStorage(data).WithDefaults(true)

			var config AppConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)

			So(config.Name, ShouldEqual, "CustomApp")
			So(config.Server.Host, ShouldEqual, "custom.example.com")
			So(config.Server.Port, ShouldEqual, 9090)
			So(config.Server.Enabled, ShouldBeTrue)                // 使用默认值
			So(config.Server.Timeout, ShouldEqual, 30*time.Second) // 使用默认值
		})

		Convey("禁用默认值功能", func() {
			data := map[string]interface{}{}
			storage := NewFlatStorage(data).WithDefaults(false)

			var config AppConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)

			So(config.Name, ShouldEqual, "")
			So(config.Debug, ShouldBeFalse) // bool 的零值
			So(config.Server.Host, ShouldEqual, "")
			So(config.Server.Port, ShouldEqual, 0)
			So(config.Server.Enabled, ShouldBeFalse)
			So(config.Server.Timeout, ShouldEqual, 0)
		})
	})
}

// TestFlatStorage_CaseConversion 测试大小写转换功能
func TestFlatStorage_CaseConversion(t *testing.T) {
	Convey("FlatStorage 大小写转换功能测试", t, func() {
		type ServerConfig struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		}

		Convey("大写转换", func() {
			data := map[string]interface{}{
				"HOST": "localhost",
				"PORT": 8080,
			}
			storage := NewFlatStorage(data).WithUppercase(true)

			var config ServerConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)
			So(config.Host, ShouldEqual, "localhost")
			So(config.Port, ShouldEqual, 8080)
		})

		Convey("小写转换", func() {
			data := map[string]interface{}{
				"host": "localhost",
				"port": 8080,
			}
			storage := NewFlatStorage(data).WithLowercase(true)

			var config ServerConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)
			So(config.Host, ShouldEqual, "localhost")
			So(config.Port, ShouldEqual, 8080)
		})

		Convey("嵌套结构的大小写转换", func() {
			type AppConfig struct {
				Name   string       `json:"name"`
				Server ServerConfig `json:"server"`
			}

			data := map[string]interface{}{
				"NAME":        "test-app",
				"SERVER.HOST": "localhost",
				"SERVER.PORT": 3306,
			}
			storage := NewFlatStorage(data).WithUppercase(true)

			var config AppConfig
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)
			So(config.Name, ShouldEqual, "test-app")
			So(config.Server.Host, ShouldEqual, "localhost")
			So(config.Server.Port, ShouldEqual, 3306)
		})

		Convey("数组索引的大小写转换", func() {
			data := map[string]interface{}{
				"SERVERS.0": "server1",
				"SERVERS.1": "server2",
			}
			storage := NewFlatStorage(data).WithUppercase(true)

			var servers []string
			err := storage.Sub("servers").ConvertTo(&servers)
			So(err, ShouldBeNil)
			So(len(servers), ShouldEqual, 2)
			So(servers[0], ShouldEqual, "server1")
			So(servers[1], ShouldEqual, "server2")
		})

		Convey("Map的大小写转换", func() {
			data := map[string]interface{}{
				"CONFIG.TIMEOUT": "30s",
				"CONFIG.RETRIES": 3,
			}
			storage := NewFlatStorage(data).WithUppercase(true)

			var config map[string]interface{}
			err := storage.Sub("config").ConvertTo(&config)
			So(err, ShouldBeNil)
			So(config["TIMEOUT"], ShouldEqual, "30s")
			So(config["RETRIES"], ShouldEqual, 3)
		})
	})
}

// TestFlatStorage_Equals 测试 Equals 方法
func TestFlatStorage_Equals(t *testing.T) {
	Convey("FlatStorage Equals 方法测试", t, func() {
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

		Convey("相同数据的FlatStorage应该相等", func() {
			storage1 := NewFlatStorage(data1)
			storage2 := NewFlatStorage(data2)
			So(storage1.Equals(storage2), ShouldBeTrue)
		})

		Convey("不同数据的FlatStorage应该不相等", func() {
			storage1 := NewFlatStorage(data1)
			storage3 := NewFlatStorage(data3)
			So(storage1.Equals(storage3), ShouldBeFalse)
		})

		Convey("storage与自身应该相等", func() {
			storage1 := NewFlatStorage(data1)
			So(storage1.Equals(storage1), ShouldBeTrue)
		})

		Convey("相同配置选项的storage应该相等", func() {
			storage1 := NewFlatStorage(data1).WithDefaults(true).WithSeparator("-").WithUppercase(true)
			storage2 := NewFlatStorage(data2).WithDefaults(true).WithSeparator("-").WithUppercase(true)
			So(storage1.Equals(storage2), ShouldBeTrue)
		})

		Convey("不同配置选项的storage应该不相等", func() {
			storage1 := NewFlatStorage(data1).WithDefaults(true)
			storage2 := NewFlatStorage(data1).WithDefaults(false)
			So(storage1.Equals(storage2), ShouldBeFalse)

			storage3 := NewFlatStorage(data1).WithSeparator(".")
			storage4 := NewFlatStorage(data1).WithSeparator("-")
			So(storage3.Equals(storage4), ShouldBeFalse)

			storage5 := NewFlatStorage(data1).WithUppercase(true)
			storage6 := NewFlatStorage(data1).WithLowercase(true)
			So(storage5.Equals(storage6), ShouldBeFalse)
		})

		Convey("SubStorage的比较", func() {
			complexData := map[string]interface{}{
				"database.host": "localhost",
				"database.port": 3306,
			}

			storage1 := NewFlatStorage(complexData)
			storage2 := NewFlatStorage(complexData)

			sub1 := storage1.Sub("database")
			sub2 := storage2.Sub("database")

			So(sub1.Equals(sub2), ShouldBeTrue)
		})

		Convey("空数据的比较", func() {
			empty1 := NewFlatStorage(nil)
			empty2 := NewFlatStorage(nil)
			emptyMap1 := NewFlatStorage(map[string]interface{}{})
			emptyMap2 := NewFlatStorage(map[string]interface{}{})

			So(empty1.Equals(empty2), ShouldBeTrue)
			So(emptyMap1.Equals(emptyMap2), ShouldBeTrue)
			So(empty1.Equals(emptyMap1), ShouldBeFalse) // nil vs empty map
		})
	})
}

// TestFlatStorage_Equals_DifferentTypes 测试不同类型的比较
func TestFlatStorage_Equals_DifferentTypes(t *testing.T) {
	Convey("FlatStorage 不同类型比较测试", t, func() {
		// 使用map_storage_test.go中已定义的MockStorage
		storage := NewFlatStorage(testFlatData)
		mockStorage := &MockStorage{}

		Convey("FlatStorage与其他类型的Storage比较应该返回false", func() {
			So(storage.Equals(mockStorage), ShouldBeFalse)
		})
	})
}

// TestFlatStorage_Sub_PathAccess 测试路径访问功能
func TestFlatStorage_Sub_PathAccess(t *testing.T) {
	Convey("FlatStorage 路径访问测试", t, func() {
		storage := NewFlatStorage(testFlatData)

		Convey("空key返回自身", func() {
			result := storage.Sub("")
			So(result, ShouldEqual, storage)
		})

		Convey("访问存在的键", func() {
			result := storage.Sub("database.host")
			var host string
			err := result.ConvertTo(&host)
			So(err, ShouldBeNil)
			So(host, ShouldEqual, "localhost")
		})

		Convey("访问不存在的键", func() {
			result := storage.Sub("nonexistent.key")
			var value string
			err := result.ConvertTo(&value)
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "") // 零值
		})

		Convey("嵌套Sub调用", func() {
			result1 := storage.Sub("database")
			result2 := result1.Sub("host")
			var host string
			err := result2.ConvertTo(&host)
			So(err, ShouldBeNil)
			So(host, ShouldEqual, "localhost")
		})
	})
}

// TestFlatStorage_EdgeCases 测试边界情况
func TestFlatStorage_EdgeCases(t *testing.T) {
	Convey("FlatStorage 边界情况测试", t, func() {
		Convey("空字符串分隔符", func() {
			data := map[string]interface{}{
				"host": "localhost",
			}
			storage := NewFlatStorage(data).WithSeparator("")

			var host string
			err := storage.Sub("host").ConvertTo(&host)
			So(err, ShouldBeNil)
			So(host, ShouldEqual, "localhost")
		})

		Convey("特殊字符作为分隔符", func() {
			data := map[string]interface{}{
				"database#host": "localhost",
				"database#port": 3306,
			}
			storage := NewFlatStorage(data).WithSeparator("#")

			var host string
			err := storage.Sub("database").Sub("host").ConvertTo(&host)
			So(err, ShouldBeNil)
			So(host, ShouldEqual, "localhost")
		})

		Convey("nil storage处理", func() {
			var storage *FlatStorage = nil
			result := storage.WithDefaults(true)
			So(result, ShouldBeNil)

			err := storage.ConvertTo(nil)
			So(err, ShouldBeNil)
		})

		Convey("接口类型转换", func() {
			data := map[string]interface{}{
				"value": "hello",
			}
			storage := NewFlatStorage(data)

			var result interface{}
			err := storage.Sub("value").ConvertTo(&result)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "hello")
		})

		Convey("指针字段处理", func() {
			type Config struct {
				Host *string `json:"host"`
				Port *int    `json:"port"`
			}

			data := map[string]interface{}{
				"host": "localhost",
				"port": 8080,
			}
			storage := NewFlatStorage(data)

			var config Config
			err := storage.ConvertTo(&config)
			So(err, ShouldBeNil)
			So(config.Host, ShouldNotBeNil)
			So(*config.Host, ShouldEqual, "localhost")
			So(config.Port, ShouldNotBeNil)
			So(*config.Port, ShouldEqual, 8080)
		})

		Convey("复杂嵌套结构", func() {
			data := map[string]interface{}{
				"app.name":                   "test-app",
				"app.services.0.name":        "web",
				"app.services.0.endpoints.0": "http://localhost:8080",
				"app.services.0.endpoints.1": "http://localhost:8081",
				"app.services.1.name":        "api",
				"app.services.1.endpoints.0": "http://localhost:9090",
			}

			type Service struct {
				Name      string   `json:"name"`
				Endpoints []string `json:"endpoints"`
			}

			type App struct {
				Name     string    `json:"name"`
				Services []Service `json:"services"`
			}

			storage := NewFlatStorage(data)
			var app App
			err := storage.Sub("app").ConvertTo(&app)

			So(err, ShouldBeNil)
			So(app.Name, ShouldEqual, "test-app")
			So(len(app.Services), ShouldEqual, 2)
			So(app.Services[0].Name, ShouldEqual, "web")
			So(len(app.Services[0].Endpoints), ShouldEqual, 2)
			So(app.Services[0].Endpoints[0], ShouldEqual, "http://localhost:8080")
			So(app.Services[1].Name, ShouldEqual, "api")
			So(len(app.Services[1].Endpoints), ShouldEqual, 1)
		})
	})
}

// TestFlatStorage_ConvertTo_ComplexNestedStructure 测试复杂嵌套结构转换
// 包含结构体中有map和slice，而slice/map中也包含结构体的情况
func TestFlatStorage_ConvertTo_ComplexNestedStructure(t *testing.T) {
	Convey("FlatStorage 复杂嵌套结构转换测试", t, func() {
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

		// 构造复杂的扁平化测试数据
		flatData := map[string]interface{}{
			"application.name":    "complex-app",
			"application.version": "1.0.0",

			// 第一个服务
			"services.0.name":                          "auth-service",
			"services.0.enabled":                       true,
			"services.0.endpoints.0.url":               "https://auth.example.com/login",
			"services.0.endpoints.0.timeout":           "30s",
			"services.0.endpoints.0.retries":           3,
			"services.0.endpoints.1.url":               "https://auth.example.com/logout",
			"services.0.endpoints.1.timeout":           "15s",
			"services.0.endpoints.1.retries":           1,
			"services.0.metadata.team":                 "security",
			"services.0.metadata.environment":          "production",
			"services.0.advanced.health_check.url":     "https://auth.example.com/health",
			"services.0.advanced.health_check.timeout": "5s",
			"services.0.advanced.health_check.retries": 2,
			"services.0.advanced.metrics.url":          "https://auth.example.com/metrics",
			"services.0.advanced.metrics.timeout":      "10s",
			"services.0.advanced.metrics.retries":      1,

			// 第二个服务
			"services.1.name":                "notification-service",
			"services.1.enabled":             false,
			"services.1.endpoints.0.url":     "https://notify.example.com/send",
			"services.1.endpoints.0.timeout": "60s",
			"services.1.endpoints.0.retries": 5,
			"services.1.metadata.team":       "messaging",

			// 数据库配置
			"databases.primary.host":          "primary-db.example.com",
			"databases.primary.port":          5432,
			"databases.primary.database":      "app_production",
			"databases.primary.pool.min_size": 5,
			"databases.primary.pool.max_size": 20,
			"databases.cache.host":            "cache.example.com",
			"databases.cache.port":            6379,
			"databases.cache.database":        "0",
			"databases.cache.pool.min_size":   2,
			"databases.cache.pool.max_size":   10,

			// 环境变量
			"environment.stage":      "production",
			"environment.debug":      false,
			"environment.log_level":  "info",
			"environment.max_memory": "2GB",

			// 特性配置
			"features.experimental.0": "feature-a",
			"features.experimental.1": "feature-b",
			"features.stable.0":       "feature-x",
			"features.stable.1":       "feature-y",
			"features.stable.2":       "feature-z",
		}

		Convey("完整复杂结构转换", func() {
			storage := NewFlatStorage(flatData)
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
				So(len(config.Features), ShouldEqual, 2)

				experimental, exists := config.Features["experimental"]
				So(exists, ShouldBeTrue)
				So(len(experimental), ShouldEqual, 2)
				So(experimental[0], ShouldEqual, "feature-a")
				So(experimental[1], ShouldEqual, "feature-b")

				stable, exists := config.Features["stable"]
				So(exists, ShouldBeTrue)
				So(len(stable), ShouldEqual, 3)
			})
		})

		Convey("分隔符自定义测试", func() {
			// 使用不同分隔符的扁平化数据
			customSeparatorData := map[string]interface{}{
				"application-name":    "custom-app",
				"application-version": "2.0.0",
				"services-0-name":     "test-service",
				"services-0-enabled":  true,
			}

			storage := NewFlatStorage(customSeparatorData).WithSeparator("-")
			var config ComplexConfig
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)
			So(config.Application.Name, ShouldEqual, "custom-app")
			So(config.Application.Version, ShouldEqual, "2.0.0")
			So(len(config.Services), ShouldEqual, 1)
			So(config.Services[0].Name, ShouldEqual, "test-service")
			So(config.Services[0].Enabled, ShouldBeTrue)
		})

		Convey("大小写转换测试", func() {
			uppercaseData := map[string]interface{}{
				"APPLICATION.NAME":    "UPPERCASE-APP",
				"APPLICATION.VERSION": "3.0.0",
				"SERVICES.0.NAME":     "UPPER-SERVICE",
				"SERVICES.0.ENABLED":  true,
			}

			storage := NewFlatStorage(uppercaseData).WithUppercase(true)
			var config ComplexConfig
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)
			So(config.Application.Name, ShouldEqual, "UPPERCASE-APP")
			So(config.Application.Version, ShouldEqual, "3.0.0")
			So(len(config.Services), ShouldEqual, 1)
			So(config.Services[0].Name, ShouldEqual, "UPPER-SERVICE")
			So(config.Services[0].Enabled, ShouldBeTrue)
		})

		Convey("默认值和复杂结构混合测试", func() {
			type ServiceWithDefaults struct {
				Name      string            `json:"name" def:"default-service"`
				Enabled   bool              `json:"enabled" def:"false"`
				Endpoints []Endpoint        `json:"endpoints"`
				Metadata  map[string]string `json:"metadata"`
			}

			type ConfigWithDefaults struct {
				Application struct {
					Name    string `json:"name" def:"default-app"`
					Version string `json:"version" def:"1.0.0"`
				} `json:"application"`
				Services []ServiceWithDefaults `json:"services"`
			}

			partialData := map[string]interface{}{
				"services.0.name":                "partial-service",
				"services.0.endpoints.0.url":     "http://example.com",
				"services.0.endpoints.0.timeout": "30s",
				"services.0.endpoints.0.retries": 3,
			}

			storage := NewFlatStorage(partialData).WithDefaults(true)
			var config ConfigWithDefaults
			err := storage.ConvertTo(&config)

			So(err, ShouldBeNil)
			So(config.Application.Name, ShouldEqual, "default-app")
			So(config.Application.Version, ShouldEqual, "1.0.0")
			So(len(config.Services), ShouldEqual, 1)
			So(config.Services[0].Name, ShouldEqual, "partial-service")
			So(config.Services[0].Enabled, ShouldBeFalse) // 使用默认值
			So(len(config.Services[0].Endpoints), ShouldEqual, 1)
			So(config.Services[0].Endpoints[0].URL, ShouldEqual, "http://example.com")
		})
	})
}
