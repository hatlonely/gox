package decoder

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/hatlonely/gox/cfg/storage"
	"gopkg.in/ini.v1"
)

// IniDecoder INI格式编解码器
// 支持标准INI格式，包含注释和分组支持
type IniDecoder struct {
	// AllowEmptyValues 允许空值
	AllowEmptyValues bool
	// AllowBoolKeys 允许布尔型键（无值的键）
	AllowBoolKeys bool
	// AllowShadows 允许重复键（创建数组）
	AllowShadows bool
}

// NewIniDecoder 创建新的INI解码器
func NewIniDecoder() *IniDecoder {
	return &IniDecoder{
		AllowEmptyValues: true,
		AllowBoolKeys:    true,
		AllowShadows:     true,
	}
}

// NewIniDecoderWithOptions 创建带选项的INI解码器
func NewIniDecoderWithOptions(allowEmptyValues, allowBoolKeys, allowShadows bool) *IniDecoder {
	return &IniDecoder{
		AllowEmptyValues: allowEmptyValues,
		AllowBoolKeys:    allowBoolKeys,
		AllowShadows:     allowShadows,
	}
}

// Decode 将INI数据解码为Storage对象
func (i *IniDecoder) Decode(data []byte) (storage.Storage, error) {
	// 创建INI加载选项
	loadOptions := ini.LoadOptions{
		AllowBooleanKeys:           i.AllowBoolKeys,
		AllowShadows:               i.AllowShadows,
		AllowPythonMultilineValues: true,
		SpaceBeforeInlineComment:   true,
	}

	// 从字节数组加载INI
	cfg, err := ini.LoadSources(loadOptions, data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode INI: %w", err)
	}

	// 转换INI结构为map[string]interface{}
	result := make(map[string]interface{})

	// 处理默认section（没有section头的键值对）
	defaultSection := cfg.Section("")
	if len(defaultSection.Keys()) > 0 {
		for _, key := range defaultSection.Keys() {
			result[key.Name()] = i.parseValue(key)
		}
	}

	// 处理其他sections
	for _, section := range cfg.Sections() {
		if section.Name() == "" {
			continue // 跳过默认section，已经处理过了
		}

		sectionMap := make(map[string]interface{})
		for _, key := range section.Keys() {
			sectionMap[key.Name()] = i.parseValue(key)
		}

		result[section.Name()] = sectionMap
	}

	// 创建MapStorage包装解析结果
	return storage.NewMapStorage(result), nil
}

// parseValue 解析INI键的值，尝试自动类型转换
func (i *IniDecoder) parseValue(key *ini.Key) interface{} {
	value := key.String()

	// 处理重复键（shadows）
	if i.AllowShadows {
		strings := key.StringsWithShadows(",")
		if len(strings) > 1 {
			// 如果有多个值，返回数组
			values := make([]interface{}, len(strings))
			for idx, str := range strings {
				values[idx] = i.parseStringValue(str)
			}
			return values
		}
	}

	// 处理空值：如果允许空值，则返回空字符串；如果是布尔键，则返回true
	if value == "" {
		if i.AllowEmptyValues {
			return ""
		} else if i.AllowBoolKeys {
			return true
		}
	}

	return i.parseStringValue(value)
}

// parseStringValue 解析字符串值，尝试自动类型转换
func (i *IniDecoder) parseStringValue(value string) interface{} {
	// 空字符串
	if value == "" {
		return ""
	}

	// 尝试解析为布尔值
	if strings.ToLower(value) == "true" {
		return true
	}
	if strings.ToLower(value) == "false" {
		return false
	}

	// 尝试解析为整数
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intVal
	}

	// 尝试解析为浮点数
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// 处理用逗号分隔的数组
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		// 只有当所有部分都不包含空格时才认为是数组
		isArray := true
		for _, part := range parts {
			if strings.TrimSpace(part) != part {
				isArray = false
				break
			}
		}
		if isArray && len(parts) > 1 {
			values := make([]interface{}, len(parts))
			for idx, part := range parts {
				values[idx] = i.parseStringValue(strings.TrimSpace(part))
			}
			return values
		}
	}

	// 默认返回字符串
	return value
}

// Encode 将Storage对象编码为INI数据
func (i *IniDecoder) Encode(s storage.Storage) ([]byte, error) {
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

	// 创建新的INI文件
	cfg := ini.Empty()

	// 转换数据到INI格式
	if err := i.encodeToINI(cfg, "", data); err != nil {
		return nil, fmt.Errorf("failed to encode to INI format: %w", err)
	}

	// 写入到buffer
	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("failed to write INI: %w", err)
	}

	return buf.Bytes(), nil
}

// encodeToINI 递归地将数据编码到INI结构中
func (i *IniDecoder) encodeToINI(cfg *ini.File, sectionName string, data interface{}) error {
	switch v := data.(type) {
	case map[string]interface{}:
		var section *ini.Section
		if sectionName == "" {
			section = cfg.Section("")
		} else {
			section = cfg.Section(sectionName)
		}

		for key, value := range v {
			switch val := value.(type) {
			case map[string]interface{}:
				// 嵌套对象作为新section
				newSectionName := key
				if sectionName != "" {
					newSectionName = sectionName + "." + key
				}
				if err := i.encodeToINI(cfg, newSectionName, val); err != nil {
					return err
				}
			case []interface{}:
				// 数组转换为逗号分隔的字符串或多个键
				if i.AllowShadows {
					// 使用shadows支持多个相同键
					for _, item := range val {
						section.NewKey(key, fmt.Sprintf("%v", item))
					}
				} else {
					// 转换为逗号分隔的字符串
					var strs []string
					for _, item := range val {
						strs = append(strs, fmt.Sprintf("%v", item))
					}
					section.NewKey(key, strings.Join(strs, ","))
				}
			default:
				// 基本类型
				section.NewKey(key, fmt.Sprintf("%v", val))
			}
		}
	default:
		return fmt.Errorf("unsupported data type for INI encoding: %T", data)
	}

	return nil
}