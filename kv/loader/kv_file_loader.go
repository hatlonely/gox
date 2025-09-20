// KVFileLoader 从文件加载KV数据，支持监听文件变化
// 该文件必须是文本文件，且每行一个KV数据，格式由KVFileLineParser定义

package loader

import (
	"bufio"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

type KVFileLoaderOptions struct {
	FilePath         string          `cfg:"filePath"`
	KVFileLineParser ref.TypeOptions `cfg:"kvFileLineParser"`
	// 是否跳过脏数据（默认遇到脏数据时，直接报错并返回；启用这个选项的话，仅打印错误日志，不提前返回）
	SkipDirtyRows        bool `cfg:"skipDirtyRows"`
	ScannerBufferMinSize int  `cfg:"scannerBufferMinSize" def:"65536"`
	ScannerBufferMaxSize int  `cfg:"scannerBufferMaxSize" def:"4194304"`
}

type KVFileLoader[K, V any] struct {
	filePath             string
	kvFileLineParser     KVFileLineParser[K, V]
	skipDirtyRows        bool
	scannerBufferMinSize int
	scannerBufferMaxSize int

	done chan struct{}
	wg   sync.WaitGroup
}

func NewKVFileLoaderWithOptions[K, V any](options *KVFileLoaderOptions) (*KVFileLoader[K, V], error) {
	kvFileLineParser, err := NewKVFileLineParserWithOptions[K, V](&options.KVFileLineParser)
	if err != nil {
		return nil, err
	}

	return &KVFileLoader[K, V]{
		filePath:             options.FilePath,
		kvFileLineParser:     kvFileLineParser,
		scannerBufferMinSize: options.ScannerBufferMaxSize,
		scannerBufferMaxSize: options.ScannerBufferMaxSize,
		done:                 make(chan struct{}, 1),
		skipDirtyRows:        options.SkipDirtyRows,
	}, nil
}

func (l *KVFileLoader[K, V]) OnChange(listener Listener[K, V]) error {
	// 加载初始数据
	err := listener(&KVFileStream[K, V]{
		filePath:             l.filePath,
		kvFileLineParser:     l.kvFileLineParser,
		skipDirtyRows:        l.skipDirtyRows,
		scannerBufferMaxSize: l.scannerBufferMaxSize,
		scannerBufferMinSize: l.scannerBufferMinSize,
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
					filePath:         l.filePath,
					kvFileLineParser: l.kvFileLineParser,
					skipDirtyRows:    l.skipDirtyRows,
				})
				if err != nil {
					log.Default().Warn("KVFileLoader.OnChange: listener failed: %v", err)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Default().Warn("KVFileLoader.OnChange: watcher error: %v", err)
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
	kvFileLineParser     KVFileLineParser[K, V]
	skipDirtyRows        bool
	scannerBufferMinSize int
	scannerBufferMaxSize int
}

func (s *KVFileStream[K, V]) Each(handler func(ChangeType, K, V) error) error {
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
		changeType, key, val, err2 := s.kvFileLineParser.Parse(line)

		// debug log for the 1st line
		if rowCount == 1 {
			log.Default().Debug("[kvFileStream] [%s] first row is: %s. parsed key: %s. parsed val: %v", s.filePath, line, key, val)
		}

		if err2 == nil {
			err2 = handler(changeType, key, val)
		}

		if err2 != nil {
			dirtyRowCount++
			if s.skipDirtyRows {
				log.Default().Error("[kvFileStream] [%s] parse failed for line %d, content: %q. err: %s. skip this line", s.filePath, rowCount, line, err2.Error())
				continue
			} else {
				return errors.Wrapf(err2, "[kvFileStream] parse failed for line %d, content: %q", rowCount, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "scanner.Err failed")
	}

	return nil
}

type KVFileLineParser[K, V any] interface {
	Parse(line string) (ChangeType, K, V, error)
}

func NewKVFileLineParserWithOptions[K, V any](options *ref.TypeOptions) (KVFileLineParser[K, V], error) {
	parser, err := ref.NewT[KVFileLineParser[K, V]](options)
	if err != nil {
		return nil, errors.WithMessage(err, "refx.NewT failed")
	}

	return parser, nil
}
