package writer

import (
	"io"
)

// Writer 日志输出器接口
type Writer interface {
	io.Writer
	io.Closer
}