package provider

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

type FileProvider struct {
	filePath string
	watcher  *fsnotify.Watcher
	mu       sync.RWMutex
	onChange []func(data []byte) error
	watching bool
	once     sync.Once // 用于确保只初始化一次
}

type FileProviderOptions struct {
	FilePath string
}

func NewFileProviderWithOptions(options *FileProviderOptions) (*FileProvider, error) {
	if options == nil || options.FilePath == "" {
		return nil, errors.New("file path is required")
	}

	absPath, err := filepath.Abs(options.FilePath)
	if err != nil {
		return nil, errors.Wrap(err, "invalid file path")
	}

	return &FileProvider{
		filePath: absPath,
	}, nil
}

func (p *FileProvider) Load() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	data, err := os.ReadFile(p.filePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}

	return data, nil
}

func (p *FileProvider) Save(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := os.WriteFile(p.filePath, data, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	return nil
}

func (p *FileProvider) OnChange(fn func(data []byte) error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 仅仅将新的回调函数添加到队列中
	p.onChange = append(p.onChange, fn)
}

func (p *FileProvider) Watch() error {
	var initErr error
	p.once.Do(func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		// 创建文件监听器
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			initErr = errors.Wrap(err, "failed to create file watcher")
			return
		}

		p.watcher = watcher
		p.watching = true

		// 启动监听 goroutine
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&fsnotify.Write == fsnotify.Write {
						if data, err := os.ReadFile(p.filePath); err == nil {
							// 安全地复制 handler 列表
							p.mu.RLock()
							handlers := make([]func(data []byte) error, len(p.onChange))
							copy(handlers, p.onChange)
							p.mu.RUnlock()

							// 调用所有注册的回调函数
							for _, handler := range handlers {
								if handler != nil {
									handler(data)
								}
							}
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					_ = err
				}
			}
		}()

		// 添加文件所在目录到监听器
		dir := filepath.Dir(p.filePath)
		if err := watcher.Add(dir); err != nil {
			initErr = errors.Wrap(err, "failed to add directory to watcher")
			return
		}
	})

	return initErr
}

func (p *FileProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.watcher != nil {
		return p.watcher.Close()
	}
	return nil
}
