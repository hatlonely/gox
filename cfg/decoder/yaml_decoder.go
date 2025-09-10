package decoder

import (
	"fmt"

	"github.com/hatlonely/gox/cfg/storage"
	"gopkg.in/yaml.v3"
)

// YamlDecoder YAML格式编解码器
// 支持标准YAML格式，自带注释支持
type YamlDecoder struct {
	// Indent YAML缩进空格数，默认为2
	Indent int
}

// NewYamlDecoder 创建新的YAML解码器
func NewYamlDecoder() *YamlDecoder {
	return &YamlDecoder{
		Indent: 2, // 默认2个空格缩进
	}
}

// NewYamlDecoderWithIndent 创建指定缩进的YAML解码器
func NewYamlDecoderWithIndent(indent int) *YamlDecoder {
	return &YamlDecoder{
		Indent: indent,
	}
}

// Decode 将YAML数据解码为Storage对象
func (y *YamlDecoder) Decode(data []byte) (storage.Storage, error) {
	var result interface{}
	
	err := yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	// 创建MapStorage包装解析结果
	return storage.NewMapStorage(result), nil
}

// Encode 将Storage对象编码为YAML数据
func (y *YamlDecoder) Encode(s storage.Storage) ([]byte, error) {
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

	// 创建编码器并设置缩进
	var encoder yaml.Node
	err := encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode YAML node: %w", err)
	}

	// 使用标准的yaml.Marshal进行编码
	result, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode YAML: %w", err)
	}

	return result, nil
}