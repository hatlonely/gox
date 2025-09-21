package cfg

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hatlonely/gox/cfg/decoder"
	"github.com/hatlonely/gox/cfg/provider"
	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/log/logger"
	"github.com/hatlonely/gox/ref"
)

// ConfigSource 配置源，包含 Provider、Decoder 和当前存储的数据
type ConfigSource struct {
	provider provider.Provider // 配置数据提供者
	decoder  decoder.Decoder   // 配置数据解码器
	storage  storage.Storage   // 当前配置源的数据
}

// ConfigSourceOptions 配置源选项，用于创建配置源
type ConfigSourceOptions struct {
	Provider ref.TypeOptions `cfg:"provider"`
	Decoder  ref.TypeOptions `cfg:"decoder"`
}

// MultiConfigOptions 多配置管理器初始化选项
type MultiConfigOptions struct {
	// 配置源数组，按优先级排序（索引越大优先级越高）
	// 配置合并策略：
	//   - 结构体：字段级覆盖，后面的配置覆盖前面的配置，不存在的字段保持原值
	//   - map：增量合并，新键被添加，已存在的键被覆盖，其他键被保留
	//   - 其他类型：按照各 Storage 实现的语义处理
	// 示例：[基础配置文件, 环境变量配置, 数据库配置] -> 数据库配置优先级最高
	Sources []*ConfigSourceOptions `cfg:"sources"`

	// 可选的日志配置，用于记录配置变更和处理器执行情况
	Logger *ref.TypeOptions `cfg:"logger"`

	// 可选的处理器执行配置，控制 OnChange/OnKeyChange 回调的执行行为
	// 包括超时时长、异步/同步执行、错误处理策略等
	HandlerExecution *HandlerExecutionOptions `cfg:"handlerExecution"`
}

// MultiConfig 多配置管理器
// 支持从多个配置源获取配置数据，并按优先级合并
type MultiConfig struct {
	// 配置源数组，索引越大优先级越高（后面的覆盖前面的）
	sources []ConfigSource

	// 多配置存储
	multiStorage storage.MultiStorage

	// 通用配置
	logger           logger.Logger
	handlerExecution *HandlerExecutionOptions

	// 变更监听相关
	onKeyChangeHandlers map[string][]func(storage.Storage) error

	// 子配置支持
	parent *MultiConfig
	prefix string

	// 关闭控制
	closeMu     sync.Mutex
	closed      bool
	closeResult error
}

