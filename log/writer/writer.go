package writer

import (
	"io"

	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
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

func NewWriterWithOptions(options *ref.TypeOptions) (Writer, error) {
	// 处理默认配置
	actualOptions := options
	if actualOptions == nil {
		actualOptions = &ref.TypeOptions{
			Namespace: "github.com/hatlonely/gox/log/writer",
			Type:      "ConsoleWriter",
		}
	}

	writer, err := ref.New(actualOptions.Namespace, actualOptions.Type, actualOptions.Options)
	if err != nil {
		return nil, errors.WithMessage(err, "refx.NewT failed")
	}
	if writer == nil {
		return nil, errors.New("writer is nil")
	}
	if _, ok := writer.(Writer); !ok {
		return nil, errors.New("writer is not a Writer")
	}

	return writer.(Writer), nil
}
