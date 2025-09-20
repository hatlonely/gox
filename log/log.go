package log

import (
	"github.com/hatlonely/gox/log/logger"
	"github.com/hatlonely/gox/ref"
)

var (
	defaultLogger     logger.Logger
	defaultLogManager *LogManager
)

func init() {
	// 创建默认的SLog实例，向终端输出text格式日志
	slog, err := logger.NewSLogWithOptions(&logger.SLogOptions{
		Level:  "info",
		Format: "text",
	})
	if err != nil {
		panic("failed to initialize default logger: " + err.Error())
	}
	defaultLogger = slog

	ref.Register("github.com/hatlonely/gox/log", "Logger", func(name string) logger.Logger {
		return GetLogger(name)
	})
}

func Default() logger.Logger {
	return defaultLogger
}

// Init 初始化默认的 LogManager
func Init(options Options) error {
	manager, err := NewLogManagerWithOptions(options)
	if err != nil {
		return err
	}
	defaultLogManager = manager

	// 更新默认日志器为 LogManager 的默认日志器
	defaultLogger = manager.GetDefault()

	return nil
}

// Manager 获取默认的 LogManager
func Manager() *LogManager {
	return defaultLogManager
}

// GetLogger 从默认 LogManager 获取指定名称的日志器
func GetLogger(name string) logger.Logger {
	if defaultLogManager != nil {
		return defaultLogManager.GetLogger(name)
	}
	return defaultLogger
}
