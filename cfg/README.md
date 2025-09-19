# Config 配置管理库

一个功能强大、易于使用的 Go 配置管理库，支持多种配置格式和实时配置变更监听。

## 特性

- 🔧 **多格式支持**: JSON、YAML、TOML、INI、ENV 格式
- 🎯 **类型安全**: 自动类型转换和结构体绑定
- 🔄 **实时监听**: 配置文件变更自动重载，支持延迟初始化
- 📊 **层级访问**: 支持嵌套配置和数组索引访问
- ⚡ **简单易用**: 一行代码即可开始使用
- 🏗️ **接口驱动**: 基于 Config 接口的设计，支持多种实现
- 🔀 **多源合并**: 支持多个配置源按优先级合并（MultiConfig）

## 快速开始

### 安装

```bash
go get github.com/hatlonely/gox/cfg
```

### 基本用法

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/hatlonely/gox/cfg"
    "github.com/hatlonely/gox/cfg/storage"
)

func main() {
    // 从配置文件创建配置对象（自动识别格式）
    config, err := cfg.NewSingleConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    defer config.Close() // 确保资源释放
    
    // 读取配置到结构体
    type DatabaseConfig struct {
        Host string `yaml:"host"`
        Port int    `yaml:"port"`
    }
    
    dbConfig := config.Sub("database")
    var db DatabaseConfig
    dbConfig.ConvertTo(&db)
    
    fmt.Printf("Database: %s:%d\n", db.Host, db.Port)
    
    // 可选：启动配置监听
    // config.OnChange(func(s storage.Storage) error { ... })
    // config.Watch()
}
```

## 配置格式支持

### YAML 配置 (config.yaml)

```yaml
database:
  host: localhost
  port: 3306
  timeout: "30s"

servers:
  - name: web1
    port: 8080
  - name: web2
    port: 8081
```

### JSON 配置 (config.json)

```json
{
  "database": {
    "host": "localhost",
    "port": 3306,
    "timeout": "30s"
  },
  "servers": [
    {"name": "web1", "port": 8080},
    {"name": "web2", "port": 8081}
  ]
}
```

### TOML 配置 (config.toml)

```toml
[database]
host = "localhost"
port = 3306
timeout = "30s"

[[servers]]
name = "web1"
port = 8080

[[servers]]
name = "web2"
port = 8081
```

### INI 配置 (config.ini)

```ini
[database]
host = localhost
port = 3306
timeout = 30s
```

### ENV 配置 (config.env)

```env
DATABASE_HOST=localhost
DATABASE_PORT=3306
DATABASE_TIMEOUT=30s
```

## 核心接口

### Config 接口

配置库基于 `Config` 接口设计，提供统一的配置访问方式：

```go
type Config interface {
    // 获取子配置对象
    Sub(key string) Config
    
    // 将配置数据转成结构体或 map/slice 等
    ConvertTo(object any) error
    
    // 设置日志记录器
    SetLogger(logger log.Logger)
    
    // 监听配置变更（回调参数为 storage.Storage）
    OnChange(fn func(storage.Storage) error)
    
    // 监听指定键的配置变更
    OnKeyChange(key string, fn func(storage.Storage) error)
    
    // 启动配置变更监听
    Watch() error
    
    // 关闭配置对象，释放相关资源
    Close() error
}
```

### SingleConfig 实现

`SingleConfig` 是 `Config` 接口的默认实现，提供完整的配置管理功能。

### MultiConfig 实现

`MultiConfig` 支持多配置源合并，按优先级覆盖配置，适用于多环境配置管理。

```go
import (
    "github.com/hatlonely/gox/cfg"
    "github.com/hatlonely/gox/cfg/provider"
    "github.com/hatlonely/gox/ref"
)

