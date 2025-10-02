# UID 生成器

轻量级的唯一ID生成器包，支持整数和字符串两种类型的ID生成。

## 快速开始

### 整数ID生成（Snowflake算法）

```go
package main

import (
    "fmt"
    "github.com/hatlonely/gox/uid"
)

func main() {
    // 创建默认的整数生成器
    generator := uid.NewIntGenerator()
    
    // 生成ID
    id := generator.Generate()
    fmt.Println(id) // 输出：1793943234567890123
}
```

### 字符串ID生成（UUID v7）

```go
package main

import (
    "fmt"
    "github.com/hatlonely/gox/uid"
)

func main() {
    // 创建默认的字符串生成器
    generator := uid.NewStrGenerator()
    
    // 生成UUID
    uuid := generator.Generate()
    fmt.Println(uuid) // 输出：01929394-5678-7abc-def0-123456789abc
}
```

## 高级用法

### 自定义配置

```go
// 自定义整数生成器
intOptions := &ref.TypeOptions{
    Type: "SnowflakeGenerator",
    Options: map[string]interface{}{
        "MachineID": 123,
    },
}
intGen, _ := uid.NewIntGeneratorWithOptions(intOptions)

// 自定义字符串生成器
strOptions := &ref.TypeOptions{
    Type: "UUIDGenerator", 
    Options: map[string]interface{}{
        "Version": "v4",
    },
}
strGen, _ := uid.NewStrGeneratorWithOptions(strOptions)
```

## API 参考

### 简化API

- `NewIntGenerator()` - 创建默认整数生成器（Snowflake）
- `NewStrGenerator()` - 创建默认字符串生成器（UUID v7）

### 完整API

- `NewIntGeneratorWithOptions(options)` - 创建自定义整数生成器
- `NewStrGeneratorWithOptions(options)` - 创建自定义字符串生成器

## 支持的生成器

### 整数生成器
- **Snowflake** - 64位分布式唯一ID（默认）
- **TimestampSeq** - 时间戳+序列号
- **Redis** - 基于Redis的计数器

### 字符串生成器
- **UUID v7** - 时间排序的UUID（默认）
- **UUID v4** - 随机UUID
- **UUID v6** - 重排序的UUID v1
- **UUID v1** - 基于时间和MAC地址

## 性能

```
BenchmarkNewIntGenerator-10     4930040      244.0 ns/op
BenchmarkNewStrGenerator-10     4654872      261.2 ns/op
```

## 特性

- 🚀 高性能：纳秒级生成速度
- 🔒 线程安全：支持并发调用
- 📦 轻量级：最小依赖
- 🎯 易用性：提供简化API
- 🔧 可配置：支持自定义参数