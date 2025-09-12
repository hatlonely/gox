package provider

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type FileProvider struct {
	filePath string
	watcher  *fsnotify.Watcher
	mu       sync.RWMutex
	onChange func(data []byte) error
}

type FileProviderOptions struct {
	FilePath string
}

func NewFileProviderWithOptions(options *FileProviderOptions) (*FileProvider, error) {
	if options == nil || options.FilePath == "" {
		return nil, &ProviderError{Msg: "file path is required"}
	}

	absPath, err := filepath.Abs(options.FilePath)
	if err != nil {
		return nil, &ProviderError{Msg: "invalid file path", Err: err}
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
		return nil, &ProviderError{Msg: "failed to read file", Err: err}
	}

	return data, nil
}

func (p *FileProvider) Save(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := os.WriteFile(p.filePath, data, 0644)
	if err != nil {
		return &ProviderError{Msg: "failed to write file", Err: err}
	}

	return nil
}

func (p *FileProvider) OnChange(fn func(data []byte) error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.onChange = fn

	if p.watcher != nil {
		p.watcher.Close()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}

	p.watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					if data, err := os.ReadFile(p.filePath); err == nil {
						if p.onChange != nil {
							p.onChange(data)
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

	dir := filepath.Dir(p.filePath)
	watcher.Add(dir)
}

func (p *FileProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.watcher != nil {
		return p.watcher.Close()
	}
	return nil
}

type ProviderError struct {
	Msg string
	Err error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
