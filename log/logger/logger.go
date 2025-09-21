package logger

import (
	"context"

	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

func init() {
	ref.MustRegisterT[*SLog](NewSLogWithOptions)
	ref.MustRegisterT[SLog](NewSLogWithOptions)
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

	// 带字段的日志器
	With(args ...any) Logger
	WithGroup(name string) Logger
}

func NewLoggerWithOptions(options *ref.TypeOptions) (Logger, error) {
	if options == nil {
		return nil, errors.New("options cannot be nil")
	}
	logger, err := ref.New(options.Namespace, options.Type, options.Options)
	if err != nil {
		return nil, errors.WithMessage(err, "refx.NewT failed")
	}
	if logger == nil {
		return nil, errors.New("logger is nil")
	}
	if _, ok := logger.(Logger); !ok {
		return nil, errors.New("logger is not a Logger")
	}

	return logger.(Logger), nil
}
