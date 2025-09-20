package logger

import (
	"context"
)

// Logger 日志接口
type Logger interface {
	// 基础日志方法
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	// 带上下文的日志方法
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)

	// 带字段的日志器
	With(args ...any) Logger
	WithGroup(name string) Logger
}
