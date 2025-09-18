# Storage

配置数据存储模块，提供统一的配置访问接口和多种存储实现。

## 核心接口

```go
type Storage interface {
    Sub(key string) Storage              // 获取子配置存储对象
    ConvertTo(object interface{}) error  // 将配置数据转换为结构体
    Equals(other Storage) bool           // 比较两个存储是否相同
}
```

## 存储实现

### MapStorage

基于 map 和 slice 的层级化存储，适用于嵌套配置数据。

```go
// 准备层级化数据
data := map[string]interface{}{
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
    "servers": []interface{}{"web1", "web2"},
    "config": map[string]interface{}{
        "timeout":    "30s",
        "created_at": "2023-12-25T15:30:45Z",
        "enabled":    true,
    },
}

// 创建存储
storage := NewMapStorage(data)

// 配置选项
storage.WithDefaults(false)  // 禁用默认值

// 获取子配置
dbConfig := storage.Sub("database")
dbHost := storage.Sub("database.host")

// 转换为结构体
var config Config
storage.ConvertTo(&config)
```

支持的 key 格式：
- `"database.host"` - 多级嵌套访问
- `"servers[0].port"` - 数组索引访问

### FlatStorage

扁平化存储，所有配置项存储在单层 map 中，使用分隔符表示层级。

```go
data := map[string]interface{}{
    "database.host":     "localhost",
    "database.port":     3306,
    "servers.0.host":    "server1",
    "servers.1.host":    "server2",
}

storage := NewFlatStorage(data)
storage.WithSeparator("-")     // 自定义分隔符，默认为 "."
storage.WithUppercase(true)    // 键名转大写
storage.WithLowercase(true)    // 键名转小写
```

### MultiStorage

多配置源存储，支持按优先级合并多个配置源。

```go
sources := []Storage{
    NewMapStorage(defaultConfig),  // 低优先级
    NewMapStorage(userConfig),     // 高优先级
}

// 创建多源存储，索引越大优先级越高
multiStorage := NewMultiStorage(sources)

// 动态更新配置源
changed := multiStorage.UpdateStorage(1, newUserConfig)
```

### ValidateStorage

配置验证存储，在配置转换后自动进行结构体验证。

```go
storage := NewValidateStorage(baseStorage)

// 转换时会自动验证结构体字段
var config Config
err := storage.ConvertTo(&config) // 验证失败时返回错误
```

## 主要特性

### 类型转换

自动支持多种类型转换：
- 基本类型：字符串、数字、布尔值
- 时间类型：`time.Time`、`time.Duration`
- 集合类型：map、slice、array
- 结构体字段映射

### 标签支持

结构体字段映射支持多种标签，优先级从高到低：

```go
type Config struct {
    Host string `cfg:"host" json:"hostname"`
    Port int    `yaml:"port"`
}
```

支持标签：`cfg` > `json` > `yaml` > `toml` > `ini` > 字段名

### 默认值

支持自动设置默认值（需配合 `def` 包）：

```go
type Config struct {
    Host string `def:"localhost"`
    Port int    `def:"8080"`
}

storage := NewMapStorage(data)
storage.ConvertTo(&config) // 自动设置未配置字段的默认值
```

### 智能指针处理

- 配置不存在时：保持指针原状态（nil 保持 nil）
- 配置存在时：自动创建实例并赋值

## 使用示例

```go
// 准备配置数据
data := map[string]interface{}{
    "database": map[string]interface{}{
        "host": "localhost",
        "port": 3306,
    },
    "servers": []interface{}{"web1", "web2"},
}

// 创建存储
storage := NewMapStorage(data)

// 定义配置结构
type Config struct {
    Database struct {
        Host string `json:"host"`
        Port int    `json:"port"`
    } `json:"database"`
    Servers []string `json:"servers"`
}

// 转换配置
var config Config
err := storage.ConvertTo(&config)
if err != nil {
    log.Fatal(err)
}

// 使用配置
fmt.Printf("Database: %s:%d\n", config.Database.Host, config.Database.Port)
fmt.Printf("Servers: %v\n", config.Servers)
```

## 最佳实践

1. **选择合适的存储类型**：
   - 层级数据使用 `MapStorage`
   - 扁平数据使用 `FlatStorage`
   - 多源配置使用 `MultiStorage`
   - 需要验证使用 `ValidateStorage`

2. **合理使用默认值**：
   - 生产环境建议启用默认值功能
   - 测试环境可禁用以便发现配置缺失

3. **优化性能**：
   - 频繁访问的配置可缓存转换结果
   - 大型配置建议分模块使用 `Sub()` 方法