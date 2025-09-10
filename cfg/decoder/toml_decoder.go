package decoder

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/hatlonely/gox/cfg/storage"
)

// TomlDecoder TOML格式编解码器
// 支持标准TOML格式，包含注释支持
type TomlDecoder struct {
	// Indent TOML缩进空格数，用于格式化输出
	Indent string
}

// NewTomlDecoder 创建新的TOML解码器
func NewTomlDecoder() *TomlDecoder {
	return &TomlDecoder{
		Indent: "  ", // 默认2个空格缩进
	}
}

// NewTomlDecoderWithIndent 创建指定缩进的TOML解码器
func NewTomlDecoderWithIndent(indent string) *TomlDecoder {
	return &TomlDecoder{
		Indent: indent,
	}
}

// Decode 将TOML数据解码为Storage对象
func (t *TomlDecoder) Decode(data []byte) (storage.Storage, error) {
	var result interface{}
	
	// 使用 toml.Decode 解析到 map[string]interface{}
	var parsed map[string]interface{}
	if err := toml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode TOML: %w", err)
	}
	
	result = parsed

	// 创建MapStorage包装解析结果
	return storage.NewMapStorage(result), nil
}

// Encode 将Storage对象编码为TOML数据
func (t *TomlDecoder) Encode(s storage.Storage) ([]byte, error) {
	var data interface{}

	// 尝试直接获取MapStorage的内部数据
	if mapStorage, ok := s.(*storage.MapStorage); ok {
		data = mapStorage.Data()
	} else {
		// 如果不是MapStorage，尝试转换为通用interface{}
		if err := s.ConvertTo(&data); err != nil {
			return nil, fmt.Errorf("failed to convert storage to data: %w", err)
		}
	}

	// 使用buffer来控制输出格式
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	
	// 设置缩进（如果支持的话）
	encoder.Indent = t.Indent
	
	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to encode TOML: %w", err)
	}

	return buf.Bytes(), nil
}