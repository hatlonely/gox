// FileTrigger 监听文件变化并触发通知，不读取文件内容
// 其作用是只通知使用者数据发生了变化，使用者自己加载对应的数据

package loader

import (
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/hatlonely/gox/kv/parser"
	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/log/logger"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

type FileTriggerOptions struct {
	FilePath string           `cfg:"filePath" validate:"required"` // 文件路径
	Logger   *ref.TypeOptions `cfg:"logger"`
}

type FileTrigger[K, V any] struct {
	filePath string

	done chan struct{}
	wg   sync.WaitGroup

	logger logger.Logger
}

func NewFileTriggerWithOptions[K, V any](options *FileTriggerOptions) (*FileTrigger[K, V], error) {
	if options == nil {
		return nil, errors.New("options is nil")
	}

	// 创建 logger (当 options.Logger 为 nil 时，log.NewLoggerWithOptions 自动返回默认 Logger)
	l, err := log.NewLoggerWithOptions(options.Logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logger")
	}

	// 为 logger 添加组和文件路径上下文
	l = l.WithGroup("fileTrigger").With("filePath", options.FilePath)

	return &FileTrigger[K, V]{
		filePath: options.FilePath,
		done:     make(chan struct{}, 1),
		logger:   l,
	}, nil
}

func (t *FileTrigger[K, V]) OnChange(listener Listener[K, V]) error {
	// 触发初始通知（不加载任何数据）
	err := listener(&EmptyKVStream[K, V]{
		logger: t.logger.WithGroup("emptyKVStream"),
	})
	if err != nil {
		return errors.WithMessage(err, "listener failed")
	}

	// 监听文件变化
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "fsnotify.NewWatcher failed")
	}

	err = watcher.Add(filepath.Dir(t.filePath))
	if err != nil {
		return errors.Wrap(err, "watcher.Add failed")
	}

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		defer watcher.Close()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Rename) {
					continue
				}
				if event.Name != t.filePath {
					continue
				}

				// 变化时触发通知（不加载任何数据）
				err := listener(&EmptyKVStream[K, V]{
					logger: t.logger.WithGroup("emptyKVStream"),
				})
				if err != nil {
					t.logger.Warn("listener failed", "error", err)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				t.logger.Warn("watcher error", "error", err)
			case <-t.done:
				return
			}
		}
	}()

	return nil
}

func (t *FileTrigger[K, V]) Close() error {
	t.done <- struct{}{}
	t.wg.Wait()
	close(t.done)
	return nil
}

// EmptyKVStream 空的数据流，不包含任何数据
type EmptyKVStream[K, V any] struct {
	logger logger.Logger
}

func (s *EmptyKVStream[K, V]) Each(handler func(parser.ChangeType, K, V) error) error {
	// 空实现，不调用 handler
	s.logger.Debug("empty stream, no data to process")
	return nil
}