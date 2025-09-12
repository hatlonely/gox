package decoder

import (
	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/refx"
)

func init() {
	refx.RegisterT[EnvDecoder](NewEnvDecoderWithOptions)
	refx.RegisterT[JsonDecoder](NewJsonDecoderWithOptions)
	refx.RegisterT[YamlDecoder](NewYamlDecoderWithOptions)
	refx.RegisterT[TomlDecoder](NewTomlDecoderWithOptions)
	refx.RegisterT[IniDecoder](NewIniDecoderWithOptions)
}

// Decoder 配置数据编解码器接口
// 负责将原始数据和存储对象之间进行转换
type Decoder interface {
	// Decode 将原始数据解码为存储对象
	Decode(data []byte) (storage storage.Storage, err error)
	// Encode 将存储对象编码为原始数据
	Encode(storage storage.Storage) (data []byte, err error)
}
