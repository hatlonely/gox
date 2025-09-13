package writer

import (
	"io"

	"github.com/hatlonely/gox/refx"
)

func init() {
	// 注册所有输出器到 refx 框架
	refx.MustRegisterT[ConsoleWriter](NewConsoleWriterWithOptions)
	refx.MustRegisterT[FileWriter](NewFileWriterWithOptions)
	refx.MustRegisterT[MultiWriter](NewMultiWriterWithOptions)

	refx.MustRegisterT[*ConsoleWriter](NewConsoleWriterWithOptions)
	refx.MustRegisterT[*FileWriter](NewFileWriterWithOptions)
	refx.MustRegisterT[*MultiWriter](NewMultiWriterWithOptions)
}

// Writer 日志输出器接口
type Writer interface {
	io.Writer
	io.Closer
}
