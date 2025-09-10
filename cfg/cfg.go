package cfg

import (
	"github.com/hatlonely/gox/refx"
)

type Provider interface {
	Read() (data []byte, err error)
	OnChange(fn func(data []byte) error)
}

type Storage interface {
	Sub(key string) Storage
	ToStruct(object any) error
}

type Decoder interface {
	Decode(data []byte) (storage Storage, err error)
	Encode(storage Storage) (data []byte, err error)
}

type Options struct {
	Provider refx.TypeOptions
	Decoder  refx.TypeOptions
}

type Config struct {
	provider Provider
	storage  Storage
	decoder  Decoder

	parent *Config
	key    string
}

func NewConfigWithOptions(options *Options) (*Config, error) {
	return nil, nil
}

func (c *Config) Sub(key string) *Config {
	return nil
}

func (c *Config) ToStruct(object any) error {
	return nil
}

func (c *Config) OnChange(fn func(*Config) error) {

}

func (c *Config) OnKeyChange(key string, fn func(*Config) error) {

}
