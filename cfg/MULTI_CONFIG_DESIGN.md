# MultiConfig 设计文档

## 概述

MultiConfig 是对现有 SingleConfig 的扩展，支持从多个配置源获取配置数据，并按优先级合并。任何一个配置源发生变更时，都会触发统一的变更回调。

## 设计目标

1. **多源支持**：支持任意多个配置源的组合（文件、环境变量、数据库等）
2. **优先级合并**：按配置源顺序进行优先级合并，后面的配置覆盖前面的配置
3. **统一监听**：任何配置源变更都触发统一的变更回调
4. **接口兼容**：完全实现 Config 接口，对使用者透明
5. **简洁设计**：保持简单的架构，避免过度设计

## 架构设计

### 核心组件

#### 1. ConfigSource（简化版）

```go
// ConfigSource 配置源，包含 Provider、Decoder 和当前存储的数据
type ConfigSource struct {
    provider provider.Provider // 配置数据提供者
    decoder  decoder.Decoder   // 配置数据解码器  
    storage  storage.Storage   // 当前配置源的数据
}

// ConfigSourceOptions 配置源选项，用于创建配置源
type ConfigSourceOptions struct {
    Provider refx.TypeOptions `cfg:"provider"`
    Decoder  refx.TypeOptions `cfg:"decoder"`
}
```

#### 2. MultiStorage 接口（极简版）

```go
// MultiStorage 多配置存储接口
type MultiStorage interface {
    Storage // 继承基本的 Storage 接口
    
    // UpdateStorage 更新指定索引的存储源，返回是否有变更
    UpdateStorage(index int, storage Storage) bool
}

// NewMultiStorage 创建多配置存储
func NewMultiStorage(sources []Storage) MultiStorage
```

**设计说明**：
- 创建时传入 Storage 切片，确定配置源数量和初始数据
- 只提供 `UpdateStorage` 方法按索引更新存储源
- 移除所有其他复杂接口，保持极简设计

#### 3. MultiConfig 结构（简化版）

```go
// MultiConfig 多配置管理器
type MultiConfig struct {
    // 配置源数组，索引越大优先级越高（后面的覆盖前面的）
    sources []ConfigSource
    
    // 多配置存储
    multiStorage MultiStorage
    
    // 通用配置
    logger           log.Logger
    handlerExecution *HandlerExecutionOptions
    
    // 变更监听相关
    onKeyChangeHandlers map[string][]func(storage.Storage) error
    
    // 子配置支持
    parent *MultiConfig
    prefix string
    
    // 关闭控制
    closeMu     sync.Mutex
    closed      bool
    closeResult error
}
```

## 合并策略（唯一且简单）

MultiConfig 只有一种合并策略，封装在 MultiStorage 中：

### ConvertTo 合并逻辑

```go
func (ms *multiStorage) ConvertTo(object any) error {
    // 依次调用每个 Storage 的 ConvertTo，后面的覆盖前面的
    for i, storage := range ms.sources {
        if storage != nil {
            if err := storage.ConvertTo(object); err != nil {
                return fmt.Errorf("failed to convert from source %d: %w", i, err)
            }
        }
    }
    return nil
}
```

**合并规则**：
1. 按数组顺序依次调用每个 Storage 的 ConvertTo
2. 后面的配置会覆盖前面的配置（Go 结构体字段赋值特性）
3. 没有复杂的深度合并逻辑，利用 ConvertTo 的天然覆盖特性

## 优先级规则

使用**数组索引**作为优先级：

- `sources[0]`：基础配置（优先级最低）
- `sources[1]`：环境特定配置
- `sources[2]`：用户自定义配置（优先级最高）

## 变更监听机制

### 监听流程

1. **源监听**：每个 ConfigSource 的 Provider 监听自己的数据变更
2. **数据更新**：变更发生时，重新解码数据并更新对应的 Storage
3. **变更检测**：MultiStorage 检测更新前后是否有实际变更
4. **回调触发**：如果有变更，触发相应的变更回调

### 变更处理（简化版）

```go
func (c *MultiConfig) handleSourceChange(sourceIndex int, newData []byte) error {
    source := &c.sources[sourceIndex]
    
    // 重新解码数据
    newStorage, err := source.decoder.Decode(newData)
    if err != nil {
        return fmt.Errorf("failed to decode data from source %d: %w", sourceIndex, err)
    }
    
    // 更新存储源，检测是否有变更
    source.storage = newStorage
    changed := c.multiStorage.UpdateStorage(sourceIndex, newStorage)
    
    if changed {
        // 触发变更监听器
        c.triggerChangeHandlers()
    }
    
    return nil
}
```

## 使用示例

### 基本用法

