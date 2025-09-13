package cfg

import (
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

// Options 配置类初始化选项
type Options struct {
	Provider refx.TypeOptions
	Decoder  refx.TypeOptions
	Logger   *log.Options // 可选的日志配置
}

// Config 配置管理器
// 提供配置数据的统一访问入口和变更监听功能
type Config struct {
	provider provider.Provider
	storage  storage.Storage
	decoder  decoder.Decoder
	logger   log.Logger // 可选的日志记录器

	parent *Config
	key    string

	// 只有根配置才使用这些字段
	onChangeHandlers    []func(*Config) error
	onKeyChangeHandlers map[string][]func(*Config) error

	// Close 状态管理（只有根配置使用）
	closeMu     sync.Mutex
	closed      bool
	closeResult error
}

// NewConfigWithOptions 根据选项创建配置对象
func NewConfigWithOptions(options *Options) (*Config, error) {
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

	// 创建 Config 实例
	cfg := &Config{
		provider:            prov,
		storage:             stor,
		decoder:             dec,
		logger:              logger,
		onKeyChangeHandlers: make(map[string][]func(*Config) error),
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
func NewConfig(filename string) (*Config, error) {
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
		decoderOptions = &decoder.EnvDecoderOptions{
			Separator:     "_",
			ArrayFormat:   "_%d",
			AllowComments: true,
		}
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
func (c *Config) handleProviderChange(newData []byte) error {
	// 保存旧的 storage
	oldStorage := c.storage

	// 重新解码数据
	newStorage, err := c.decoder.Decode(newData)
	if err != nil {
		return fmt.Errorf("failed to decode new data: %w", err)
	}
	c.storage = newStorage

	// 触发根配置的全局变更监听器
	for _, handler := range c.onChangeHandlers {
		start := time.Now()
		err := handler(c)
		duration := time.Since(start)

		if c.logger != nil {
			if err != nil {
				c.logger.Warn("onChange handler failed", "key", "root", "duration", duration, "error", err)
			} else {
				c.logger.Info("onChange handler succeeded", "key", "root", "duration", duration)
			}
		}
	}

	// 检查并触发特定 key 的变更监听器
	for key, handlers := range c.onKeyChangeHandlers {
		if c.isKeyChanged(oldStorage, newStorage, key) {
			// 创建对应的 Sub Config 对象
			subConfig := &Config{
				provider: c.provider,
				decoder:  c.decoder,
				storage:  newStorage.Sub(key),
				logger:   c.logger,
				parent:   c,
				key:      key,
			}

			// 触发所有注册的监听器
			for _, handler := range handlers {
				start := time.Now()
				err := handler(subConfig)
				duration := time.Since(start)

				if c.logger != nil {
					if err != nil {
						c.logger.Warn("onKeyChange handler failed", "key", key, "duration", duration, "error", err)
					} else {
						c.logger.Info("onKeyChange handler succeeded", "key", key, "duration", duration)
					}
				}
			}
		}
	}

	return nil
}

// isKeyChanged 检查指定 key 的数据是否发生变更
func (c *Config) isKeyChanged(oldStorage, newStorage storage.Storage, key string) bool {
	oldSubStorage := oldStorage.Sub(key)
	newSubStorage := newStorage.Sub(key)

	// 使用 Storage 的 Equals 方法进行比较，各实现可以根据自身特点优化
	return !oldSubStorage.Equals(newSubStorage)
}

// Sub 获取子配置对象
func (c *Config) Sub(key string) *Config {
	root := c.getRoot()
	return &Config{
		provider: root.provider,
		decoder:  root.decoder,
		storage:  c.storage.Sub(key),
		logger:   root.logger,
		parent:   c,
		key:      key,
	}
}

// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
func (c *Config) ConvertTo(object any) error {
	return c.storage.ConvertTo(object)
}

// SetLogger 设置日志记录器（只有根配置才能设置）
func (c *Config) SetLogger(logger log.Logger) {
	root := c.getRoot()
	root.logger = logger
}

// OnChange 监听配置变更
func (c *Config) OnChange(fn func(*Config) error) {
	if c.parent != nil {
		// 子配置：重定向到根配置的 OnKeyChange
		root := c.getRoot()
		fullKey := c.getFullKey()
		root.OnKeyChange(fullKey, fn)
	} else {
		// 根配置：直接添加到全局变更监听器
		c.onChangeHandlers = append(c.onChangeHandlers, fn)
	}
}

// OnKeyChange 监听指定键的配置变更
func (c *Config) OnKeyChange(key string, fn func(*Config) error) {
	root := c.getRoot()

	if root.onKeyChangeHandlers == nil {
		root.onKeyChangeHandlers = make(map[string][]func(*Config) error)
	}

	// 所有 key 变更监听器都注册到根配置上
	root.onKeyChangeHandlers[key] = append(root.onKeyChangeHandlers[key], fn)
}

// getRoot 获取根配置对象
func (c *Config) getRoot() *Config {
	root := c
	for root.parent != nil {
		root = root.parent
	}
	return root
}

// getFullKey 获取当前配置对象的完整路径
func (c *Config) getFullKey() string {
	if c.parent == nil {
		return ""
	}

	keys := []string{c.key}
	current := c.parent

	for current.parent != nil {
		keys = append([]string{current.key}, keys...)
		current = current.parent
	}

	return strings.Join(keys, ".")
}

// Close 关闭配置对象，释放相关资源
// 只有根配置对象才能执行关闭操作，子配置对象会将关闭请求转发到根配置
// 多次调用只会执行一次，后续调用直接返回第一次调用的结果
func (c *Config) Close() error {
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
