# Log - 基于标准 slog 的日志库

基于 Go 1.21+ 标准 `log/slog` 包构建的结构化日志库，与 [cfg](../cfg) 配置库完全集成。

## 快速开始

### 基本用法

```go
package main

import (
    "github.com/hatlonely/gox/log"
    "github.com/hatlonely/gox/log/writer"
    "github.com/hatlonely/gox/refx"
)

func main() {
    // 创建日志器
    logger, err := log.NewLogWithOptions(&log.Options{
        Level:  "info",
        Format: "json",
        Output: refx.TypeOptions{
            Namespace: "github.com/hatlonely/gox/log/writer",
            Type:      "ConsoleWriter",
            Options: &writer.ConsoleWriterOptions{
                Color:  true,
                Target: "stdout",
            },
        },
    })
    if err != nil {
        panic(err)
    }

    // 使用日志
    logger.Info("应用启动", "version", "1.0.0")
    logger.Warn("连接池容量不足", "current", 10, "max", 100)
    logger.Error("数据库连接失败", "error", err)
}
```

## 配置选项

### 日志级别
- `debug` - 调试信息
- `info` - 一般信息  
- `warn` - 警告信息
- `error` - 错误信息

### 输出格式
- `text` - 文本格式（默认）
- `json` - JSON 格式

### 时间格式
- 支持 Go 时间格式，默认 RFC3339
- 示例：`"2006-01-02 15:04:05"`

## 输出器类型

### 控制台输出
```go
Output: refx.TypeOptions{
    Namespace: "github.com/hatlonely/gox/log/writer",
    Type:      "ConsoleWriter",
    Options: &writer.ConsoleWriterOptions{
        Color:  true,    // 彩色输出
        Target: "stdout", // stdout 或 stderr
    },
}
```

### 文件输出
```go
Output: refx.TypeOptions{
    Namespace: "github.com/hatlonely/gox/log/writer",
    Type:      "FileWriter", 
    Options: &writer.FileWriterOptions{
        Path:       "./logs/app.log",
        MaxSize:    100,  // MB
        MaxBackups: 3,    // 备份文件数
        MaxAge:     7,    // 保留天数
        Compress:   true, // 压缩旧文件
    },
}
```

### 多输出
```go
Output: refx.TypeOptions{
    Namespace: "github.com/hatlonely/gox/log/writer",
    Type:      "MultiWriter",
    Options: &writer.MultiWriterOptions{
        Writers: []refx.TypeOptions{
            // 控制台输出
            {
                Namespace: "github.com/hatlonely/gox/log/writer",
                Type:      "ConsoleWriter",
                Options: &writer.ConsoleWriterOptions{Color: true},
            },
            // 文件输出
            {
                Namespace: "github.com/hatlonely/gox/log/writer",
                Type:      "FileWriter",
                Options: &writer.FileWriterOptions{
                    Path: "./logs/app.log",
                },
            },
        },
    },
}
```

## 配置文件集成

### YAML 配置
```yaml
log:
  level: info
  format: json
  timeFormat: "2006-01-02 15:04:05"
  addSource: true
  fields:
    service: "my-service"
    version: "1.0.0"
  output:
    namespace: "github.com/hatlonely/gox/log/writer"
    type: "MultiWriter"
    options:
      writers:
        - namespace: "github.com/hatlonely/gox/log/writer"
          type: "ConsoleWriter"
          options:
            color: true
        - namespace: "github.com/hatlonely/gox/log/writer"
          type: "FileWriter"
          options:
            path: "./logs/app.log"
            maxSize: 100
            maxBackups: 3
            maxAge: 7
            compress: true
```

### 使用配置文件
```go
import (
    "github.com/hatlonely/gox/cfg"
    "github.com/hatlonely/gox/log"
)

// 加载配置
config, err := cfg.NewConfig("config.yaml")
if err != nil {
    panic(err)
}

// 转换为日志选项
var logOptions log.Options
err = config.Sub("log").ConvertTo(&logOptions)
if err != nil {
    panic(err)
}

// 创建日志器
logger, err := log.NewLogWithOptions(&logOptions)
if err != nil {
    panic(err)
}
```

## 高级用法

### 带字段的日志
```go
// 添加固定字段
userLogger := logger.With("userId", "12345", "requestId", "req-001")
userLogger.Info("处理用户请求")

// 分组字段
dbLogger := logger.WithGroup("database")
dbLogger.Info("查询执行", "table", "users", "duration", "50ms")
// 输出: {"database":{"table":"users","duration":"50ms"},...}
```

### 上下文日志
```go
import "context"

ctx := context.Background()
logger.InfoContext(ctx, "处理请求", "path", "/api/users")
logger.ErrorContext(ctx, "请求失败", "error", err)
```

### 自定义级别
```go
import "log/slog"

logger.Log(ctx, slog.LevelDebug, "自定义调试信息", "key", "value")
```

## 默认配置

不提供输出配置时，会使用默认的控制台输出：
```go
logger, err := log.NewLogWithOptions(&log.Options{
    Level: "info",
})
// 等同于彩色控制台输出，文本格式
```

## 性能说明

- 基于标准库 `log/slog`，性能优异
- 结构化日志，便于日志分析
- 支持延迟求值，避免不必要的计算
- 零分配的快速路径优化

## 扩展开发

可以通过实现 `writer.Writer` 接口来添加自定义输出器：

```go
type Writer interface {
    io.Writer
    io.Closer
}
```

然后使用 `refx.MustRegister` 注册到框架中。