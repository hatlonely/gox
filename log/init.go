package log

import (
	"github.com/hatlonely/gox/refx"
)

func init() {
	// 注册所有输出器到 refx 框架
	refx.MustRegister("github.com/hatlonely/gox/log", "ConsoleWriter", NewConsoleWriter)
	refx.MustRegister("github.com/hatlonely/gox/log", "FileWriter", NewFileWriter)
	refx.MustRegister("github.com/hatlonely/gox/log", "MultiWriter", NewMultiWriter)
}