# ref - 反射构造器

一个基于反射的 Go 构造器注册和创建系统，支持动态对象创建和依赖注入。

## 功能特性

- **多种构造函数签名支持**：支持 0 个或 1 个参数，返回对象或对象+错误
- **泛型支持**：提供类型安全的泛型 API
- **重复注册检查**：相同函数跳过，不同函数报错
- **Must 方法**：适用于初始化阶段，注册失败直接 panic
- **线程安全**：使用 `sync.Map` 保证并发安全

## 安装

```bash
go get github.com/hatlonely/gox/ref
```

## 支持的构造函数类型

ref 支持以下几种构造函数签名：

```go
// 1. 接收 options 参数，返回对象和错误
func NewValue(options *Options) (*Value, error)

// 2. 不接收参数，只返回对象
func NewDefaultValue() *Value

// 3. 接收 options 参数，只返回对象
func NewSimpleValue(options *Options) *Value

// 4. 不接收参数，返回对象和错误
func NewErrorValue() (*Value, error)
```

## API 参考

### 注册方法

#### `Register(namespace, type_, newFunc)`
基础注册方法，手动指定 namespace 和 type。

```go
err := ref.Register("myapp", "Database", NewDatabase)
if err != nil {
    log.Fatal(err)
}
```

#### `RegisterT[T](newFunc)`
泛型注册方法，自动从类型 T 推导 namespace 和 type。

```go
err := ref.RegisterT[*Database](NewDatabase)
if err != nil {
    log.Fatal(err)
}
```

#### `MustRegister(namespace, type_, newFunc)`
Must 版本的注册方法，失败时 panic，适用于初始化阶段。

```go
func init() {
    ref.MustRegister("myapp", "Database", NewDatabase)
}
```

#### `MustRegisterT[T](newFunc)`
Must 版本的泛型注册方法。

```go
func init() {
    ref.MustRegisterT[*Database](NewDatabase)
}
```

### 创建方法

#### `New(namespace, type_, options)`
基础创建方法，根据 namespace 和 type 创建对象。

```go
obj, err := ref.New("myapp", "Database", &DatabaseOptions{
    Host: "localhost",
    Port: 5432,
})
if err != nil {
    log.Fatal(err)
}
db := obj.(*Database)
```

#### `NewT[T](options)`
泛型创建方法，类型安全地创建对象。

```go
db, err := ref.NewT[*Database](&DatabaseOptions{
    Host: "localhost",
    Port: 5432,
})
if err != nil {
    log.Fatal(err)
}
// db 已经是 *Database 类型，无需类型转换
```

## 使用示例

### 基本使用

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/hatlonely/gox/ref"
)

type Database struct {
    Host string
    Port int
}

type DatabaseOptions struct {
    Host string
    Port int
}

// 构造函数
func NewDatabase(options *DatabaseOptions) (*Database, error) {
    if options.Host == "" {
        return nil, fmt.Errorf("host cannot be empty")
    }
    return &Database{
        Host: options.Host,
        Port: options.Port,
    }, nil
}

