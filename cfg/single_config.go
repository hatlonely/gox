package cfg

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hatlonely/gox/cfg/decoder"
	"github.com/hatlonely/gox/cfg/provider"
	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/refx"
)

// HandlerExecutionOptions onChange handler 执行配置
type HandlerExecutionOptions struct {
	// 超时时长，默认 5 秒
	Timeout time.Duration `cfg:"timeout"`
	// 是否异步执行，默认 true
	Async bool `cfg:"async"`
	// 错误处理策略："continue" 继续执行其他 handler，"stop" 停止执行
	ErrorPolicy string `cfg:"errorPolicy"`
}

// Options 配置类初始化选项
type Options struct {
	Provider         refx.TypeOptions         `cfg:"provider"`
	Decoder          refx.TypeOptions         `cfg:"decoder"`
	Logger           *log.Options             `cfg:"logger"`
	HandlerExecution *HandlerExecutionOptions `cfg:"handlerExecution"`
}

// SingleConfig 配置管理器
// 提供配置数据的统一访问入口和变更监听功能
type SingleConfig struct {
	provider         provider.Provider
	storage          storage.Storage
	decoder          decoder.Decoder
	logger           log.Logger               // 可选的日志记录器
	handlerExecution *HandlerExecutionOptions // handler 执行配置

	parent *SingleConfig
	prefix string

	// 只有根配置才使用这些字段
	// 统一的变更处理器映射，使用空字符串作为根配置变更的特殊key
	onKeyChangeHandlers map[string][]func(*SingleConfig) error

	// Close 状态管理（只有根配置使用）
	closeMu     sync.Mutex
	closed      bool
	closeResult error
}

