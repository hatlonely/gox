package cfg

import (
	"github.com/hatlonely/gox/refx"
)

// Provider 配置数据提供者接口
// 负责读取配置数据和监听配置变更
type Provider interface {
	// Read 读取配置数据
	Read() (data []byte, err error)
	// OnChange 监听配置数据变更
	OnChange(fn func(data []byte) error)
}

// Storage 配置数据存储接口
// 提供层级化配置访问和结构体绑定功能
type Storage interface {
	// Sub 获取子配置存储对象
	// key 可以包含点号（.）表示多级嵌套，[]表示数组索引
	// 例如 "database.connections[0].host"
	Sub(key string) Storage

	// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
	ConvertTo(object any) error
}

// Decoder 配置数据编解码器接口
// 负责将原始数据和存储对象之间进行转换
type Decoder interface {
	// Decode 将原始数据解码为存储对象
	Decode(data []byte) (storage Storage, err error)
	// Encode 将存储对象编码为原始数据
	Encode(storage Storage) (data []byte, err error)
}

// Options 配置类初始化选项
type Options struct {
	Provider refx.TypeOptions
	Decoder  refx.TypeOptions
}

// Config 配置管理器
// 提供配置数据的统一访问入口和变更监听功能
type Config struct {
	provider Provider
	storage  Storage
	decoder  Decoder

	parent *Config
	key    string
}

// NewConfigWithOptions 根据选项创建配置对象
func NewConfigWithOptions(options *Options) (*Config, error) {
	return nil, nil
}

// Sub 获取子配置对象
func (c *Config) Sub(key string) *Config {
	return nil
}

// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
func (c *Config) ConvertTo(object any) error {
	return nil
}

// OnChange 监听配置变更
func (c *Config) OnChange(fn func(*Config) error) {

}

// OnKeyChange 监听指定键的配置变更
func (c *Config) OnKeyChange(key string, fn func(*Config) error) {

}