// 创建多配置源：基础配置 + 环境配置 + 数据库配置
multiConfig, err := cfg.NewMultiConfigWithOptions(&cfg.MultiConfigOptions{
    Sources: []*cfg.ConfigSourceOptions{
        {
            // 基础配置文件（优先级最低）
            Provider: ref.TypeOptions{
                Type: "FileProvider",
                Options: &provider.FileProviderOptions{FilePath: "config.yaml"},
            },
            Decoder: ref.TypeOptions{Type: "YamlDecoder"},
        },
        {
            // 环境变量覆盖（中等优先级）
            Provider: ref.TypeOptions{
                Type: "EnvProvider",
                Options: &provider.EnvProviderOptions{EnvFiles: []string{}},
            },
            Decoder: ref.TypeOptions{Type: "EnvDecoder"},
        },
        {
            // 数据库配置（优先级最高）
            Provider: ref.TypeOptions{
                Type: "GormProvider",
                Options: &provider.GormProviderOptions{
                    DSN: "postgres://...",
                    Table: "app_configs",
                },
            },
            Decoder: ref.TypeOptions{Type: "JsonDecoder"},
        },
    },
})

// 使用方式与 SingleConfig 完全相同
var app AppConfig
multiConfig.ConvertTo(&app)

// 监听任意配置源的变更
multiConfig.OnChange(func(s storage.Storage) error {
    // 重新加载配置
    return nil
})
multiConfig.Watch()
```

**典型使用场景：**
- **多环境部署**: 基础配置 + 环境特定配置（dev/test/prod）
- **配置分层管理**: 默认配置 + 环境变量覆盖 + 运行时配置
- **动态配置中心**: 本地配置文件 + 远程配置服务

## 核心功能

### 1. 层级配置访问

```go
// 访问嵌套配置
dbConfig := config.Sub("database")
host := config.Sub("database.host")

// 访问数组元素
server1 := config.Sub("servers[0]")
serverName := config.Sub("servers[0].name")
```

### 2. 结构体绑定

```go
type AppConfig struct {
    Database struct {
        Host    string        `yaml:"host"`
        Port    int           `yaml:"port"`
        Timeout time.Duration `yaml:"timeout"`
    } `yaml:"database"`
    
    Servers []struct {
        Name string `yaml:"name"`
        Port int    `yaml:"port"`
    } `yaml:"servers"`
}

var app AppConfig
config.ConvertTo(&app)
```

### 3. 配置变更监听

```go
import "github.com/hatlonely/gox/cfg/storage"

// 注册变更回调函数（参数为 storage.Storage）
config.OnChange(func(s storage.Storage) error {
    fmt.Println("Configuration changed!")
    
    // 直接操作存储层数据
    var data map[string]any
    if err := s.ConvertTo(&data); err != nil {
        return err
    }
    fmt.Printf("New config: %+v\n", data)
    return nil
})

// 监听特定键变更
config.OnKeyChange("database", func(s storage.Storage) error {
    var db DatabaseConfig
    s.ConvertTo(&db)
    fmt.Printf("Database config changed: %+v\n", db)
    return nil
})

// 子配置监听（等价于 OnKeyChange）
dbConfig := config.Sub("database")
dbConfig.OnChange(func(s storage.Storage) error {
    fmt.Println("Database config changed!")
    
    // 可以访问子存储的任意路径
    hostStorage := s.Sub("host")
    var host string
    hostStorage.ConvertTo(&host)
    fmt.Printf("New host: %s\n", host)
    return nil
})

// 启动监听（必须调用才会真正开始监听）
config.Watch()
```

**Storage 参数优势：**
- **直接数据访问**: 回调函数接收 `storage.Storage` 接口，可直接操作配置数据
- **高性能**: 避免了 Config 到 Storage 的类型转换开销
- **灵活操作**: 可使用 Storage 的所有方法（Sub、ConvertTo、Equals）
- **简化代码**: 减少了中间层的复杂性

**监听机制说明：**
- `OnChange/OnKeyChange`: 仅注册回调函数，不启动监听
- `Watch()`: 真正启动监听，只有调用后回调才会被触发
- **延迟初始化**: 监听器在第一次调用 Watch 时才初始化
- **线程安全**: 多次调用 Watch 是安全的
```

### 4. 类型转换

库支持自动类型转换，包括：

- 基础类型：`string`, `int`, `float`, `bool`
- 时间类型：`time.Duration`, `time.Time`
- 复合类型：`map`, `slice`, `struct`

```go
// 自动转换时间类型
type SingleConfig struct {
    Timeout  time.Duration `yaml:"timeout"`   // "30s" -> 30 * time.Second
    Created  time.Time     `yaml:"created"`   // "2023-01-01" -> time.Time
}
```

### 5. 资源管理

配置对象可能会持有一些资源（如文件监听器、数据库连接等），使用完毕后应该调用 Close 方法释放资源。