func main() {
    // 注册构造函数
    err := ref.Register("myapp", "Database", NewDatabase)
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建对象
    obj, err := ref.New("myapp", "Database", &DatabaseOptions{
        Host: "localhost",
        Port: 5432,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    db := obj.(*Database)
    fmt.Printf("Database: %+v\n", db)
}
```

### 使用泛型 API

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/hatlonely/gox/ref"
)

func main() {
    // 使用泛型注册
    err := ref.RegisterT[*Database](NewDatabase)
    if err != nil {
        log.Fatal(err)
    }
    
    // 使用泛型创建，类型安全
    db, err := ref.NewT[*Database](&DatabaseOptions{
        Host: "localhost",
        Port: 5432,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // db 已经是正确类型，无需类型转换
    fmt.Printf("Database: %+v\n", db)
}
```

### 初始化阶段注册

```go
package main

import (
    "github.com/hatlonely/gox/ref"
)

func init() {
    // 在初始化阶段使用 Must 方法注册
    // 如果注册失败，程序会 panic
    ref.MustRegisterT[*Database](NewDatabase)
    ref.MustRegisterT[*Redis](NewRedis)
    ref.MustRegisterT[*Logger](NewLogger)
}

func main() {
    // 直接使用，无需担心注册失败
    db, _ := ref.NewT[*Database](&DatabaseOptions{
        Host: "localhost",
        Port: 5432,
    })
    
    redis, _ := ref.NewT[*Redis](&RedisOptions{
        Addr: "localhost:6379",
    })
    
    logger, _ := ref.NewT[*Logger](nil)
}
```

### 不同构造函数类型示例

```go
// 1. 有参数，返回对象+错误
func NewDatabase(options *DatabaseOptions) (*Database, error) {
    // ...
}

// 2. 无参数，只返回对象
func NewLogger() *Logger {
    return &Logger{Level: "info"}
}

// 3. 有参数，只返回对象
func NewCache(options *CacheOptions) *Cache {
    return &Cache{Size: options.Size}
}

// 4. 无参数，返回对象+错误
func NewMonitor() (*Monitor, error) {
    return &Monitor{}, nil
}

func init() {
    ref.MustRegisterT[*Database](NewDatabase)
    ref.MustRegisterT[*Logger](NewLogger)
    ref.MustRegisterT[*Cache](NewCache)
    ref.MustRegisterT[*Monitor](NewMonitor)
}
```

## 重复注册处理

ref 智能处理重复注册：

```go
// 第一次注册
ref.MustRegister("myapp", "Database", NewDatabase)

// 相同函数重复注册 - 成功（跳过）
ref.MustRegister("myapp", "Database", NewDatabase) // OK

// 不同函数重复注册 - 会 panic
ref.MustRegister("myapp", "Database", NewAnotherDatabase) // PANIC!
```

## 错误处理

### 注册错误

- 构造函数不是函数类型
- 构造函数参数数量错误（必须是 0 个或 1 个）
- 构造函数返回值数量错误（必须是 1 个或 2 个）
- 第二个返回值不是 error 类型
- 重复注册不同的构造函数

### 创建错误

- 构造函数未注册
- 构造函数需要参数但传入了 nil
- 构造函数执行时返回错误

## 最佳实践

1. **在 `init()` 函数中使用 `MustRegister`**：
   ```go
   func init() {
       ref.MustRegisterT[*Database](NewDatabase)
   }
   ```

2. **优先使用泛型 API**：
   ```go
   // 推荐
   db, err := ref.NewT[*Database](options)
   
   // 不推荐
   obj, err := ref.New("pkg", "Database", options)
   db := obj.(*Database)
   ```

3. **构造函数命名约定**：
   ```go
   // 推荐：New + 类型名
   func NewDatabase(options *DatabaseOptions) (*Database, error)
   func NewRedis(options *RedisOptions) (*Redis, error)
   ```

4. **错误处理**：
   ```go
   // 在构造函数中进行参数验证
   func NewDatabase(options *DatabaseOptions) (*Database, error) {
       if options.Host == "" {
           return nil, fmt.Errorf("host cannot be empty")
       }
       return &Database{Host: options.Host}, nil
   }
   ```

## 使用场景

### 依赖注入

```go
type Service struct {
    db    *Database
    cache *Cache
    logger *Logger
}

func NewService(options *ServiceOptions) (*Service, error) {
    db, err := ref.NewT[*Database](options.Database)
    if err != nil {
        return nil, err
    }
    
    cache, err := ref.NewT[*Cache](options.Cache)
    if err != nil {
        return nil, err
    }
    
    logger, err := ref.NewT[*Logger](options.Logger)
    if err != nil {
        return nil, err
    }
    
    return &Service{
        db:     db,
        cache:  cache,
        logger: logger,
    }, nil
}
```

### 配置驱动的对象创建

```go
type Config struct {
    Database DatabaseConfig `json:"database"`
    Redis    RedisConfig    `json:"redis"`
    Logger   LoggerConfig   `json:"logger"`
}

func CreateFromConfig(config *Config) error {
    // 根据配置动态创建对象
    db, err := ref.NewT[*Database](&config.Database)
    if err != nil {
        return fmt.Errorf("create database: %w", err)
    }
    
    redis, err := ref.NewT[*Redis](&config.Redis)
    if err != nil {
        return fmt.Errorf("create redis: %w", err)
    }
    
    logger, err := ref.NewT[*Logger](&config.Logger)
    if err != nil {
        return fmt.Errorf("create logger: %w", err)
    }
    
    // 使用创建的对象...
    return nil
}
```

## 线程安全

ref 使用 `sync.Map` 保证注册和创建操作的并发安全，可以在多 goroutine 环境中安全使用。

```go
// 可以在多个 goroutine 中安全调用
go func() {
    obj, _ := ref.NewT[*Database](options)
    // 使用 obj...
}()

go func() {
    obj, _ := ref.NewT[*Cache](options)
    // 使用 obj...
}()
```

## 许可证

MIT License