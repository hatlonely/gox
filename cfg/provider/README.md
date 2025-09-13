# Provider

配置数据提供者，支持文件存储、数据库存储、环境变量和命令行参数。

## 支持的提供者

- **FileProvider**: 本地文件存储，支持文件监听
- **GormProvider**: 数据库存储，支持 SQLite/MySQL
- **EnvProvider**: 环境变量和 .env 文件
- **CmdProvider**: 命令行参数

## 使用方法

### 文件存储

```go
provider, _ := NewFileProviderWithOptions(&FileProviderOptions{
    FilePath: "/path/to/config.json",
})
defer provider.Close()

// 读取配置
data, _ := provider.Load()

// 保存配置
provider.Save([]byte(`{"key": "value"}`))

// 注册变更回调
provider.OnChange(func(data []byte) error {
    // 处理配置变更
    return nil
})

// 启动监听
provider.Watch()
```

### 环境变量存储

```go
provider, _ := NewEnvProviderWithOptions(&EnvProviderOptions{
    EnvFiles: []string{".env.local", ".env"},  // 优先级：后面覆盖前面
})

// 读取配置（合并系统环境变量和 .env 文件）
data, _ := provider.Load()

// 注意：EnvProvider 不支持 Save，Watch 静默处理
```

### 命令行参数

```go
// 无前缀过滤
provider, _ := NewCmdProviderWithOptions(nil)

// 带前缀过滤
provider, _ := NewCmdProviderWithOptions(&CmdProviderOptions{
    Prefix: "app-",  // 只处理 --app-* 参数，并移除前缀
})

// 读取配置（来自 os.Args[1:]）  
data, _ := provider.Load()

// 注意：CmdProvider 不支持 Save，Watch 静默处理
```

### 数据库存储

**SQLite**
```go
provider, _ := NewGormProviderWithOptions(&GormProviderOptions{
    ConfigID: "app_config",
    Driver:   "sqlite",
    DSN:      "config.db",
})
defer provider.Close()
```

**MySQL**
```go
provider, _ := NewGormProviderWithOptions(&GormProviderOptions{
    ConfigID: "app_config", 
    Driver:   "mysql",
    DSN:      "user:pass@tcp(host:port)/dbname",
})
defer provider.Close()
```

基本操作相同：
```go
// 读取配置
data, _ := provider.Load()

// 保存配置  
provider.Save([]byte(`{"key": "value"}`))

// 注册变更回调
provider.OnChange(func(data []byte) error {
    // 处理配置变更
    return nil
})

// 启动监听
provider.Watch()
```

## 监听机制

- **OnChange**: 注册变更回调函数，不启动监听
- **Watch**: 真正启动监听，只有调用后 OnChange 回调才会被触发
- **延迟初始化**: 监听器在第一次调用 Watch 时才初始化
- **线程安全**: 多次调用 Watch 是安全的

## 配置优先级

当使用多个 Provider 时，建议的优先级顺序：

```
系统环境变量 < .env 文件 < 命令行参数 < 配置文件
```

## 特性

- **统一接口**: 所有 Provider 都实现相同的接口
- **灵活配置**: 支持多种数据源和配置方式  
- **优先级管理**: 可组合多个 Provider 实现配置覆盖
- **实时监听**: FileProvider 和 GormProvider 支持配置变更监听
- **容错处理**: 文件不存在等错误不会影响其他数据源
- **延迟初始化**: 监听器在第一次调用 Watch 时才初始化