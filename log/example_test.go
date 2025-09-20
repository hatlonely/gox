package log_test

import (
	"fmt"
	"os"

	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/log/writer"
	"github.com/hatlonely/gox/ref"
)

func ExampleNewSLogWithOptions_console() {
	// 创建控制台输出的日志配置
	options := &log.SLogOptions{
		Level:      "info",
		Format:     "text",
		TimeFormat: "2006-01-02 15:04:05",
		AddSource:  false,
		Fields: map[string]any{
			"service": "example-service",
			"version": "1.0.0",
		},
		Output: ref.TypeOptions{
			Namespace: "github.com/hatlonely/gox/log/writer",
			Type:      "ConsoleWriter",
			Options: &writer.ConsoleWriterOptions{
				Color:  true,
				Target: "stdout",
			},
		},
	}

	logger, err := log.NewSLogWithOptions(options)
	if err != nil {
		fmt.Printf("创建日志失败: %v\n", err)
		return
	}

	// 使用日志
	logger.Info("这是一条信息日志", "key", "value")
	logger.Warn("这是一条警告日志", "count", 42)
	logger.Error("这是一条错误日志", "error", "something went wrong")

	// 带字段的日志
	contextLogger := logger.With("requestId", "12345", "userId", "user-001")
	contextLogger.Info("处理用户请求")

	// 分组日志
	dbLogger := logger.WithGroup("database")
	dbLogger.Info("连接数据库成功", "host", "localhost", "port", 5432)
}

func ExampleNewSLogWithOptions_file() {
	// 确保日志目录存在
	os.MkdirAll("./logs", 0755)
	defer os.RemoveAll("./logs") // 清理测试文件

	// 创建文件输出的日志配置
	options := &log.SLogOptions{
		Level:      "debug",
		Format:     "json",
		TimeFormat: "2006-01-02T15:04:05Z07:00",
		AddSource:  true,
		Output: ref.TypeOptions{
			Namespace: "github.com/hatlonely/gox/log/writer",
			Type:      "FileWriter",
			Options: &writer.FileWriterOptions{
				Path:       "./logs/app.log",
				MaxSize:    10, // 10MB
				MaxBackups: 3,
				MaxAge:     7,
				Compress:   true,
			},
		},
	}

	logger, err := log.NewSLogWithOptions(options)
	if err != nil {
		fmt.Printf("创建日志失败: %v\n", err)
		return
	}

	logger.Debug("调试信息", "module", "auth")
	logger.Info("用户登录", "userId", "12345", "ip", "192.168.1.100")
	logger.Error("数据库连接失败", "error", "connection timeout")

	fmt.Println("日志已写入文件 ./logs/app.log")
	// Output: 日志已写入文件 ./logs/app.log
}

func ExampleNewSLogWithOptions_multi() {
	// 确保日志目录存在
	os.MkdirAll("./logs", 0755)
	defer os.RemoveAll("./logs") // 清理测试文件

	// 创建多输出的日志配置
	options := &log.SLogOptions{
		Level:  "info",
		Format: "text",
		Output: ref.TypeOptions{
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
							Path: "./logs/multi.log",
						},
					},
				},
			},
		},
	}

	logger, err := log.NewSLogWithOptions(options)
	if err != nil {
		fmt.Printf("创建日志失败: %v\n", err)
		return
	}

	logger.Info("这条日志会同时输出到控制台和文件")
	logger.Warn("警告信息", "component", "cache")

	fmt.Println("日志已输出到控制台和文件")
}

func ExampleNewSLogWithOptions_default() {
	// 使用默认配置（只指定必要参数）
	options := &log.SLogOptions{
		Level: "info",
	}

	logger, err := log.NewSLogWithOptions(options)
	if err != nil {
		fmt.Printf("创建日志失败: %v\n", err)
		return
	}

	logger.Info("使用默认配置的日志", "timestamp", "2023-01-01T12:00:00Z")
}
