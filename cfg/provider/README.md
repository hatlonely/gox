# Provider

Provider 包实现了配置数据提供者接口，支持多种配置源和存储后端。

## 接口定义

```go
type Provider interface {
    Load() (data []byte, err error)
    Save(data []byte) error
    OnChange(fn func(data []byte) error)
    Close() error
}
```

## 支持的 Provider

### FileProvider

从本地文件系统读取和写入配置数据，支持文件变更监听。

```go
provider, err := NewFileProviderWithOptions(&FileProviderOptions{
    FilePath: "/path/to/config.json",
})
if err != nil {
    panic(err)
}
defer provider.Close()

// 读取配置
data, err := provider.Load()

// 保存配置
err = provider.Save([]byte(`{"key": "value"}`))

// 监听文件变更
provider.OnChange(func(data []byte) error {
    fmt.Println("Config changed:", string(data))
    return nil
})
```

### GormProvider

使用 GORM 将配置数据存储在数据库中，支持多种数据库后端。

```go
provider, err := NewGormProviderWithOptions(&GormProviderOptions{
    ConfigID:     "app_config",
    Driver:       "sqlite",          // 支持: sqlite, mysql
    DSN:          "config.db",
    TableName:    "config_data",     // 可选，默认 config_data
    PollInterval: 5 * time.Second,   // 轮询间隔
})
if err != nil {
    panic(err)
}
defer provider.Close()

// 读取配置
data, err := provider.Load()

// 保存配置
err = provider.Save([]byte(`{"database": {"host": "localhost"}}`))

// 监听配置变更（基于轮询）
provider.OnChange(func(data []byte) error {
    fmt.Println("Database config changed:", string(data))
    return nil
})
```

#### 支持的数据库

- **SQLite**: `Driver: "sqlite"`, `DSN: "path/to/file.db"`
- **MySQL**: `Driver: "mysql"`, `DSN: "user:pass@tcp(host:port)/dbname"`

#### 数据库表结构

```sql
CREATE TABLE config_data (
    id varchar(255) PRIMARY KEY,
    content longtext NOT NULL,
    version bigint AUTO_INCREMENT,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

## 错误处理

所有 Provider 使用统一的错误类型：

```go
type ProviderError struct {
    Msg string
    Err error
}
```

## 特性对比

| 特性 | FileProvider | GormProvider |
|------|-------------|--------------|
| 读取配置 | ✅ | ✅ |
| 保存配置 | ✅ | ✅ |
| 变更监听 | ✅ (fsnotify) | ✅ (轮询) |
| 并发安全 | ✅ | ✅ |
| 版本控制 | ❌ | ✅ |
| 分布式 | ❌ | ✅ |
| 依赖 | fsnotify | GORM + 数据库驱动 |

## 测试

```bash
# 运行所有测试
go test ./provider/

# 运行特定测试
go test ./provider/ -run TestFileProvider
go test ./provider/ -run TestGormProvider
```

### MySQL 测试

设置环境变量启用 MySQL 测试：

```bash
export MYSQL_TEST_DSN="user:pass@tcp(localhost:3306)/test_db"
go test ./provider/ -run TestGormProvider_MySQL
```