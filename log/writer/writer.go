package writer

import (
	"io"

	"github.com/hatlonely/gox/ref"
)

func init() {
	// 注册所有输出器到 ref 框架
	ref.MustRegisterT[ConsoleWriter](NewConsoleWriterWithOptions)
	ref.MustRegisterT[FileWriter](NewFileWriterWithOptions)
	ref.MustRegisterT[MultiWriter](NewMultiWriterWithOptions)

	ref.MustRegisterT[*ConsoleWriter](NewConsoleWriterWithOptions)
	ref.MustRegisterT[*FileWriter](NewFileWriterWithOptions)
	ref.MustRegisterT[*MultiWriter](NewMultiWriterWithOptions)
}

// Writer 日志输出器接口
type Writer interface {
	io.Writer
	io.Closer
}
