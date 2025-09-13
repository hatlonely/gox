package decoder

import (
	"bufio"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hatlonely/gox/cfg/storage"
)

// CmdDecoderOptions 命令行参数解码器配置选项
type CmdDecoderOptions struct {
	// 保留结构体以兼容现有代码，但不再使用字段
}

// CmdDecoder 命令行参数格式编解码器
// 支持命令行参数格式，使用FlatStorage进行智能字段匹配
// 使用固定的默认配置：分隔符"-"，数组格式"-%d"，支持注释和空行
type CmdDecoder struct {}

// NewCmdDecoder 创建新的命令行参数解码器，使用默认配置
func NewCmdDecoder() *CmdDecoder {
	return &CmdDecoder{}
}

// NewCmdDecoderWithOptions 使用选项创建命令行参数解码器
// 为了兼容性保留，忽略 options 参数，始终使用默认配置
func NewCmdDecoderWithOptions(options *CmdDecoderOptions) *CmdDecoder {
	return NewCmdDecoder()
}

// Decode 将命令行参数数据解码为FlatStorage对象
func (c *CmdDecoder) Decode(data []byte) (storage.Storage, error) {
	result := make(map[string]interface{})
	
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// 跳过空行
		if line == "" {
			continue
		}
		
		// 跳过注释行
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		
		// 解析键值对
		if err := c.parseLine(line, result, lineNum); err != nil {
			return nil, err
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan cmd data: %w", err)
	}
	
	// 创建FlatStorage，使用固定的默认配置
	return storage.NewFlatStorageWithOptions(result, "-", "-%d"), nil
}

// parseLine 解析单行数据
func (c *CmdDecoder) parseLine(line string, result map[string]interface{}, lineNum int) error {
	// 跳过空行和注释
	if line == "" {
		return nil
	}
	
	// 查找等号分隔符
	equalIndex := strings.Index(line, "=")
	if equalIndex == -1 {
		return fmt.Errorf("invalid format at line %d: missing '=' separator", lineNum)
	}
	
	key := strings.TrimSpace(line[:equalIndex])
	value := line[equalIndex+1:]
	
	// 验证键名
	if key == "" {
		return fmt.Errorf("invalid format at line %d: empty key", lineNum)
	}
	
	// 解析值
	parsedValue, err := c.parseValue(value)
	if err != nil {
		return fmt.Errorf("invalid value at line %d: %w", lineNum, err)
	}
	
	result[key] = parsedValue
	return nil
}

// parseValue 解析值，支持字符串、数字、布尔值
func (c *CmdDecoder) parseValue(value string) (interface{}, error) {
	value = strings.TrimSpace(value)
	
	// 处理引号包围的字符串
	if len(value) >= 2 {
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		   (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			// 去除引号
			unquoted := value[1 : len(value)-1]
			// 处理转义字符
			return c.unescapeString(unquoted), nil
		}
	}
	
	// 尝试解析为布尔值
	if boolVal, err := strconv.ParseBool(value); err == nil {
		return boolVal, nil
	}
	
	// 尝试解析为整数
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intVal, nil
	}
	
	// 尝试解析为浮点数
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal, nil
	}
	
	// 默认作为字符串处理
	return value, nil
}

// unescapeString 处理字符串中的转义字符
func (c *CmdDecoder) unescapeString(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\\"", "\"")
	s = strings.ReplaceAll(s, "\\'", "'")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

// Encode 将Storage对象编码为命令行参数数据
func (c *CmdDecoder) Encode(s storage.Storage) ([]byte, error) {
	var data map[string]interface{}
	
	// 尝试获取FlatStorage的数据
	if flatStorage, ok := s.(*storage.FlatStorage); ok {
		data = flatStorage.Data()
	} else {
		// 如果不是FlatStorage，尝试转换
		if err := s.ConvertTo(&data); err != nil {
			return nil, fmt.Errorf("failed to convert storage to map: %w", err)
		}
	}
	
	var lines []string
	var keys []string
	
	// 收集所有键并排序
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	
	// 生成键值对行
	for _, key := range keys {
		value := data[key]
		line := fmt.Sprintf("%s=%s", key, c.formatValue(value))
		lines = append(lines, line)
	}
	
	return []byte(strings.Join(lines, "\n")), nil
}

// formatValue 格式化值为命令行参数格式
func (c *CmdDecoder) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// 如果字符串包含特殊字符，需要加引号
		if c.needsQuoting(v) {
			return fmt.Sprintf("\"%s\"", c.escapeString(v))
		}
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	default:
		// 其他类型转为字符串
		str := fmt.Sprintf("%v", v)
		if c.needsQuoting(str) {
			return fmt.Sprintf("\"%s\"", c.escapeString(str))
		}
		return str
	}
}

// needsQuoting 判断字符串是否需要加引号
func (c *CmdDecoder) needsQuoting(s string) bool {
	// 空字符串需要引号
	if s == "" {
		return true
	}
	
	// 包含空格、特殊字符的字符串需要引号
	return strings.ContainsAny(s, " \"'#\\=\n\t\r")
}

// escapeString 转义字符串中的特殊字符
func (c *CmdDecoder) escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}