```go
config, err := cfg.NewSingleConfig("config.yaml")
if err != nil {
    log.Fatal(err)
}
defer config.Close() // 释放资源

// 子配置也可以调用 Close，会自动转发到根配置
dbConfig := config.Sub("database")
defer dbConfig.Close() // 等价于 config.Close()
```

**重要特性：**
- Close 方法只会执行一次，多次调用会返回第一次调用的结果
- 线程安全，支持并发调用
- 子配置和根配置的 Close 调用会产生同样的结果

## 高级用法

### 自定义 Provider 和 Decoder

```go
import (
    "github.com/hatlonely/gox/cfg"
    "github.com/hatlonely/gox/cfg/provider"
    "github.com/hatlonely/gox/cfg/decoder"
    "github.com/hatlonely/gox/ref"
)

options := &cfg.SingleConfigOptions{
    Provider: ref.TypeOptions{
        Namespace: "github.com/hatlonely/gox/cfg/provider",
        Type:      "FileProvider",
        Options: &provider.FileProviderOptions{
            FilePath: "config.yaml",
        },
    },
    Decoder: ref.TypeOptions{
        Namespace: "github.com/hatlonely/gox/cfg/decoder",
        Type:      "YamlDecoder",
        Options:   &decoder.YamlDecoderOptions{Indent: 2},
    },
}

config, err := cfg.NewSingleConfigWithOptions(options)
```

### 多配置源合并 (MultiConfig)

```go
// 简化配置：基础 + 环境变量
multiConfig, err := cfg.NewMultiConfigWithOptions(&cfg.MultiConfigOptions{
    Sources: []*cfg.ConfigSourceOptions{
        {
            Provider: ref.TypeOptions{Type: "FileProvider", Options: &provider.FileProviderOptions{FilePath: "config.yaml"}},
            Decoder:  ref.TypeOptions{Type: "YamlDecoder"},
        },
        {
            Provider: ref.TypeOptions{Type: "EnvProvider", Options: &provider.EnvProviderOptions{}},
            Decoder:  ref.TypeOptions{Type: "EnvDecoder"},
        },
    },
})
// 后续使用与 SingleConfig 完全相同
```

## 最佳实践

### 1. 配置结构体设计

```go
type AppConfig struct {
    Server struct {
        Host string `yaml:"host" default:"localhost"`
        Port int    `yaml:"port" default:"8080"`
    } `yaml:"server"`
    
    Database struct {
        DSN         string        `yaml:"dsn"`
        MaxConns    int           `yaml:"max_conns" default:"10"`
        Timeout     time.Duration `yaml:"timeout" default:"30s"`
    } `yaml:"database"`
    
    Logger struct {
        Level  string `yaml:"level" default:"info"`
        Output string `yaml:"output" default:"stdout"`
    } `yaml:"logger"`
}
```

### 2. 配置验证

```go
config, err := cfg.NewSingleConfig("config.yaml")
if err != nil {
    log.Fatal(err)
}
defer config.Close() // 确保资源释放

var app AppConfig
if err := config.ConvertTo(&app); err != nil {
    log.Fatal("配置格式错误:", err)
}

// 验证必要配置
if app.Database.DSN == "" {
    log.Fatal("数据库 DSN 不能为空")
}
```

### 3. 配置热重载

```go
import "github.com/hatlonely/gox/cfg/storage"

config, _ := cfg.NewSingleConfig("config.yaml")

// 注册配置变更监听（使用 storage.Storage 参数）
config.OnKeyChange("server", func(s storage.Storage) error {
    var serverConfig ServerConfig
    s.ConvertTo(&serverConfig)
    
    // 重启 HTTP 服务器
    return restartServer(serverConfig)
})

// 启动监听
config.Watch()
```

## 支持的标签

### 字段映射标签

配置库支持多种结构体标签进行字段映射：

- `cfg:"field_name"` - 优先级最高
- `json:"field_name"` 
- `yaml:"field_name"`
- `toml:"field_name"`
- `ini:"field_name"`

```go
type SingleConfig struct {
    Host string `cfg:"host" json:"host" yaml:"host"`
    Port int    `cfg:"port" json:"port" yaml:"port"`
}
```

### def 标签 - 默认值设置

