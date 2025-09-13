package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileWriterOptions 文件输出配置
type FileWriterOptions struct {
	// 文件路径
	Path string `cfg:"path"`
	// 最大文件大小（MB），0表示不限制
	MaxSize int `cfg:"maxSize"`
	// 最大备份数量，0表示不限制
	MaxBackups int `cfg:"maxBackups"`
	// 最大保留天数，0表示不限制
	MaxAge int `cfg:"maxAge"`
	// 是否压缩旧文件
	Compress bool `cfg:"compress"`
}

// FileWriter 文件输出器
type FileWriter struct {
	options *FileWriterOptions
	file    *os.File
	mu      sync.Mutex
}

// NewFileWriterWithOptions 创建文件输出器
func NewFileWriterWithOptions(options *FileWriterOptions) (*FileWriter, error) {
	if options == nil || options.Path == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// 确保目录存在
	dir := filepath.Dir(options.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// 打开或创建文件
	file, err := os.OpenFile(options.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", options.Path, err)
	}

	return &FileWriter{
		options: options,
		file:    file,
	}, nil
}

// Write 实现 io.Writer 接口
func (f *FileWriter) Write(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file == nil {
		return 0, fmt.Errorf("file is closed")
	}

	// TODO: 实现文件轮转逻辑
	// 这里可以后续集成 lumberjack 或自己实现轮转逻辑
	return f.file.Write(p)
}

// Close 实现 io.Closer 接口
func (f *FileWriter) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file != nil {
		err := f.file.Close()
		f.file = nil
		return err
	}
	return nil
}
