package log

import (
	"fmt"

	"github.com/hatlonely/gox/log/logger"
	"github.com/hatlonely/gox/ref"
)

type Options map[string]*ref.TypeOptions

type LogManager struct {
	loggers       map[string]logger.Logger
	defaultLogger logger.Logger
}

func NewLogManagerWithOptions(options Options) (*LogManager, error) {
	manager := &LogManager{
		loggers: make(map[string]logger.Logger),
	}

	// 创建每个配置的日志器
	for name, typeOpts := range options {
		if typeOpts == nil {
			continue
		}

		// 使用 ref 创建日志器实例
		loggerObj, err := ref.New(typeOpts.Namespace, typeOpts.Type, typeOpts.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to create logger '%s': %w", name, err)
		}

		// 检查类型是否实现了 Logger 接口
		loggerInstance, ok := loggerObj.(logger.Logger)
		if !ok {
			return nil, fmt.Errorf("logger '%s' does not implement Logger interface", name)
		}

		manager.loggers[name] = loggerInstance

		// 如果名称是 "default"，设置为默认日志器
		if name == "default" {
			manager.defaultLogger = loggerInstance
		}
	}

	// 如果没有设置默认日志器，使用全局默认的
	if manager.defaultLogger == nil {
		manager.defaultLogger = Default()
	}

	return manager, nil
}

// GetLogger 获取指定名称的日志器
func (m *LogManager) GetLogger(name string) logger.Logger {
	if l, ok := m.loggers[name]; ok {
		return l
	}
	// 如果找不到指定名称的日志器，返回默认日志器
	return m.defaultLogger
}

// GetLoggerExists 获取指定名称的日志器，返回日志器和是否存在的标志
func (m *LogManager) GetLoggerExists(name string) (logger.Logger, bool) {
	l, exists := m.loggers[name]
	return l, exists
}

// ListLoggers 返回所有已注册的日志器名称
func (m *LogManager) ListLoggers() []string {
	names := make([]string, 0, len(m.loggers))
	for name := range m.loggers {
		names = append(names, name)
	}
	return names
}

// GetDefault 获取默认日志器
func (m *LogManager) GetDefault() logger.Logger {
	return m.defaultLogger
}

// SetDefault 设置默认日志器
func (m *LogManager) SetDefault(l logger.Logger) {
	if l != nil {
		m.defaultLogger = l
	}
}

// SetDefaultByName 通过名称设置默认日志器
func (m *LogManager) SetDefaultByName(name string) error {
	if l, ok := m.loggers[name]; ok {
		m.defaultLogger = l
		return nil
	}
	return fmt.Errorf("logger '%s' not found", name)
}