// NewMultiConfigWithOptions 根据选项创建多配置对象
func NewMultiConfigWithOptions(options *MultiConfigOptions) (*MultiConfig, error) {
	if options == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	if len(options.Sources) == 0 {
		return nil, fmt.Errorf("at least one configuration source is required")
	}

	// 创建配置源
	sources := make([]ConfigSource, len(options.Sources))
	storages := make([]storage.Storage, len(options.Sources))

	for i, sourceOptions := range options.Sources {
		// 创建 Provider 实例
		prov, err := provider.NewProviderWithOptions(&sourceOptions.Provider)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %d: %w", i, err)
		}

		// 创建 Decoder 实例
		dec, err := decoder.NewDecoderWithOptions(&sourceOptions.Decoder)
		if err != nil {
			return nil, fmt.Errorf("failed to create decoder %d: %w", i, err)
		}

		// 从 Provider 加载数据
		data, err := prov.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load data from provider %d: %w", i, err)
		}

		// 用 Decoder 解码数据为 Storage
		stor, err := dec.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode data from source %d: %w", i, err)
		}

		// 用 ValidateStorage 包装 storage 以提供自动校验功能
		stor = storage.NewValidateStorage(stor)

		sources[i] = ConfigSource{
			provider: prov,
			decoder:  dec,
			storage:  stor,
		}
		storages[i] = stor
	}

	// 创建 MultiStorage
	multiStorage := storage.NewMultiStorage(storages)

	// 创建或使用默认 Logger
	var log logger.Logger
	if options.Logger != nil {
		// 使用提供的日志配置创建 Logger
		var err error
		log, err = logger.NewLoggerWithOptions(options.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	} else {
		// 创建默认的终端输出 Logger
		var err error
		log, err = logger.NewLoggerWithOptions(&ref.TypeOptions{
			Namespace: "github.com/hatlonely/gox/log/logger",
			Type:      "SLog",
			Options: &logger.SLogOptions{
				Level:  "info",
				Format: "text",
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create default logger: %w", err)
		}
	}

	// 设置默认的 handler 执行配置
	handlerExecution := options.HandlerExecution
	if handlerExecution == nil {
		handlerExecution = &HandlerExecutionOptions{
			Timeout:     5 * time.Second,
			Async:       true,
			ErrorPolicy: "continue",
		}
	} else {
		if handlerExecution.Timeout == 0 {
			handlerExecution.Timeout = 5 * time.Second
		}
		if handlerExecution.ErrorPolicy == "" {
			handlerExecution.ErrorPolicy = "continue"
		}
	}

	// 创建 MultiConfig 实例
	cfg := &MultiConfig{
		sources:             sources,
		multiStorage:        multiStorage,
		logger:              log,
		handlerExecution:    handlerExecution,
		onKeyChangeHandlers: make(map[string][]func(storage.Storage) error),
	}

	// 设置每个 Provider 的变更监听
	for i, source := range cfg.sources {
		sourceIndex := i // 捕获循环变量
		source.provider.OnChange(func(newData []byte) error {
			return cfg.handleSourceChange(sourceIndex, newData)
		})
	}

	return cfg, nil
}

// handleSourceChange 处理某个配置源的数据变更
func (c *MultiConfig) handleSourceChange(sourceIndex int, newData []byte) error {
	if sourceIndex < 0 || sourceIndex >= len(c.sources) {
		return fmt.Errorf("invalid source index: %d", sourceIndex)
	}

	source := &c.sources[sourceIndex]

	// 创建旧的合并存储状态的快照，用于变更检测
	// 这里我们重新创建一个 MultiStorage 来保存旧状态
	oldStorages := make([]storage.Storage, len(c.sources))
	for i, s := range c.sources {
		oldStorages[i] = s.storage
	}
	oldMergedStorage := storage.NewMultiStorage(oldStorages)

	// 重新解码数据
	newStorage, err := source.decoder.Decode(newData)
	if err != nil {
		return fmt.Errorf("failed to decode new data from source %d: %w", sourceIndex, err)
	}

	// 用 ValidateStorage 包装新的 storage 以提供自动校验功能
	newStorage = storage.NewValidateStorage(newStorage)

	// 更新存储
	source.storage = newStorage
	changed := c.multiStorage.UpdateStorage(sourceIndex, newStorage)

	if changed {
		// 新的合并存储就是当前的 multiStorage
		newMergedStorage := c.multiStorage

		// 检查并触发变更监听器（统一处理根配置和特定key）
		for key, handlers := range c.onKeyChangeHandlers {
			// 统一使用 isKeyChanged 检查，空字符串key会让Storage.Sub("")返回自己
			if c.isKeyChanged(oldMergedStorage, newMergedStorage, key) {
				// 统一使用 Sub 方法获取目标存储，Sub("")会返回自身
				targetStorage := newMergedStorage.Sub(key)

				// 执行 handlers
				c.executeHandlers(key, handlers, targetStorage)
			}
		}
	}

	return nil
}

// isKeyChanged 检查指定 key 的数据是否发生变更
func (c *MultiConfig) isKeyChanged(oldStorage, newStorage storage.Storage, key string) bool {
	oldSubStorage := oldStorage.Sub(key)
	newSubStorage := newStorage.Sub(key)

	// 使用 Storage 的 Equals 方法进行比较，各实现可以根据自身特点优化
	return !oldSubStorage.Equals(newSubStorage)
}

// executeHandlers 执行 handler 列表，支持异步、超时和错误处理
func (c *MultiConfig) executeHandlers(key string, handlers []func(storage.Storage) error, targetStorage storage.Storage) {
	if len(handlers) == 0 {
		return
	}

	if c.handlerExecution.Async {
		// 异步执行：每个 handler 在独立的 goroutine 中运行
		var wg sync.WaitGroup
		for i, handler := range handlers {
			wg.Add(1)
			go func(idx int, h func(storage.Storage) error) {
				defer wg.Done()
				c.executeHandler(key, idx, h, targetStorage)
			}(i, handler)
		}
		wg.Wait()
	} else {
		// 同步执行：顺序执行每个 handler
		for i, handler := range handlers {
			handlerFailed := c.executeHandler(key, i, handler, targetStorage)
			if c.handlerExecution.ErrorPolicy == "stop" && handlerFailed {
				if c.logger != nil {
					c.logger.Warn("handler execution stopped due to error policy",
						"key", key,
						"stoppedAtIndex", i,
						"remainingHandlers", len(handlers)-i-1)
				}
				break
			}
		}
	}
}

// executeHandler 执行单个 handler，带有超时控制和日志记录
func (c *MultiConfig) executeHandler(key string, index int, handler func(storage.Storage) error, targetStorage storage.Storage) bool {
	ctx, cancel := context.WithTimeout(context.Background(), c.handlerExecution.Timeout)
	defer cancel()

	resultChan := make(chan error, 1)
	start := time.Now()

	go func() {
		resultChan <- handler(targetStorage)
	}()

	select {
	case err := <-resultChan:
		duration := time.Since(start)
		if c.logger != nil {
			if err != nil {
				c.logger.Error("onChange handler failed",
					"key", key,
					"index", index,
					"duration", duration,
					"error", err)
				return true
			} else {
				c.logger.Info("onChange handler succeeded",
					"key", key,
					"index", index,
					"duration", duration)
				return false
			}
		}
		return err != nil
	case <-ctx.Done():
		duration := time.Since(start)
		if c.logger != nil {
			c.logger.Error("onChange handler timeout",
				"key", key,
				"index", index,
				"duration", duration,
				"timeout", c.handlerExecution.Timeout,
				"error", "handler execution timeout")
		}
		return true
	}
}

// Sub 获取子配置对象
func (c *MultiConfig) Sub(key string) Config {
	if key == "" {
		return c
	}

	root := c.getRoot()
	var fullPrefix string
	if c.parent != nil {
		fullPrefix = c.prefix + "." + key
	} else {
		fullPrefix = key
	}

	return &MultiConfig{
		parent: root,
		prefix: fullPrefix,
	}
}

// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
func (c *MultiConfig) ConvertTo(object any) error {
	if c.parent == nil {
		// 根配置直接使用 MultiStorage
		return c.multiStorage.ConvertTo(object)
	}

	// 子配置从父配置获取对应的子存储
	subStorage := c.parent.multiStorage.Sub(c.prefix)
	return subStorage.ConvertTo(object)
}

// SetLogger 设置日志记录器（只有根配置才能设置）
func (c *MultiConfig) SetLogger(logger logger.Logger) {
	root := c.getRoot()
	root.logger = logger
}

// OnChange 监听配置变更
func (c *MultiConfig) OnChange(fn func(storage.Storage) error) {
	if c.parent != nil {
		// 子配置：重定向到根配置的 OnKeyChange
		root := c.getRoot()
		root.OnKeyChange(c.prefix, fn)
	} else {
		// 根配置：使用空字符串作为根配置变更的特殊key
		c.OnKeyChange("", fn)
	}
}

// OnKeyChange 监听指定键的配置变更
func (c *MultiConfig) OnKeyChange(key string, fn func(storage.Storage) error) {
	root := c.getRoot()

	if root.onKeyChangeHandlers == nil {
		root.onKeyChangeHandlers = make(map[string][]func(storage.Storage) error)
	}

	// 所有 key 变更监听器都注册到根配置上
	root.onKeyChangeHandlers[key] = append(root.onKeyChangeHandlers[key], fn)
}

// Watch 启动配置变更监听
func (c *MultiConfig) Watch() error {
	root := c.getRoot()

	// 启动所有 Provider 的监听
	for i, source := range root.sources {
		if err := source.provider.Watch(); err != nil {
			return fmt.Errorf("failed to start watching source %d: %w", i, err)
		}

		// 主动检查一次配置变更，防止在初始化和 Watch 之间丢失变更
		newData, loadErr := source.provider.Load()
		if loadErr == nil {
			// 触发变更检查和处理
			root.handleSourceChange(i, newData)
		}
		// 即使 Load 失败也不影响 Watch 的成功
	}

	return nil
}

// getRoot 获取根配置对象
func (c *MultiConfig) getRoot() *MultiConfig {
	root := c
	for root.parent != nil {
		root = root.parent
	}
	return root
}

// Close 关闭配置对象，释放相关资源
func (c *MultiConfig) Close() error {
	root := c.getRoot()

	root.closeMu.Lock()
	defer root.closeMu.Unlock()

	if root.closed {
		return root.closeResult
	}

	root.closed = true

	// 关闭所有 Provider
	var lastErr error
	for i, source := range root.sources {
		if err := source.provider.Close(); err != nil {
			if root.logger != nil {
				root.logger.Error("failed to close provider", "index", i, "error", err)
			}
			lastErr = err // 记录最后一个错误
		}
	}

	root.closeResult = lastErr
	return root.closeResult
}
