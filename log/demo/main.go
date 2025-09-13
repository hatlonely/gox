package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/log/writer"
	"github.com/hatlonely/gox/refx"
)

func main() {
	// 示例1: 基本的控制台输出
	fmt.Println("=== 示例1: 控制台输出 ===")
	demoConsoleLog()

	fmt.Println("\n=== 示例2: 文件输出 ===")
	demoFileLog()

	fmt.Println("\n=== 示例3: 多输出 ===")
	demoMultiLog()

	fmt.Println("\n=== 示例4: 使用cfg配置 ===")
	demoCfgLog()
}

func demoConsoleLog() {
	options := &log.Options{
		Level:      "debug",
		Format:     "text",
		TimeFormat: "2006-01-02 15:04:05",
		AddSource:  false,
		Fields: map[string]any{
			"service": "demo-service",
			"version": "1.0.0",
		},
		Output: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/log/writer",
			Type:      "ConsoleWriter",
			Options: &writer.ConsoleWriterOptions{
				Color:  true,
				Target: "stdout",
			},
		},
	}

	logger, err := log.NewLogWithOptions(options)
	if err != nil {
		fmt.Printf("创建日志失败: %v\n", err)
		return
	}

	logger.Debug("这是调试信息", "module", "auth")
	logger.Info("用户登录成功", "userId", "12345", "ip", "192.168.1.100")
	logger.Warn("缓存命中率低", "hitRate", 0.3)
	logger.Error("数据库连接失败", "error", "connection timeout")

	// 带上下文
	ctx := context.Background()
	logger.InfoContext(ctx, "处理请求完成", "duration", "150ms")

	// 带字段的子日志器
	userLogger := logger.With("userId", "user-001", "sessionId", "session-123")
	userLogger.Info("用户操作", "action", "create_post", "postId", "post-456")

	// 分组日志器
	dbLogger := logger.WithGroup("database")
	dbLogger.Info("执行查询", "table", "users", "duration", "50ms")
}

func demoFileLog() {
	// 确保日志目录存在
	os.MkdirAll("./logs", 0755)
	defer func() {
		// 演示完成后可以选择清理
		fmt.Println("日志文件保存在 ./logs/ 目录中")
	}()

	options := &log.Options{
		Level:      "info",
		Format:     "json",
		TimeFormat: "2006-01-02T15:04:05Z07:00",
		AddSource:  true,
		Output: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/log/writer",
			Type:      "FileWriter",
			Options: &writer.FileWriterOptions{
				Path:       "./logs/app.log",
				MaxSize:    10, // 10MB
				MaxBackups: 3,
				MaxAge:     7,
				Compress:   false, // 演示时不压缩
			},
		},
	}

	logger, err := log.NewLogWithOptions(options)
	if err != nil {
		fmt.Printf("创建日志失败: %v\n", err)
		return
	}

	logger.Info("应用启动", "version", "2.0.0", "env", "development")
	logger.Warn("配置项缺失", "key", "redis.host", "default", "localhost")
	logger.Error("外部API调用失败", "api", "user-service", "status", 500)

	fmt.Println("日志已写入 ./logs/app.log")
}

func demoMultiLog() {
	// 确保日志目录存在
	os.MkdirAll("./logs", 0755)

	options := &log.Options{
		Level:  "info",
		Format: "text",
		Output: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/log/writer",
			Type:      "MultiWriter",
			Options: &writer.MultiWriterOptions{
				Writers: []refx.TypeOptions{
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
							Path: "./logs/multi.log",
						},
					},
				},
			},
		},
	}

	logger, err := log.NewLogWithOptions(options)
	if err != nil {
		fmt.Printf("创建日志失败: %v\n", err)
		return
	}

	logger.Info("多输出演示", "target", "console+file")
	logger.Warn("同时输出到控制台和文件", "component", "multi-writer")
	logger.Error("错误信息", "error", "示例错误")

	fmt.Println("日志同时输出到控制台和 ./logs/multi.log")
}

func demoCfgLog() {
	// 这里演示如何与cfg库集成使用
	fmt.Println("配置文件集成示例 (需要配置文件)")
	fmt.Println("可以创建 config.yaml 文件，内容如下:")
	fmt.Println(`
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
            target: "stdout"
        - namespace: "github.com/hatlonely/gox/log/writer"
          type: "FileWriter"
          options:
            path: "./logs/config.log"
            maxSize: 100
            maxBackups: 3
            maxAge: 7
            compress: true

然后使用:
  cfg, err := cfg.NewConfig("config.yaml")
  if err != nil {
      // 处理错误
  }
  
  var logOptions log.Options
  err = cfg.Sub("log").ConvertTo(&logOptions)
  if err != nil {
      // 处理错误
  }
  
  logger, err := log.NewLogWithOptions(&logOptions)
  if err != nil {
      // 处理错误
  }`)
}