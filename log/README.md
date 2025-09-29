# Log 日志库

一个基于 Go 标准库 `slog` 的高性能日志库，支持多种输出方式和灵活的配置。

## 特性

- 🚀 基于 Go 1.21+ `slog` 标准库
- 📝 支持 Text/JSON 格式输出
- 🎯 支持控制台、文件、多输出器
- 🎨 控制台彩色输出
- 📦 文件轮转和压缩
- ⚙️ 灵活的配置管理
- 🏗️ 支持分组和字段日志

## 快速开始

### 基础使用

```go
package main

import "github.com/hatlonely/gox/log"

func main() {
    // 使用默认日志器
    log.Default().Info("Hello, World!", "user", "john")
    
    // 使用全局方法
    logger := log.GetLogger("default")
    logger.Warn("这是一条警告", "code", 404)
}
```

## 推荐使用方式

### 日志管理器配置

使用 LogManager 管理多个日志器实例（**推荐**）：

```go
import (
    "github.com/hatlonely/gox/log"
    "github.com/hatlonely/gox/log/manager"
    "github.com/hatlonely/gox/log/logger"
    "github.com/hatlonely/gox/log/writer"
    "github.com/hatlonely/gox/ref"
)

func main() {
    // 初始化日志管理器
    options := manager.Options{
        "default": &ref.TypeOptions{
            Namespace: "github.com/hatlonely/gox/log/logger",
            Type:      "SLog",
            Options: &logger.SLogOptions{
                Level:  "info",
                Format: "text",
            },
        },
        "api": &ref.TypeOptions{
            Namespace: "github.com/hatlonely/gox/log/logger",
            Type:      "SLog",
            Options: &logger.SLogOptions{
                Level:  "info",
                Format: "json",
                Output: &ref.TypeOptions{
                    Namespace: "github.com/hatlonely/gox/log/writer",
                    Type:      "FileWriter",
                    Options: &writer.FileWriterOptions{
                        Path: "./logs/api.log",
                    },
                },
            },
        },
        "db": &ref.TypeOptions{
            Namespace: "github.com/hatlonely/gox/log/logger", 
            Type:      "SLog",
            Options: &logger.SLogOptions{
                Level:  "debug",
                Format: "text",
                Output: &ref.TypeOptions{
                    Namespace: "github.com/hatlonely/gox/log/writer",
                    Type:      "MultiWriter",
                    Options: &writer.MultiWriterOptions{
                        Writers: []ref.TypeOptions{
                            {
                                Namespace: "github.com/hatlonely/gox/log/writer",
                                Type:      "ConsoleWriter",
                                Options: &writer.ConsoleWriterOptions{
                                    Color:  true,
                                    Target: "stdout",
                                },
                            },
                            {
                                Namespace: "github.com/hatlonely/gox/log/writer",
                                Type:      "FileWriter",
                                Options: &writer.FileWriterOptions{
                                    Path: "./logs/db.log",
                                },
                            },
                        },
                    },
                },
            },
        },
    }
    
    if err := log.Init(options); err != nil {
        panic(err)
    }
    
    // 使用不同的日志器
    log.GetLogger("default").Info("默认日志器")
    log.GetLogger("api").Info("API 请求", "path", "/users", "method", "GET")
    log.GetLogger("db").Debug("数据库操作", "table", "users")
    
    // 获取管理器
    mgr := log.Manager()
    apiLogger := mgr.GetLogger("api")
    apiLogger.Error("API 错误", "error", "timeout")
}
```

### 通过配置获取日志器

使用 `NewLoggerWithOptions` 通过日志器名称获取（**推荐**）：

```go
func main() {
    // 先初始化日志管理器（同上）
    if err := log.Init(options); err != nil {
        panic(err)
    }
    
    // 通过名称获取日志器
    apiLogger, err := log.NewLoggerWithOptions(&ref.TypeOptions{
        Namespace: "github.com/hatlonely/gox/log",
        Type:      "GetLogger", 
        Options:   "api",  // 直接设置为日志器名称
    })
    if err != nil {
        panic(err)
    }
    
    dbLogger, err := log.NewLoggerWithOptions(&ref.TypeOptions{
        Namespace: "github.com/hatlonely/gox/log", 
        Type:      "GetLogger",
        Options:   "db",   // 直接设置为日志器名称
    })
    if err != nil {
        panic(err)
    }
    
    // 使用获取到的日志器
    apiLogger.Info("API 请求处理", "path", "/users", "method", "GET")
    dbLogger.Debug("数据库查询", "table", "users", "query", "SELECT * FROM users")
}
```

