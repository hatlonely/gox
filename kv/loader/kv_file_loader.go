// KVFileLoader 从文件加载KV数据，支持监听文件变化
// 该文件必须是文本文件，且每行一个KV数据，格式由KVFileLineParser定义

package loader

import (
	"bufio"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/hatlonely/gox/kv/parser"
	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/log/logger"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

type KVFileLoaderOptions struct {
	FilePath string          `cfg:"filePath" validate:"required"` // 文件路径
	Parser   ref.TypeOptions `cfg:"parser"`
	// 是否跳过脏数据（默认遇到脏数据时，直接报错并返回；启用这个选项的话，仅打印错误日志，不提前返回）
	SkipDirtyRows        bool `cfg:"skipDirtyRows"`
	ScannerBufferMinSize int  `cfg:"scannerBufferMinSize" def:"65536"`
	ScannerBufferMaxSize int  `cfg:"scannerBufferMaxSize" def:"4194304"`
}

type KVFileLoader[K, V any] struct {
	filePath             string
	parser               parser.Parser[K, V]
	skipDirtyRows        bool
	scannerBufferMinSize int
	scannerBufferMaxSize int

	done chan struct{}
	wg   sync.WaitGroup

	logger logger.Logger
}

func NewKVFileLoaderWithOptions[K, V any](options *KVFileLoaderOptions) (*KVFileLoader[K, V], error) {
	if options == nil {
		return nil, errors.New("options is nil")
	}

	if options.ScannerBufferMinSize <= 0 {
		options.ScannerBufferMinSize = 64 * 1024
	}
	if options.ScannerBufferMaxSize <= 0 {
		options.ScannerBufferMaxSize = 4 * 1024 * 1024
	}

	p, err := parser.NewParserWithOptions[K, V](&options.Parser)
	if err != nil {
		return nil, err
	}

	return &KVFileLoader[K, V]{
		filePath:             options.FilePath,
		parser:               p,
		scannerBufferMinSize: options.ScannerBufferMaxSize,
		scannerBufferMaxSize: options.ScannerBufferMaxSize,
		done:                 make(chan struct{}, 1),
		skipDirtyRows:        options.SkipDirtyRows,
		logger:               log.Default().WithGroup("kvFileLoader").With("filePath", options.FilePath),
	}, nil
}

func (l *KVFileLoader[K, V]) OnChange(listener Listener[K, V]) error {
	// 加载初始数据
	err := listener(&KVFileStream[K, V]{
		filePath:             l.filePath,
		kvFileLineParser:     l.parser,
		skipDirtyRows:        l.skipDirtyRows,
		scannerBufferMaxSize: l.scannerBufferMaxSize,
		scannerBufferMinSize: l.scannerBufferMinSize,
		logger:               l.logger.WithGroup("kvFileStream"),
	})
	if err != nil {
		return errors.WithMessage(err, "listener failed")
	}

	// 监听文件变化
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "fsnotify.NewWatcher failed")
	}

	err = watcher.Add(filepath.Dir(l.filePath))
	if err != nil {
		return errors.Wrap(err, "watcher.Add failed")
	}

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
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
				if event.Name != l.filePath {
					continue
				}

				// 变化时触发数据加载
				err := listener(&KVFileStream[K, V]{
					filePath:             l.filePath,
					kvFileLineParser:     l.parser,
					skipDirtyRows:        l.skipDirtyRows,
					scannerBufferMaxSize: l.scannerBufferMaxSize,
					scannerBufferMinSize: l.scannerBufferMinSize,
					logger:               l.logger.WithGroup("kvFileStream"),
				})
				if err != nil {
					l.logger.Warn("listener failed", "error", err)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				l.logger.Warn("watcher error", "error", err)
			case <-l.done:
				return
			}
		}
	}()

	return nil
}

func (l *KVFileLoader[K, V]) Close() error {
	l.done <- struct{}{}
	l.wg.Wait()
	close(l.done)
	return nil
}

type KVFileStream[K, V any] struct {
	filePath             string
	kvFileLineParser     parser.Parser[K, V]
	skipDirtyRows        bool
	scannerBufferMinSize int
	scannerBufferMaxSize int
	logger               logger.Logger
}

func (s *KVFileStream[K, V]) Each(handler func(parser.ChangeType, K, V) error) error {
	fp, err := os.Open(s.filePath)
	if err != nil {
		return errors.Wrap(err, "os.Open failed")
	}
	defer fp.Close()
	scanner := bufio.NewScanner(fp)
	buf := make([]byte, 0, s.scannerBufferMinSize)
	scanner.Buffer(buf, s.scannerBufferMaxSize)

	rowCount := 0
	dirtyRowCount := 0
	for scanner.Scan() {
		rowCount++
		line := scanner.Text()
		changeType, key, val, err2 := s.kvFileLineParser.Parse([]byte(line))

		// debug log for the 1st line
		if rowCount == 1 {
			s.logger.Debug("first row parsed", "line", line, "key", key, "val", val)
		}

		if err2 == nil {
			err2 = handler(changeType, key, val)
		}

		if err2 != nil {
			dirtyRowCount++
			if s.skipDirtyRows {
				s.logger.Error("parse failed, skipping line", "lineNumber", rowCount, "content", line, "error", err2)
				continue
			} else {
				return errors.Wrapf(err2, "parse failed for line %d, content: %q", rowCount, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "scanner.Err failed")
	}

	return nil
}
