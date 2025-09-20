package log

import "github.com/hatlonely/gox/log/logger"

var defaultLogger logger.Logger

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
}

func Default() logger.Logger {
	return defaultLogger
}
