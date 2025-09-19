package writer

import (
	"fmt"
	"io"

	"github.com/hatlonely/gox/ref"
)

// MultiWriterOptions 多输出配置
type MultiWriterOptions struct {
	// 输出器列表
	Writers []ref.TypeOptions `cfg:"writers"`
}

// MultiWriter 多输出器
type MultiWriter struct {
	writers []Writer
}

// NewMultiWriterWithOptions 创建多输出器
func NewMultiWriterWithOptions(options *MultiWriterOptions) (*MultiWriter, error) {
	if options == nil || len(options.Writers) == 0 {
		return nil, fmt.Errorf("at least one writer is required")
	}

	writers := make([]Writer, 0, len(options.Writers))

	for i, writerOpt := range options.Writers {
		// 使用 ref 创建输出器
		writerObj, err := ref.New(writerOpt.Namespace, writerOpt.Type, writerOpt.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to create writer %d: %w", i, err)
		}

		writer, ok := writerObj.(Writer)
		if !ok {
			return nil, fmt.Errorf("writer %d does not implement Writer interface", i)
		}

		writers = append(writers, writer)
	}

	return &MultiWriter{
		writers: writers,
	}, nil
}

// Write 实现 io.Writer 接口，写入所有输出器
func (m *MultiWriter) Write(p []byte) (n int, err error) {
	for i, writer := range m.writers {
		n, err = writer.Write(p)
		if err != nil {
			return n, fmt.Errorf("writer %d failed: %w", i, err)
		}
	}
	return len(p), nil
}

// Close 实现 io.Closer 接口，关闭所有输出器
func (m *MultiWriter) Close() error {
	var lastErr error
	for i, writer := range m.writers {
		if err := writer.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close writer %d: %w", i, err)
		}
	}
	return lastErr
}

// multiWriter 是标准库 io.MultiWriter 的封装，提供 Close 方法
type multiWriter struct {
	writers []io.Writer
	closers []io.Closer
}

// NewMultiWriterFromWriters 从已有的 Writer 创建多输出器
func NewMultiWriterFromWriters(writers ...Writer) *multiWriter {
	ioWriters := make([]io.Writer, len(writers))
	closers := make([]io.Closer, len(writers))

	for i, w := range writers {
		ioWriters[i] = w
		closers[i] = w
	}

	return &multiWriter{
		writers: ioWriters,
		closers: closers,
	}
}

// Write 实现 io.Writer 接口
func (m *multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range m.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}

// Close 实现 io.Closer 接口
func (m *multiWriter) Close() error {
	var lastErr error
	for _, closer := range m.closers {
		if err := closer.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
