package log

import (
	"github.com/hatlonely/gox/log/writer"
	"github.com/hatlonely/gox/refx"
)

func init() {
	// 注册所有输出器到 refx 框架
	refx.MustRegister("github.com/hatlonely/gox/log/writer", "ConsoleWriter", writer.NewConsoleWriter)
	refx.MustRegister("github.com/hatlonely/gox/log/writer", "FileWriter", writer.NewFileWriter)
	refx.MustRegister("github.com/hatlonely/gox/log/writer", "MultiWriter", writer.NewMultiWriter)
}