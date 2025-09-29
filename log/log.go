package log

import (
	"github.com/hatlonely/gox/log/logger"
	"github.com/hatlonely/gox/log/manager"
	"github.com/hatlonely/gox/ref"
)

var (
	defaultLogger     logger.Logger
	defaultLogManager *manager.LogManager
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

	ref.Register("github.com/hatlonely/gox/log", "GetLogger", GetLogger)
}

func Default() logger.Logger {
	return defaultLogger
}

// Init 初始化默认的 LogManager
func Init(options manager.Options) error {
	mgr, err := manager.NewLogManagerWithOptions(options)
	if err != nil {
		return err
	}
	defaultLogManager = mgr

	// 设置默认日志器如果 LogManager 的默认日志器为 nil
	mgr.SetDefaultLoggerIfNil(defaultLogger)

	// 更新默认日志器为 LogManager 的默认日志器
	defaultLogger = mgr.GetDefault()

	return nil
}

// Manager 获取默认的 LogManager
func Manager() *manager.LogManager {
	return defaultLogManager
}

// GetLogger 从默认 LogManager 获取指定名称的日志器
func GetLogger(name string) logger.Logger {
	if defaultLogManager != nil {
		return defaultLogManager.GetLogger(name)
	}
	return defaultLogger
}

// NewLoggerWithOptions 使用指定配置创建日志器
// 当 options 为 nil 时，返回默认日志器
func NewLoggerWithOptions(options *ref.TypeOptions) (logger.Logger, error) {
	if options == nil {
		return defaultLogger, nil
	}
	return logger.NewLoggerWithOptions(options)
}