```go
// 创建多配置源
multiConfig, err := cfg.NewMultiConfigWithSources([]*cfg.ConfigSourceOptions{
    {
        // 基础配置文件（优先级最低）
        Provider: refx.TypeOptions{
            Type: "FileProvider",
            Options: &provider.FileProviderOptions{
                FilePath: "config.yaml",
            },
        },
        Decoder: refx.TypeOptions{Type: "YamlDecoder"},
    },
    {
        // 环境变量配置（中等优先级）
        Provider: refx.TypeOptions{Type: "EnvProvider"},
        Decoder:  refx.TypeOptions{Type: "EnvDecoder"},
    },
    {
        // 数据库配置（优先级最高）
        Provider: refx.TypeOptions{
            Type: "GormProvider",
            Options: &provider.GormProviderOptions{
                DSN:   "postgres://...",
                Table: "app_configs",
            },
        },
        Decoder: refx.TypeOptions{Type: "JsonDecoder"},
    },
})

if err != nil {
    log.Fatal(err)
}
```

### 配置使用

```go
// 完全兼容 Config 接口
var appConfig AppConfig
if err := multiConfig.ConvertTo(&appConfig); err != nil {
    log.Fatal(err)
}

// 子配置使用
dbConfig := multiConfig.Sub("database")
var dbConf DatabaseConfig
dbConfig.ConvertTo(&dbConf)
```

### 监听配置变更

```go
// 监听整体配置变更
multiConfig.OnChange(func(storage storage.Storage) error {
    log.Info("配置已更新")
    
    // 重新加载配置到应用
    var appConfig AppConfig
    if err := storage.ConvertTo(&appConfig); err != nil {
        return fmt.Errorf("failed to convert config: %w", err)
    }
    
    // 应用新配置
    return applyNewConfig(&appConfig)
})

// 启动配置监听
if err := multiConfig.Watch(); err != nil {
    log.Fatal(err)
}
```

## 实现要点

### 1. MultiStorage 实现

```go
type multiStorage struct {
    sources []storage.Storage // 配置源存储数组
    mu      sync.RWMutex     // 并发保护
}

func NewMultiStorage(sources []storage.Storage) MultiStorage {
    // 复制切片，避免外部修改
    sourcesCopy := make([]storage.Storage, len(sources))
    copy(sourcesCopy, sources)
    
    return &multiStorage{
        sources: sourcesCopy,
    }
}

func (ms *multiStorage) UpdateStorage(index int, storage storage.Storage) bool {
    ms.mu.Lock()
    defer ms.mu.Unlock()
    
    if index < 0 || index >= len(ms.sources) {
        return false
    }
    
    // 检测是否有变更
    old := ms.sources[index]
    if old != nil && old.Equals(storage) {
        return false // 没有变更
    }
    
    ms.sources[index] = storage
    return true // 有变更
}

func (ms *multiStorage) ConvertTo(object any) error {
    ms.mu.RLock()
    defer ms.mu.RUnlock()
    
    // 依次调用每个 Storage 的 ConvertTo
    for i, storage := range ms.sources {
        if storage != nil {
            if err := storage.ConvertTo(object); err != nil {
                return fmt.Errorf("failed to convert from source %d: %w", i, err)
            }
        }
    }
    return nil
}

func (ms *multiStorage) Sub(key string) storage.Storage {
    ms.mu.RLock()
    defer ms.mu.RUnlock()
    
    // 创建子存储的 MultiStorage
    subSources := make([]storage.Storage, len(ms.sources))
    for i, storage := range ms.sources {
        if storage != nil {
            subSources[i] = storage.Sub(key)
        }
    }
    
    return NewMultiStorage(subSources)
}

func (ms *multiStorage) Equals(other storage.Storage) bool {
    // 实现比较逻辑
}
```

### 2. 变更检测优化

由于使用 ConvertTo 的天然覆盖特性，变更检测非常简单：
- 只需要检测单个 Storage 源是否变更
- 无需复杂的深度比较和合并逻辑
- 利用现有 Storage.Equals 方法进行比较

### 3. 并发安全

- 使用读写锁保护 MultiStorage 的并发访问
- 配置源数组在创建后不可变，只能更新其中的 Storage
- 所有变更操作都是原子的

## 兼容性

MultiConfig 完全实现 Config 接口，可以无缝替换 SingleConfig：

```go
// 原来的代码
var config cfg.Config = singleConfig

// 可以直接替换为
var config cfg.Config = multiConfig

// 所有方法调用保持不变
config.Sub("database").ConvertTo(&dbConfig)
config.OnChange(handleConfigChange)
config.Watch()
```

## 总结

简化后的 MultiConfig 设计具有以下特点：

1. **极简接口**：MultiStorage 只提供必要的 UpdateStorage 方法
2. **简单合并**：利用 ConvertTo 的天然覆盖特性，无需复杂合并逻辑
3. **固定策略**：只有一种合并策略，封装在实现中
4. **高效实现**：避免了复杂的合并算法和策略选择
5. **完全兼容**：与现有 Config 接口完全兼容

这种设计既满足了多配置源的需求，又保持了代码的简洁性和可维护性。