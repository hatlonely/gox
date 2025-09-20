package logger

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hatlonely/gox/log/writer"
	"github.com/hatlonely/gox/ref"
)

// SLogOptions 日志初始化选项
type SLogOptions struct {
	// 日志级别：debug, info, warn, error
	Level string `cfg:"level" validate:"omitempty,oneof=debug info warn error"`

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

type SLog struct {
	slogger *slog.Logger
}

func NewSLogWithOptions(options *SLogOptions) (*SLog, error) {
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

	return &SLog{slogger: slogger}, nil
}

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

func (l *SLog) Debug(msg string, args ...any) {
	l.slogger.Debug(msg, args...)
}

func (l *SLog) Info(msg string, args ...any) {
	l.slogger.Info(msg, args...)
}

func (l *SLog) Warn(msg string, args ...any) {
	l.slogger.Warn(msg, args...)
}

func (l *SLog) Error(msg string, args ...any) {
	l.slogger.Error(msg, args...)
}

func (l *SLog) DebugContext(ctx context.Context, msg string, args ...any) {
	l.slogger.DebugContext(ctx, msg, args...)
}

func (l *SLog) InfoContext(ctx context.Context, msg string, args ...any) {
	l.slogger.InfoContext(ctx, msg, args...)
}

func (l *SLog) WarnContext(ctx context.Context, msg string, args ...any) {
	l.slogger.WarnContext(ctx, msg, args...)
}

func (l *SLog) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.slogger.ErrorContext(ctx, msg, args...)
}

func (l *SLog) With(args ...any) Logger {
	return &SLog{slogger: l.slogger.With(args...)}
}

func (l *SLog) WithGroup(name string) Logger {
	return &SLog{slogger: l.slogger.WithGroup(name)}
}