// NewConfigWithOptions 根据选项创建配置对象
func NewConfigWithOptions(options *Options) (*SingleConfig, error) {
	if options == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	// 创建 Provider 实例
	providerObj, err := refx.New(options.Provider.Namespace, options.Provider.Type, options.Provider.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	prov, ok := providerObj.(provider.Provider)
	if !ok {
		return nil, fmt.Errorf("provider object does not implement Provider interface")
	}

	// 创建 Decoder 实例
	decoderObj, err := refx.New(options.Decoder.Namespace, options.Decoder.Type, options.Decoder.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	dec, ok := decoderObj.(decoder.Decoder)
	if !ok {
		return nil, fmt.Errorf("decoder object does not implement Decoder interface")
	}

	// 从 Provider 加载数据
	data, err := prov.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load data from provider: %w", err)
	}

	// 用 Decoder 解码数据为 Storage
	stor, err := dec.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	// 创建或使用默认 Logger
	var logger log.Logger
	if options.Logger != nil {
		// 使用提供的日志配置创建 Logger
		var err error
		logger, err = log.NewLogWithOptions(options.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	} else {
		// 创建默认的终端输出 Logger
		var err error
		logger, err = log.NewLogWithOptions(&log.Options{
			Level:  "info",
			Format: "text",
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
		// 设置默认值
		if handlerExecution.Timeout == 0 {
			handlerExecution.Timeout = 5 * time.Second
		}
		if handlerExecution.ErrorPolicy == "" {
			handlerExecution.ErrorPolicy = "continue"
		}
	}

	// 创建 SingleConfig 实例
	cfg := &SingleConfig{
		provider:            prov,
		storage:             stor,
		decoder:             dec,
		logger:              logger,
		handlerExecution:    handlerExecution,
		onKeyChangeHandlers: make(map[string][]func(*SingleConfig) error),
	}

	// 设置 Provider 的变更监听
	prov.OnChange(func(newData []byte) error {
		return cfg.handleProviderChange(newData)
	})

	return cfg, nil
}

// NewConfig 简单构造方法，从文件中加载配置
// 根据文件后缀自动选择对应的解码器：
//
//	.json/.json5 -> JsonDecoder
//	.yaml/.yml -> YamlDecoder
//	.toml -> TomlDecoder
//	.ini -> IniDecoder
//	.env -> EnvDecoder
func NewConfig(filename string) (*SingleConfig, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	// 根据文件扩展名确定解码器类型
	ext := strings.ToLower(filepath.Ext(filename))
	var decoderType string
	var decoderOptions any

	switch ext {
	case ".json", ".json5":
		decoderType = "JsonDecoder"
		decoderOptions = &decoder.JsonDecoderOptions{UseJSON5: ext == ".json5"}
	case ".yaml", ".yml":
		decoderType = "YamlDecoder"
		decoderOptions = &decoder.YamlDecoderOptions{Indent: 2}
	case ".toml":
		decoderType = "TomlDecoder"
		decoderOptions = &decoder.TomlDecoderOptions{Indent: "  "}
	case ".ini":
		decoderType = "IniDecoder"
		decoderOptions = &decoder.IniDecoderOptions{
			AllowEmptyValues: true,
			AllowBoolKeys:    true,
			AllowShadows:     true,
		}
	case ".env":
		decoderType = "EnvDecoder"
		decoderOptions = nil
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}

	// 构建选项
	options := &Options{
		Provider: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/provider",
			Type:      "FileProvider",
			Options: &provider.FileProviderOptions{
				FilePath: filename,
			},
		},
		Decoder: refx.TypeOptions{
			Namespace: "github.com/hatlonely/gox/cfg/decoder",
			Type:      decoderType,
			Options:   decoderOptions,
		},
	}

	return NewConfigWithOptions(options)
}

// handleProviderChange 处理 Provider 数据变更
func (c *SingleConfig) handleProviderChange(newData []byte) error {
	// 保存旧的 storage
	oldStorage := c.storage

	// 重新解码数据
	newStorage, err := c.decoder.Decode(newData)
	if err != nil {
		return fmt.Errorf("failed to decode new data: %w", err)
	}
	c.storage = newStorage

	// 检查并触发变更监听器（统一处理根配置和特定key）
	for key, handlers := range c.onKeyChangeHandlers {
		// 统一使用 isKeyChanged 检查，空字符串key会让Storage.Sub("")返回自己
		if c.isKeyChanged(oldStorage, newStorage, key) {
			// 统一使用 Sub 方法获取目标配置，Sub("")会返回自身
			targetConfig := c.Sub(key)

			// 执行 handlers，直接使用原始的 key
			c.executeHandlers(key, handlers, targetConfig)
		}
	}

	return nil
}

// executeHandlers 执行 handler 列表，支持异步、超时和错误处理
func (c *SingleConfig) executeHandlers(key string, handlers []func(*SingleConfig) error, config *SingleConfig) {
	if len(handlers) == 0 {
		return
	}

	if c.handlerExecution.Async {
		// 异步执行：每个 handler 在独立的 goroutine 中运行
		var wg sync.WaitGroup
		for i, handler := range handlers {
			wg.Add(1)
			go func(idx int, h func(*SingleConfig) error) {
				defer wg.Done()
				c.executeHandler(key, idx, h, config)
			}(i, handler)
		}
		wg.Wait()
	} else {
		// 同步执行：顺序执行每个 handler
		for i, handler := range handlers {
			handlerFailed := c.executeHandler(key, i, handler, config)
			if c.handlerExecution.ErrorPolicy == "stop" && handlerFailed {
				// 如果错误策略是 stop 且当前 handler 失败，停止执行后续 handler
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
// 返回 true 如果 handler 失败（错误或超时），false 如果成功
func (c *SingleConfig) executeHandler(key string, index int, handler func(*SingleConfig) error, config *SingleConfig) bool {
	// 创建带超时的 context
	ctx, cancel := context.WithTimeout(context.Background(), c.handlerExecution.Timeout)
	defer cancel()

	// 在 goroutine 中执行 handler，支持超时取消
	resultChan := make(chan error, 1)
	start := time.Now()

	go func() {
		resultChan <- handler(config)
	}()

	// 等待结果或超时
	select {
	case err := <-resultChan:
		// handler 正常完成
		duration := time.Since(start)
		if c.logger != nil {
			if err != nil {
				c.logger.Error("onChange handler failed",
					"key", key,
					"index", index,
					"duration", duration,
					"error", err)
				return true // handler 失败
			} else {
				c.logger.Info("onChange handler succeeded",
					"key", key,
					"index", index,
					"duration", duration)
				return false // handler 成功
			}
		}
		return err != nil
	case <-ctx.Done():
		// handler 超时
		duration := time.Since(start)
		if c.logger != nil {
			c.logger.Error("onChange handler timeout",
				"key", key,
				"index", index,
				"duration", duration,
				"timeout", c.handlerExecution.Timeout,
				"error", "handler execution timeout")
		}
		return true // 超时视为失败
	}
}

// isKeyChanged 检查指定 key 的数据是否发生变更
func (c *SingleConfig) isKeyChanged(oldStorage, newStorage storage.Storage, key string) bool {
	oldSubStorage := oldStorage.Sub(key)
	newSubStorage := newStorage.Sub(key)

	// 使用 Storage 的 Equals 方法进行比较，各实现可以根据自身特点优化
	return !oldSubStorage.Equals(newSubStorage)
}

// Sub 获取子配置对象
// 优化后的实现：所有子配置共享同一个根配置，只存储父配置引用和前缀
// 当key为空字符串时，返回自身（与Storage.Sub("")的行为一致）
func (c *SingleConfig) Sub(key string) *SingleConfig {
	if key == "" {
		return c
	}

	root := c.getRoot()
	var fullPrefix string
	if c.parent != nil {
		// 如果当前配置是子配置，构建完整的前缀路径
		fullPrefix = c.getFullKey() + "." + key
	} else {
		// 如果当前配置是根配置，直接使用key作为前缀
		fullPrefix = key
	}

	return &SingleConfig{
		parent: root,
		prefix: fullPrefix,
	}
}

// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
func (c *SingleConfig) ConvertTo(object any) error {
	if c.parent == nil {
		// 根配置直接使用自己的存储
		return c.storage.ConvertTo(object)
	}

	// 子配置从父配置获取对应的子存储
	subStorage := c.parent.storage.Sub(c.prefix)
	return subStorage.ConvertTo(object)
}

// SetLogger 设置日志记录器（只有根配置才能设置）
func (c *SingleConfig) SetLogger(logger log.Logger) {
	root := c.getRoot()
	root.logger = logger
}

// OnChange 监听配置变更
func (c *SingleConfig) OnChange(fn func(*SingleConfig) error) {
	if c.parent != nil {
		// 子配置：重定向到根配置的 OnKeyChange
		root := c.getRoot()
		fullKey := c.getFullKey()
		root.OnKeyChange(fullKey, fn)
	} else {
		// 根配置：使用空字符串作为根配置变更的特殊key
		c.OnKeyChange("", fn)
	}
}

// OnKeyChange 监听指定键的配置变更
func (c *SingleConfig) OnKeyChange(key string, fn func(*SingleConfig) error) {
	root := c.getRoot()

	if root.onKeyChangeHandlers == nil {
		root.onKeyChangeHandlers = make(map[string][]func(*SingleConfig) error)
	}

	// 所有 key 变更监听器都注册到根配置上
	root.onKeyChangeHandlers[key] = append(root.onKeyChangeHandlers[key], fn)
}

// Watch 启动配置变更监听
// 只有调用此方法后，OnChange 和 OnKeyChange 注册的回调函数才会被触发
// 对于不支持监听的 Provider，此方法静默处理不返回错误
// 为了防止在 NewConfig 和 Watch 之间丢失配置变更，会主动检查一次配置
func (c *SingleConfig) Watch() error {
	root := c.getRoot()
	if root.provider != nil {
		// 先启动 Provider 的监听
		err := root.provider.Watch()
		if err != nil {
			return err
		}

		// 主动检查一次配置变更，防止在初始化和 Watch 之间丢失变更
		newData, loadErr := root.provider.Load()
		if loadErr == nil {
			// 触发变更检查和处理，由于 handleProviderChange 内部有变更检测逻辑，
			// 如果没有实际变更就不会触发 handler
			root.handleProviderChange(newData)
		}
		// 即使 Load 失败也不影响 Watch 的成功，因为可能是网络问题等临时错误
	}
	return nil
}

// getRoot 获取根配置对象
func (c *SingleConfig) getRoot() *SingleConfig {
	root := c
	for root.parent != nil {
		root = root.parent
	}
	return root
}

// getFullKey 获取当前配置对象的完整路径
func (c *SingleConfig) getFullKey() string {
	if c.parent == nil {
		return ""
	}

	// 优化后的实现：直接返回存储的完整前缀
	return c.prefix
}

// Close 关闭配置对象，释放相关资源
// 只有根配置对象才能执行关闭操作，子配置对象会将关闭请求转发到根配置
// 多次调用只会执行一次，后续调用直接返回第一次调用的结果
func (c *SingleConfig) Close() error {
	root := c.getRoot()

	// 使用互斥锁确保线程安全
	root.closeMu.Lock()
	defer root.closeMu.Unlock()

	// 如果已经关闭过，直接返回之前的结果
	if root.closed {
		return root.closeResult
	}

	// 标记为已关闭
	root.closed = true

	// 执行关闭操作
	if root.provider != nil {
		root.closeResult = root.provider.Close()
	} else {
		root.closeResult = nil
	}

	return root.closeResult
}
