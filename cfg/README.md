# Config 配置管理库

一个功能强大、易于使用的 Go 配置管理库，支持多种配置格式和实时配置变更监听。

## 特性

- 🔧 **多格式支持**: JSON、YAML、TOML、INI、ENV 格式
- 🎯 **类型安全**: 自动类型转换和结构体绑定
- 🔄 **实时监听**: 配置文件变更自动重载，支持延迟初始化
- 📊 **层级访问**: 支持嵌套配置和数组索引访问
- ⚡ **简单易用**: 一行代码即可开始使用
- 🏗️ **接口驱动**: 基于 Config 接口的设计，支持多种实现

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
    "github.com/hatlonely/gox/refx"
)

options := &cfg.SingleConfigOptions{
    Provider: refx.TypeOptions{
        Namespace: "github.com/hatlonely/gox/cfg/provider",
        Type:      "FileProvider",
        Options: &provider.FileProviderOptions{
            FilePath: "config.yaml",
        },
    },
    Decoder: refx.TypeOptions{
        Namespace: "github.com/hatlonely/gox/cfg/decoder",
        Type:      "YamlDecoder",
        Options:   &decoder.YamlDecoderOptions{Indent: 2},
    },
}

config, err := cfg.NewSingleConfigWithOptions(options)
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

## 许可证

MIT License