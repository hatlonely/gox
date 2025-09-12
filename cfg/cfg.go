package cfg

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hatlonely/gox/cfg/decoder"
	"github.com/hatlonely/gox/cfg/provider"
	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/refx"
)

// Options 配置类初始化选项
type Options struct {
	Provider refx.TypeOptions
	Decoder  refx.TypeOptions
}

// Config 配置管理器
// 提供配置数据的统一访问入口和变更监听功能
type Config struct {
	provider provider.Provider
	storage  storage.Storage
	decoder  decoder.Decoder

	parent *Config
	key    string

	// 只有根配置才使用这些字段
	onChangeHandlers    []func(*Config) error
	onKeyChangeHandlers map[string][]func(*Config) error
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

	// 创建 Config 实例
	cfg := &Config{
		provider:            prov,
		storage:             stor,
		decoder:             dec,
		onKeyChangeHandlers: make(map[string][]func(*Config) error),
	}

	// 设置 Provider 的变更监听
	prov.OnChange(func(newData []byte) error {
		return cfg.handleProviderChange(newData)
	})

	return cfg, nil
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
		if err := handler(c); err != nil {
			// 可以记录日志，但不中断其他处理器
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
				parent:   c,
				key:      key,
			}

			// 触发所有注册的监听器
			for _, handler := range handlers {
				if err := handler(subConfig); err != nil {
					// 可以记录日志，但不中断其他处理器
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

	// 简单的深度比较（这里可以根据需要优化）
	return !reflect.DeepEqual(oldSubStorage, newSubStorage)
}

// Sub 获取子配置对象
func (c *Config) Sub(key string) *Config {
	root := c.getRoot()
	return &Config{
		provider: root.provider,
		decoder:  root.decoder,
		storage:  c.storage.Sub(key),
		parent:   c,
		key:      key,
	}
}

// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
func (c *Config) ConvertTo(object any) error {
	return c.storage.ConvertTo(object)
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
