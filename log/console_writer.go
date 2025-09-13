package log

import (
	"io"
	"os"
)

// ConsoleWriterOptions 控制台输出配置
type ConsoleWriterOptions struct {
	// 是否彩色输出
	Color bool `cfg:"color"`
	// 输出目标：stdout, stderr
	Target string `cfg:"target"`
}

// ConsoleWriter 控制台输出器
type ConsoleWriter struct {
	writer io.Writer
	color  bool
}

// NewConsoleWriter 创建控制台输出器
func NewConsoleWriter(options *ConsoleWriterOptions) (*ConsoleWriter, error) {
	if options == nil {
		options = &ConsoleWriterOptions{
			Color:  true,
			Target: "stdout",
		}
	}

	var writer io.Writer
	switch options.Target {
	case "stderr":
		writer = os.Stderr
	case "stdout", "":
		writer = os.Stdout
	default:
		writer = os.Stdout
	}

	return &ConsoleWriter{
		writer: writer,
		color:  options.Color,
	}, nil
}

// Write 实现 io.Writer 接口
func (c *ConsoleWriter) Write(p []byte) (n int, err error) {
	return c.writer.Write(p)
}

// Close 实现 io.Closer 接口
func (c *ConsoleWriter) Close() error {
	// 控制台不需要关闭
	return nil
}