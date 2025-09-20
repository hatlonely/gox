package log

import (
	"github.com/hatlonely/gox/log/logger"
	"github.com/hatlonely/gox/ref"
)

type Options map[string]*ref.TypeOptions

type LogManager struct {
	loggers map[string]logger.Logger
}

func NewLogManagerWithOptions(options Options) (*LogManager, error) {
}
