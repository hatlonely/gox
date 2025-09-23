# KV Loader

KV 数据加载器，支持从文件加载键值对数据并监听文件变化。

## 核心接口

### Loader 接口
```go
type Loader[K, V any] interface {
    // OnChange 注册数据变更监听
    OnChange(listener Listener[K, V]) error
    // Close 关闭 Loader
    Close() error
}
```

### 监听器类型
```go
// Listener 用于监听 KV 数据变更
type Listener[K, V any] func(stream KVStream[K, V]) error

// KVStream 用于遍历 KV 数据流
type KVStream[K, V any] interface {
    Each(func(changeType parser.ChangeType, key K, val V) error) error
}
```

## 基本用法

### 创建文件加载器

```go
import "github.com/hatlonely/gox/kv/loader"

options := &loader.KVFileLoaderOptions{
    FilePath: "/path/to/data.txt",
    Parser: &ref.TypeOptions{
        Namespace: "github.com/hatlonely/gox/kv/parser",
        Type:      "LineParser[string,string]",
        Options: &parser.LineParserOptions{
            Separator: "\t",
        },
    },
}

loader, err := loader.NewKVFileLoaderWithOptions[string, string](options)
if err != nil {
    panic(err)
}
defer loader.Close()
```

### 监听数据变化

```go
listener := func(stream loader.KVStream[string, string]) error {
    return stream.Each(func(changeType parser.ChangeType, key, value string) error {
        fmt.Printf("Key: %s, Value: %s\n", key, value)
        return nil
    })
}

err := loader.OnChange(listener)
if err != nil {
    panic(err)
}
```

## 配置选项

| 参数 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `FilePath` | string | 数据文件路径 | 必填 |
| `Parser` | *ref.TypeOptions | 解析器配置 | - |
| `SkipDirtyRows` | bool | 是否跳过脏数据 | false |
| `ScannerBufferMinSize` | int | 扫描器最小缓冲区大小 | 65536 |
| `ScannerBufferMaxSize` | int | 扫描器最大缓冲区大小 | 4194304 |
| `Logger` | *ref.TypeOptions | 日志配置 | - |

## 数据格式

数据文件为文本格式，每行一个键值对，格式由配置的解析器决定。

示例文件内容（使用制表符分隔）：
```
key1	value1
key2	value2
key3	value3
```

## 特性

- **文件监听**：自动监听文件变化并触发数据重新加载
- **错误处理**：支持跳过脏数据或在遇到错误时停止处理  
- **泛型支持**：支持任意键值类型的数据加载
- **缓冲控制**：可配置扫描器缓冲区大小以优化性能