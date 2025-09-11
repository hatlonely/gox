# Storage 存储实现

配置管理的存储实现，支持层次化访问和类型转换。

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

- **层次化访问**：使用点号表示法 (`database.host`) 和数组索引 (`servers[0]`) 访问嵌套结构
- **智能字段匹配**：自动将结构体字段匹配到不同命名约定的扁平化键名
- **类型转换**：支持基本类型、time.Duration、time.Time 和自定义结构体
- **标签支持**：按优先级支持 `cfg`、`json`、`yaml`、`toml` 和 `ini` 标签
- **灵活分隔符**：自定义键分隔符和数组格式

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