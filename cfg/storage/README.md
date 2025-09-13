# Storage 存储实现

配置管理的存储实现，支持层次化访问、类型转换和智能 nil 处理。

## 核心接口

```go
type Storage interface {
    Sub(key string) Storage
    ConvertTo(object interface{}) error
    Equals(other Storage) bool
}
```

## 新特性

### 智能 Nil 处理

#### Nil Storage 安全返回
当访问不存在的配置项时，`Sub` 方法返回类型化的 nil Storage，支持安全的链式调用：

```go
// 即使路径不存在也不会 panic
storage := NewMapStorage(data)
nilStorage := storage.Sub("nonexistent.deeply.nested.key")
err := nilStorage.ConvertTo(&config) // 不会修改 config，返回 nil
```

#### 智能指针字段处理
`ConvertTo` 方法对结构体中的指针字段进行智能处理：

**处理规则**：
- **配置中没有该字段**：保持指针字段的原始状态（nil 保持 nil，非 nil 保持不变）
- **配置中存在该字段**：即使指针字段为 nil，也会创建新实例并赋值

```go
type Config struct {
    Name     string
    Database *DatabaseConfig
    Cache    *CacheConfig
}

// 场景1：配置中没有对应字段
storage := NewMapStorage(map[string]interface{}{
    "name": "test",
    // 注意：没有 "database" 和 "cache" 字段
})

config := &Config{
    Name:     "original",
    Database: nil,                           // nil 指针
    Cache:    &CacheConfig{TTL: 300},       // 非 nil 指针
}

storage.ConvertTo(config)
// 结果：
// config.Name = "test"      (被覆盖)
// config.Database = nil     (保持 nil - 重要特性!)
// config.Cache.TTL = 300    (保持原值 - 重要特性!)
```

#### Nil Storage 比较规则
```go
var nilStorage1 *MapStorage = nil
var nilStorage2 *MapStorage = nil
var nilInterface Storage = nil

nilStorage1.Equals(nilStorage2) // true - 两个 nil Storage 相等
nilStorage1.Equals(nilInterface) // false - nil Storage != nil interface
```

## 实现方式

### MapStorage 映射存储

基于嵌套 map 和 slice 的层次化配置存储。

```go
data := map[string]interface{}{
    "database": map[string]interface{}{
        "host": "localhost",
        "port": 5432,
    },
    "servers": []string{"server1", "server2"},
}

storage := NewMapStorage(data)

// 访问嵌套值
hostStorage := storage.Sub("database.host")
var host string
hostStorage.ConvertTo(&host) // "localhost"

// 数组访问
serverStorage := storage.Sub("servers[0]")
var server string
serverStorage.ConvertTo(&server) // "server1"
```

### FlatStorage 打平存储

支持智能字段路径匹配的扁平化键值存储，非常适合环境变量和 .env 文件。

```go
// 环境变量风格
data := map[string]interface{}{
    "DATABASE_HOST":     "localhost",
    "DATABASE_PORT":     5432,
    "APP_NAME":          "my-service",
    "CACHE_REDIS_HOST":  "redis.example.com",
}

storage := NewFlatStorageWithOptions(data, "_", "_%d")

type Config struct {
    Database struct {
        Host string `cfg:"host"`
        Port int    `cfg:"port"`
    } `cfg:"database"`
    Name string `cfg:"name"`
    Cache struct {
        Redis struct {
            Host string `cfg:"host"`
        } `cfg:"redis"`
    } `cfg:"cache"`
}

var config Config
storage.ConvertTo(&config)
// 自动匹配：
// database.host -> DATABASE_HOST
// database.port -> DATABASE_PORT  
// name -> APP_NAME
// cache.redis.host -> CACHE_REDIS_HOST
```

#### 复杂嵌套结构支持

