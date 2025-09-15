# 配置帮助信息生成器

## 功能说明

[GenerateHelp](file:///Users/hatlonely/Documents/github.com/hatlonely/gox/cfg/help.go#L24-L56) 方法可以通过反射分析配置结构体，自动生成包含环境变量和命令行参数映射关系的帮助信息。支持所有类型的字段，包括：

- 基本类型：string, int, bool, float 等
- 时间类型：time.Duration, time.Time
- 复合类型：slice, map, struct
- 指针类型和嵌套结构体

## 使用方法

```go
package main

import (
    "fmt"
    "time"
    "github.com/hatlonely/gox/cfg"
)

type DatabaseConfig struct {
    Host     string        `cfg:"host" help:"数据库主机地址"`
    Port     int           `cfg:"port" help:"数据库端口号"`
    Username string        `cfg:"username" help:"数据库用户名"`
    Password string        `cfg:"password" help:"数据库密码"`
    Timeout  time.Duration `cfg:"timeout" help:"连接超时时间"`
}

type AppConfig struct {
    Name     string            `cfg:"name" help:"应用名称"`
    Debug    bool              `cfg:"debug" help:"是否启用调试模式"`
    Database DatabaseConfig    `cfg:"database" help:"数据库配置"`
    Pools    []DatabaseConfig  `cfg:"pools" help:"数据库连接池列表"`
    Cache    map[string]string `cfg:"cache" help:"缓存配置映射"`
}

func main() {
    config := AppConfig{}
    
    // 生成帮助信息
    help := cfg.GenerateHelp(&config, "APP_", "app-")
    fmt.Print(help)
}
```

## 标签优先级

字段配置名称按以下优先级确定：

1. `cfg` 标签（最高优先级）
2. `json` 标签
3. `yaml` 标签
4. `toml` 标签
5. `ini` 标签
6. 字段名（最低优先级）

## 支持的复杂类型

### 切片类型 (slice)

```go
type Config struct {
    Pools []DatabasePool `cfg:"pools" help:"数据库连接池列表"`
}
```

配置格式：
- 环境变量：`APP_POOLS_0_HOST=db1.com`, `APP_POOLS_1_HOST=db2.com`
- 命令行：`--app-pools-0-host=db1.com --app-pools-1-host=db2.com`

### 映射类型 (map)

```go
type Config struct {
    Cache    map[string]string `cfg:"cache" help:"缓存配置"`
    Features map[string]bool   `cfg:"features" help:"功能开关"`
}
```

配置格式：
- 环境变量：`APP_CACHE_REDIS_HOST=localhost`, `APP_CACHE_MEMCACHED_HOST=mc.com`
- 命令行：`--app-cache-redis-host=localhost --app-cache-memcached-host=mc.com`

### 时间类型

#### time.Duration
支持的格式：
- 字符串：`"30s"`, `"5m"`, `"1h30m"`
- 纳秒数值：`1000000000` (1秒)
- 秒数浮点：`2.5` (2.5秒)

#### time.Time  
支持的格式：
- RFC3339：`"2023-12-25T15:30:45Z"`
- 日期：`"2023-12-25"`
- 日期时间：`"2023-12-25 15:30:45"`
- Unix时间戳：`1703517045`

## help 标签

使用 `help` 标签为字段提供说明信息：

```go
type Config struct {
    Host    string `cfg:"host" help:"服务器绑定地址，默认 0.0.0.0"`
    Port    int    `cfg:"port" help:"服务器监听端口，范围 1-65535"`
    Timeout time.Duration `cfg:"timeout" help:"连接超时时间，建议 30s"`
}
```

## 生成的帮助信息格式

```
配置参数说明：

  name (string)
    说明: 应用名称
    环境变量: APP_NAME
    命令行参数: --app-name
    示例: "example-value"

=== database ===
  database.host (string)
    说明: 数据库主机地址
    环境变量: APP_DATABASE_HOST
    命令行参数: --app-database-host
    示例: "example-value"

  database.timeout (time.Duration)
    说明: 连接超时时间
    环境变量: APP_DATABASE_TIMEOUT
    命令行参数: --app-database-timeout
    示例: "30s", "5m", "1h"

类型说明：
  - string: 字符串类型
  - time.Duration: 时间间隔 (如: 30s, 5m, 1h)
  - []type: 数组类型，使用索引访问
  - map[string]type: 映射类型，使用键名访问

配置优先级 (从低到高):
  1. 配置文件
  2. 环境变量  
  3. 命令行参数
```

## 实际应用场景

1. **命令行工具**：在 `--help` 选项中显示配置帮助
2. **Web 服务**：在管理接口中展示配置文档
3. **配置验证**：生成配置模板和验证规则
4. **文档生成**：自动化配置文档生成