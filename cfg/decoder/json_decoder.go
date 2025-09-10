package decoder

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hatlonely/gox/cfg/storage"
)

// JsonDecoder JSON格式编解码器
// 支持标准JSON和JSON5格式（包含注释）
type JsonDecoder struct {
	// UseJSON5 是否使用JSON5解析器（支持注释、尾随逗号等）
	UseJSON5 bool
}

// NewJsonDecoder 创建新的JSON解码器
func NewJsonDecoder() *JsonDecoder {
	return &JsonDecoder{
		UseJSON5: true, // 默认启用JSON5支持
	}
}

// NewJsonDecoderWithOptions 使用选项创建JSON解码器
func NewJsonDecoderWithOptions(useJSON5 bool) *JsonDecoder {
	return &JsonDecoder{
		UseJSON5: useJSON5,
	}
}

// Decode 将JSON数据解码为Storage对象
func (j *JsonDecoder) Decode(data []byte) (storage.Storage, error) {
	var result interface{}
	var err error

	if j.UseJSON5 {
		// 使用自定义JSON5预处理，支持注释和宽松格式
		processedData := j.preprocessJSON5(data)
		err = json.Unmarshal(processedData, &result)
	} else {
		// 使用标准JSON解析器
		err = json.Unmarshal(data, &result)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// 创建MapStorage包装解析结果
	return storage.NewMapStorage(result), nil
}

// preprocessJSON5 预处理JSON5格式，移除注释和处理宽松语法
func (j *JsonDecoder) preprocessJSON5(data []byte) []byte {
	content := string(data)
	
	// 移除单行注释 // 但要保留字符串中的 //
	content = j.removeLineComments(content)
	
	// 移除多行注释 /* */ 但要保留字符串中的注释
	content = j.removeBlockComments(content)
	
	// 处理尾随逗号（移除对象和数组中的尾随逗号）
	content = j.removeTrailingCommas(content)
	
	return []byte(content)
}

// removeLineComments 移除单行注释，但保留字符串内的内容
func (j *JsonDecoder) removeLineComments(content string) string {
	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))
	
	for i, line := range lines {
		result[i] = j.removeSingleLineComment(line)
	}
	
	return strings.Join(result, "\n")
}

// removeSingleLineComment 移除单行中的注释，保留字符串内的 //
func (j *JsonDecoder) removeSingleLineComment(line string) string {
	inString := false
	escaped := false
	
	for i, char := range line {
		if escaped {
			escaped = false
			continue
		}
		
		if char == '\\' {
			escaped = true
			continue
		}
		
		if char == '"' {
			inString = !inString
			continue
		}
		
		// 如果不在字符串内，且遇到 //，则截断
		if !inString && char == '/' && i+1 < len(line) && line[i+1] == '/' {
			return line[:i]
		}
	}
	
	return line
}

// removeBlockComments 移除块注释
func (j *JsonDecoder) removeBlockComments(content string) string {
	inString := false
	escaped := false
	result := strings.Builder{}
	
	runes := []rune(content)
	i := 0
	
	for i < len(runes) {
		char := runes[i]
		
		if escaped {
			result.WriteRune(char)
			escaped = false
			i++
			continue
		}
		
		if char == '\\' {
			result.WriteRune(char)
			escaped = true
			i++
			continue
		}
		
		if char == '"' {
			inString = !inString
			result.WriteRune(char)
			i++
			continue
		}
		
		// 如果不在字符串内，且遇到 /*，则跳过到 */
		if !inString && char == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			i += 2 // 跳过 /*
			// 寻找 */
			for i+1 < len(runes) {
				if runes[i] == '*' && runes[i+1] == '/' {
					i += 2 // 跳过 */
					break
				}
				i++
			}
			continue
		}
		
		result.WriteRune(char)
		i++
	}
	
	return result.String()
}

// removeTrailingCommas 移除尾随逗号
func (j *JsonDecoder) removeTrailingCommas(content string) string {
	// 移除对象中的尾随逗号 ,}
	trailingCommaObjectRegex := regexp.MustCompile(`,(\s*})`)
	content = trailingCommaObjectRegex.ReplaceAllString(content, "$1")
	
	// 移除数组中的尾随逗号 ,]
	trailingCommaArrayRegex := regexp.MustCompile(`,(\s*])`)
	content = trailingCommaArrayRegex.ReplaceAllString(content, "$1")
	
	return content
}

// Encode 将Storage对象编码为JSON数据
func (j *JsonDecoder) Encode(s storage.Storage) ([]byte, error) {
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

	// 编码为JSON（格式化输出）
	result, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	return result, nil
}