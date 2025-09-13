package writer

import (
	"io"

	"github.com/hatlonely/gox/refx"
)

func init() {
	// 注册所有输出器到 refx 框架
	refx.MustRegister("github.com/hatlonely/gox/log/writer", "ConsoleWriter", NewConsoleWriterWithOptions)
	refx.MustRegister("github.com/hatlonely/gox/log/writer", "FileWriter", NewFileWriterWithOptions)
	refx.MustRegister("github.com/hatlonely/gox/log/writer", "MultiWriter", NewMultiWriterWithOptions)
}

// Writer 日志输出器接口
type Writer interface {
	io.Writer
	io.Closer
}
