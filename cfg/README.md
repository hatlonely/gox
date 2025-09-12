# Config 配置管理库

一个功能强大、易于使用的 Go 配置管理库，支持多种配置格式和实时配置变更监听。

## 特性

- 🔧 **多格式支持**: JSON、YAML、TOML、INI、ENV 格式
- 🎯 **类型安全**: 自动类型转换和结构体绑定
- 🔄 **实时监听**: 配置文件变更自动重载
- 📊 **层级访问**: 支持嵌套配置和数组索引访问
- ⚡ **简单易用**: 一行代码即可开始使用

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
)

func main() {
    // 从配置文件创建配置对象（自动识别格式）
    config, err := cfg.NewConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    defer config.provider.Close()
    
    // 读取配置到结构体
    type DatabaseConfig struct {
        Host string `yaml:"host"`
        Port int    `yaml:"port"`
    }
    
    dbConfig := config.Sub("database")
    var db DatabaseConfig
    dbConfig.ConvertTo(&db)
    
    fmt.Printf("Database: %s:%d\n", db.Host, db.Port)
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
// 监听整个配置变更
config.OnChange(func(c *cfg.Config) error {
    fmt.Println("Config changed!")
    return nil
})

// 监听特定键变更
config.OnKeyChange("database", func(c *cfg.Config) error {
    var db DatabaseConfig
    c.ConvertTo(&db)
    fmt.Printf("Database config changed: %+v\n", db)
    return nil
})

// 子配置监听（等价于 OnKeyChange）
dbConfig := config.Sub("database")
dbConfig.OnChange(func(c *cfg.Config) error {
    fmt.Println("Database config changed!")
    return nil
})
```

### 4. 类型转换

库支持自动类型转换，包括：

- 基础类型：`string`, `int`, `float`, `bool`
- 时间类型：`time.Duration`, `time.Time`
- 复合类型：`map`, `slice`, `struct`

```go
// 自动转换时间类型
type Config struct {
    Timeout  time.Duration `yaml:"timeout"`   // "30s" -> 30 * time.Second
    Created  time.Time     `yaml:"created"`   // "2023-01-01" -> time.Time
}
```

## 高级用法

### 自定义 Provider 和 Decoder

```go
import (
    "github.com/hatlonely/gox/cfg"
    "github.com/hatlonely/gox/cfg/provider"
    "github.com/hatlonely/gox/cfg/decoder"
    "github.com/hatlonely/gox/refx"
)

options := &cfg.Options{
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

config, err := cfg.NewConfigWithOptions(options)
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
config, err := cfg.NewConfig("config.yaml")
if err != nil {
    log.Fatal(err)
}

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
config, _ := cfg.NewConfig("config.yaml")

// 监听配置变更并重启服务组件
config.OnKeyChange("server", func(c *cfg.Config) error {
    var serverConfig ServerConfig
    c.ConvertTo(&serverConfig)
    
    // 重启 HTTP 服务器
    return restartServer(serverConfig)
})
```

## 支持的标签

配置库支持多种结构体标签进行字段映射：

- `cfg:"field_name"` - 优先级最高
- `json:"field_name"` 
- `yaml:"field_name"`
- `toml:"field_name"`
- `ini:"field_name"`

```go
type Config struct {
    Host string `cfg:"host" json:"host" yaml:"host"`
    Port int    `cfg:"port" json:"port" yaml:"port"`
}
```

## 许可证

MIT License