package log

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hatlonely/gox/log/writer"
	"github.com/hatlonely/gox/ref"
)

// Options 日志初始化选项
type Options struct {
	// 日志级别：debug, info, warn, error
	Level string `cfg:"level"`

	// 输出格式：text, json
	Format string `cfg:"format"`

	// 输出目标配置 - 使用 ref.TypeOptions
	Output ref.TypeOptions `cfg:"output"`

	// 时间格式
	TimeFormat string `cfg:"timeFormat"`

	// 是否显示调用者信息
	AddSource bool `cfg:"addSource"`

	// 自定义字段
	Fields map[string]any `cfg:"fields"`
}

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

	// 自定义日志级别
	Log(ctx context.Context, level slog.Level, msg string, args ...any)

	// 带字段的日志器
	With(args ...any) Logger
	WithGroup(name string) Logger

	// 获取底层的 slog.Logger
	Handler() slog.Handler
}

// logger 实现 Logger 接口
type logger struct {
	slogger *slog.Logger
}

// NewLogWithOptions 根据选项创建日志对象
func NewLogWithOptions(options *Options) (Logger, error) {
	if options == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	// 设置默认值
	if options.Level == "" {
		options.Level = "info"
	}
	if options.Format == "" {
		options.Format = "text"
	}
	if options.TimeFormat == "" {
		options.TimeFormat = time.RFC3339
	}

	// 解析日志级别
	level, err := parseLevel(options.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// 创建输出器
	var w writer.Writer
	if options.Output.Type != "" {
		// 使用 ref 创建输出器
		writerObj, err := ref.New(options.Output.Namespace, options.Output.Type, options.Output.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to create writer: %w", err)
		}

		var ok bool
		w, ok = writerObj.(writer.Writer)
		if !ok {
			return nil, fmt.Errorf("writer object does not implement Writer interface")
		}
	} else {
		// 默认使用控制台输出
		var err error
		w, err = writer.NewConsoleWriterWithOptions(&writer.ConsoleWriterOptions{
			Color:  true,
			Target: "stdout",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create default console writer: %w", err)
		}
	}

	// 创建 handler
	var handler slog.Handler
	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: options.AddSource,
	}

	// 自定义时间格式
	if options.TimeFormat != time.RFC3339 {
		handlerOpts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format(options.TimeFormat)),
				}
			}
			return a
		}
	}

	// 根据格式创建不同的 handler
	switch strings.ToLower(options.Format) {
	case "json":
		handler = slog.NewJSONHandler(w, handlerOpts)
	case "text":
		handler = slog.NewTextHandler(w, handlerOpts)
	default:
		return nil, fmt.Errorf("unsupported format: %s", options.Format)
	}

	// 创建 logger
	slogger := slog.New(handler)

	// 添加自定义字段
	if len(options.Fields) > 0 {
		args := make([]any, 0, len(options.Fields)*2)
		for k, v := range options.Fields {
			args = append(args, k, v)
		}
		slogger = slogger.With(args...)
	}

	return &logger{slogger: slogger}, nil
}

// parseLevel 解析日志级别
func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown level: %s", level)
	}
}

// Debug 记录 debug 级别日志
func (l *logger) Debug(msg string, args ...any) {
	l.slogger.Debug(msg, args...)
}

// Info 记录 info 级别日志
func (l *logger) Info(msg string, args ...any) {
	l.slogger.Info(msg, args...)
}

// Warn 记录 warn 级别日志
func (l *logger) Warn(msg string, args ...any) {
	l.slogger.Warn(msg, args...)
}

// Error 记录 error 级别日志
func (l *logger) Error(msg string, args ...any) {
	l.slogger.Error(msg, args...)
}

// DebugContext 记录带上下文的 debug 级别日志
func (l *logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.slogger.DebugContext(ctx, msg, args...)
}

// InfoContext 记录带上下文的 info 级别日志
func (l *logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.slogger.InfoContext(ctx, msg, args...)
}

// WarnContext 记录带上下文的 warn 级别日志
func (l *logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.slogger.WarnContext(ctx, msg, args...)
}

// ErrorContext 记录带上下文的 error 级别日志
func (l *logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.slogger.ErrorContext(ctx, msg, args...)
}

// Log 记录自定义级别日志
func (l *logger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	l.slogger.Log(ctx, level, msg, args...)
}

// With 返回一个带有指定字段的新日志器
func (l *logger) With(args ...any) Logger {
	return &logger{slogger: l.slogger.With(args...)}
}

// WithGroup 返回一个带有指定分组的新日志器
func (l *logger) WithGroup(name string) Logger {
	return &logger{slogger: l.slogger.WithGroup(name)}
}

// Handler 获取底层的 slog.Handler
func (l *logger) Handler() slog.Handler {
	return l.slogger.Handler()
}
