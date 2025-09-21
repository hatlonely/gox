package decoder

import (
	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

func init() {
	ref.MustRegisterT[EnvDecoder](NewEnvDecoder)
	ref.MustRegisterT[CmdDecoder](NewCmdDecoder)
	ref.MustRegisterT[JsonDecoder](NewJsonDecoderWithOptions)
	ref.MustRegisterT[YamlDecoder](NewYamlDecoderWithOptions)
	ref.MustRegisterT[TomlDecoder](NewTomlDecoderWithOptions)
	ref.MustRegisterT[IniDecoder](NewIniDecoderWithOptions)

	ref.MustRegisterT[*EnvDecoder](NewEnvDecoder)
	ref.MustRegisterT[*CmdDecoder](NewCmdDecoder)
	ref.MustRegisterT[*JsonDecoder](NewJsonDecoderWithOptions)
	ref.MustRegisterT[*YamlDecoder](NewYamlDecoderWithOptions)
	ref.MustRegisterT[*TomlDecoder](NewTomlDecoderWithOptions)
	ref.MustRegisterT[*IniDecoder](NewIniDecoderWithOptions)
}

// Decoder 配置数据编解码器接口
// 负责将原始数据和存储对象之间进行转换
type Decoder interface {
	// Decode 将原始数据解码为存储对象
	Decode(data []byte) (storage storage.Storage, err error)
	// Encode 将存储对象编码为原始数据
	Encode(storage storage.Storage) (data []byte, err error)
}

func NewDecoderWithOptions(options *ref.TypeOptions) (Decoder, error) {
	decoder, err := ref.New(options.Namespace, options.Type, options.Options)
	if err != nil {
		return nil, errors.WithMessage(err, "refx.NewT failed")
	}
	if decoder == nil {
		return nil, errors.New("decoder is nil")
	}
	if _, ok := decoder.(Decoder); !ok {
		return nil, errors.New("decoder is not a Decoder")
	}

	return decoder.(Decoder), nil
}
