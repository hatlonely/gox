# Provider

配置数据提供者，支持本地文件和数据库存储。

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

// 监听变更
provider.OnChange(func(data []byte) error {
    // 处理配置变更
    return nil
})
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

// 监听变更
provider.OnChange(func(data []byte) error {
    // 处理配置变更
    return nil
})
```