```go
// 同时支持 struct、map、slice 混合嵌套
data := map[string]interface{}{
    "APP_NAME": "service",
    "DATABASE_POOLS_0_HOST": "db1.com",
    "DATABASE_POOLS_1_HOST": "db2.com", 
    "CACHE_REDIS_URL": "redis://localhost",
    "FEATURES_AUTH_ENABLED": true,
}

type Config struct {
    Name string `cfg:"name"`
    Database struct {
        Pools []struct {
            Host string `cfg:"host"`
        } `cfg:"pools"`
    } `cfg:"database"`
    Cache    map[string]string `cfg:"cache"`    // redis_url: redis://localhost
    Features map[string]bool   `cfg:"features"` // auth_enabled: true
}
```

## 功能特性

- **智能 Nil 处理**：类型化 nil Storage 返回，支持安全链式调用和智能指针字段处理
- **层次化访问**：使用点号表示法 (`database.host`) 和数组索引 (`servers[0]`) 访问嵌套结构
- **智能字段匹配**：自动将结构体字段匹配到不同命名约定的扁平化键名
- **类型转换**：支持基本类型、time.Duration、time.Time 和自定义结构体
- **标签支持**：按优先级支持 `cfg`、`json`、`yaml`、`toml` 和 `ini` 标签
- **灵活分隔符**：自定义键分隔符和数组格式

## 时间类型转换

### Duration 转换
```go
type Config struct {
    Timeout time.Duration `cfg:"timeout"`
}

// 支持的格式：
// - 字符串: "30s", "5m", "1h"
// - 整数: 纳秒数
// - 浮点数: 秒数
```

### Time 转换
```go
type Config struct {
    CreatedAt time.Time `cfg:"created_at"`
}

// 支持的格式：
// - RFC3339: "2023-01-01T12:00:00Z"
// - 日期: "2023-01-01"
// - Unix 时间戳: 1672574400
```

## 智能匹配规则 (FlatStorage)

1. **精确匹配**：`name` → `name`
2. **大小写转换**：`name` → `NAME`、`Name`
3. **分隔符转换**：`database.host` ↔ `database_host`
4. **前缀匹配**：`name` → `APP_NAME`、`app_name`
5. **组合模式**：以上所有规则的组合

特别适用于：
- 环境变量 (`DATABASE_HOST`、`REDIS_URL`)
- .env 文件 (`APP_NAME=my-service`)
- 命名约定不一致的配置系统

## 使用示例

### 基本用法
```go
// 创建存储
storage := NewMapStorage(map[string]interface{}{
    "app": map[string]interface{}{
        "name":    "myapp",
        "version": "1.0.0",
        "database": map[string]interface{}{
            "host": "localhost",
            "port": 3306,
        },
    },
})

// 获取子配置
appStorage := storage.Sub("app")
dbStorage := appStorage.Sub("database")

// 转换为结构体
type DatabaseConfig struct {
    Host string `cfg:"host"`
    Port int    `cfg:"port"`
}

var dbConfig DatabaseConfig
err := dbStorage.ConvertTo(&dbConfig)
```

### 安全的链式访问
```go
// 安全的链式访问，即使中间路径不存在也不会 panic
var config DatabaseConfig
err := storage.Sub("app.database").ConvertTo(&config)

// nil Storage 处理
nilStorage := storage.Sub("nonexistent")
err = nilStorage.ConvertTo(&config) // err == nil, config 不变
```

### 智能指针字段处理
```go
type Config struct {
    Required string
    Optional *OptionalConfig
}

// 即使配置中没有 "optional" 字段，Optional 指针也会保持原状
config := &Config{Optional: nil}
storage.Sub("incomplete").ConvertTo(config) // Optional 仍为 nil

// 如果配置中有 "optional" 字段，即使 Optional 为 nil 也会创建实例
storage.Sub("complete").ConvertTo(config) // Optional 被创建并赋值
```

## 测试

运行测试：
```bash
go test ./cfg/storage/...
```

## 注意事项

1. **Nil Safety**: 所有方法都支持 nil receiver 调用，确保链式操作的安全性
2. **指针字段**: 指针字段的处理遵循智能规则，保持向后兼容的同时提供灵活的配置绑定
3. **类型转换**: 支持丰富的类型转换，包括时间类型的智能解析
4. **标签优先级**: 配置标签按优先级匹配，`cfg` 标签具有最高优先级
5. **向后兼容**: 所有新特性都保持与现有代码的向后兼容性