`def` 标签用于为结构体字段设置默认值。使用 `cfg.SetDefaults()` 函数可以自动为零值字段设置默认值。

```go
package main

import (
    "fmt"
    "time"
    "github.com/hatlonely/gox/cfg"
)

type AppConfig struct {
    Name        string        `def:"MyApp"`
    Port        int           `def:"8080"`
    Timeout     time.Duration `def:"30s"`
    Debug       bool          `def:"true"`
    Tags        []string      `def:"web,api,service"`
    CreatedAt   time.Time     `def:"2023-01-01T00:00:00Z"`
    
    // 嵌套结构体会自动递归处理
    Database DatabaseConfig
    
    // 指针结构体只有在非空时才会递归处理
    Cache *CacheConfig
}

type DatabaseConfig struct {
    Host     string `def:"localhost"`
    Port     int    `def:"3306"`
    Username string `def:"root"`
}

func main() {
    config := &AppConfig{}
    
    // 设置默认值
    if err := cfg.SetDefaults(config); err != nil {
        panic(err)
    }
    
    fmt.Printf("Name: %s, Port: %d\n", config.Name, config.Port)
    fmt.Printf("Database: %s:%d\n", config.Database.Host, config.Database.Port)
}
```

**支持的类型：**
- 基础类型：`string`, `bool`, `int/uint` 系列, `float32/float64`
- 时间类型：`time.Duration`, `time.Time`（支持多种格式）
- 切片类型：逗号分隔的值（如：`"a,b,c"`）
- 指针类型：自动分配内存（仅对有 def 标签的字段）

**注意事项：**
- 只有零值字段才会被设置默认值
- 结构体字段会自动递归处理（无需 def 标签）
- 指针结构体字段只有在非空时才会递归处理

### help 标签 - 帮助文档生成

配置库支持多种标签来自动生成详细的配置帮助文档：

```go
package main

import (
    "fmt"
    "time"
    "github.com/hatlonely/gox/cfg"
)

type DatabaseConfig struct {
    Host     string `cfg:"host" help:"数据库主机地址" eg:"localhost" def:"127.0.0.1" validate:"required,hostname"`
    Port     int    `cfg:"port" help:"数据库端口号" eg:"3306" def:"5432" validate:"required,min=1,max=65535"`
    Username string `cfg:"username" help:"数据库用户名" eg:"admin" def:"postgres" validate:"required"`
    Password string `cfg:"password" help:"数据库密码" validate:"required,min=8"`
}

type AppConfig struct {
    Name     string        `cfg:"name" help:"应用名称" eg:"my-app" def:"demo" validate:"required,min=3"`
    Port     int           `cfg:"port" help:"服务端口" eg:"8080" def:"3000" validate:"required,min=1000"`
    Debug    bool          `cfg:"debug" help:"调试模式" def:"false"`
    Timeout  time.Duration `cfg:"timeout" help:"超时时间" eg:"30s" def:"10s"`
    Database DatabaseConfig `cfg:"database" help:"数据库配置"`
}

func main() {
    config := &AppConfig{}
    
    // 生成帮助文档
    help := cfg.GenerateHelp(config, "APP_", "app-")
    fmt.Println(help)
}
```

**支持的帮助标签：**

| 标签 | 说明 | 示例 |
|------|------|------|
| `help` | 字段说明文本 | `help:"应用名称"` |
| `eg` | 示例值 | `eg:"my-app"` |
| `def` | 默认值（也用于SetDefaults） | `def:"demo"` |
| `validate` | 校验规则 | `validate:"required,min=3"` |

**输出示例：**
```
配置参数说明：

  name (string) [必填]
    说明: 应用名称
    校验规则: 必填; 最小值: 3
    环境变量: APP_NAME
    命令行参数: --app-name
    默认值: demo
    示例: my-app

  database.host (string) [必填]
    说明: 数据库主机地址
    校验规则: 必填; 主机名格式
    环境变量: APP_DATABASE_HOST
    命令行参数: --app-database-host
    默认值: 127.0.0.1
    示例: localhost
```

**功能特点：**
- 自动生成环境变量和命令行参数名称
- 支持嵌套结构体的递归文档生成
- 智能解析 validate 标签并格式化校验规则
- 包含类型说明和配置优先级说明
- 保持结构体字段的原始定义顺序

## 许可证

MIT License