## 上下文和字段

```go
// 带字段的日志
logger := log.GetLogger("api")
logger.With("requestId", "12345", "userId", "john").Info("处理请求")

// 分组日志
dbLogger := log.GetLogger("db").WithGroup("database")
dbLogger.Info("连接成功", "host", "localhost", "port", 5432)

// 上下文日志
ctx := context.Background()
logger.InfoContext(ctx, "处理完成", "duration", "200ms")
```

## 高级配置

### 多输出器示例

```go
"multi": &ref.TypeOptions{
    Namespace: "github.com/hatlonely/gox/log/logger",
    Type:      "SLog",
    Options: &logger.SLogOptions{
        Level:  "info", 
        Format: "text",
        Output: &ref.TypeOptions{
            Namespace: "github.com/hatlonely/gox/log/writer",
            Type:      "MultiWriter",
            Options: &writer.MultiWriterOptions{
                Writers: []ref.TypeOptions{
                    {
                        Namespace: "github.com/hatlonely/gox/log/writer",
                        Type:      "ConsoleWriter",
                        Options: &writer.ConsoleWriterOptions{
                            Color:  true,
                            Target: "stdout",
                        },
                    },
                    {
                        Namespace: "github.com/hatlonely/gox/log/writer", 
                        Type:      "FileWriter",
                        Options: &writer.FileWriterOptions{
                            Path:       "./logs/app.log",
                            MaxSize:    10,   // 10MB
                            MaxBackups: 5,    // 保留5个备份
                            MaxAge:     30,   // 30天
                            Compress:   true, // 压缩备份
                        },
                    },
                },
            },
        },
    },
},
```

## 直接创建日志器（不推荐）

如果不使用日志管理器，也可以直接创建日志器实例：

### 控制台输出

```go
import (
    "github.com/hatlonely/gox/log/logger"
    "github.com/hatlonely/gox/log/writer"
    "github.com/hatlonely/gox/ref"
)

func main() {
    options := &logger.SLogOptions{
        Level:  "info",
        Format: "text",        // text 或 json
        Output: &ref.TypeOptions{
            Namespace: "github.com/hatlonely/gox/log/writer",
            Type:      "ConsoleWriter",
            Options: &writer.ConsoleWriterOptions{
                Color:  true,   // 彩色输出
                Target: "stdout", // stdout 或 stderr
            },
        },
    }
    
    logger, err := logger.NewSLogWithOptions(options)
    if err != nil {
        panic(err)
    }
    
    logger.Info("用户登录", "userId", "12345")
    logger.Error("处理失败", "error", "网络超时")
}
```

### 文件输出

```go
options := &logger.SLogOptions{
    Level:  "debug",
    Format: "json",
    Output: &ref.TypeOptions{
        Namespace: "github.com/hatlonely/gox/log/writer",
        Type:      "FileWriter",
        Options: &writer.FileWriterOptions{
            Path:       "./logs/app.log",
            MaxSize:    10,   // 10MB
            MaxBackups: 5,    // 保留5个备份
            MaxAge:     30,   // 30天
            Compress:   true, // 压缩备份
        },
    },
}

logger, err := logger.NewSLogWithOptions(options)
// ...
```

## 配置选项

### SLogOptions

```go
type SLogOptions struct {
    Level      string                 // debug, info, warn, error
    Format     string                 // text, json  
    TimeFormat string                 // 时间格式
    AddSource  bool                   // 是否添加源码位置
    Fields     map[string]interface{} // 全局字段
    Output     *ref.TypeOptions       // 输出器配置
}
```

### ConsoleWriterOptions

```go  
type ConsoleWriterOptions struct {
    Color  bool   // 彩色输出
    Target string // stdout, stderr
}
```

### FileWriterOptions

```go
type FileWriterOptions struct {
    Path       string // 文件路径
    MaxSize    int    // 最大文件大小(MB)
    MaxAge     int    // 最大保存天数  
    MaxBackups int    // 最大备份数量
    Compress   bool   // 是否压缩
}
```

## 包结构

```
log/
├── log.go              # 主包，全局日志器
├── manager/            # 日志管理器
│   └── manager.go
├── logger/             # 日志器实现
│   ├── logger.go       # Logger 接口
│   └── slog_logger.go  # SLog 实现
└── writer/             # 输出器
    ├── writer.go       # Writer 接口  
    ├── console_writer.go  # 控制台输出
    ├── file_writer.go  # 文件输出
    └── multi_writer.go # 多输出器
```