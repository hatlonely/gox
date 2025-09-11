package storage

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// FlatStorage 打平的键值存储实现
// 使用点号分隔的键名存储嵌套数据，值都是基本类型
type FlatStorage struct {
	data map[string]interface{}
	// KeySeparator 键名分隔符，默认为 "."
	KeySeparator string
	// ArrayFormat 数组格式化方式，默认为 "[%d]"
	ArrayFormat string
}

// NewFlatStorage 创建一个新的 FlatStorage 实例
func NewFlatStorage(data map[string]interface{}) *FlatStorage {
	return &FlatStorage{
		data:         data,
		KeySeparator: ".",
		ArrayFormat:  "[%d]",
	}
}

// NewFlatStorageWithOptions 创建带选项的 FlatStorage 实例
func NewFlatStorageWithOptions(data map[string]interface{}, separator, arrayFormat string) *FlatStorage {
	return &FlatStorage{
		data:         data,
		KeySeparator: separator,
		ArrayFormat:  arrayFormat,
	}
}

// NewFlatStorageFromNested 从嵌套数据创建 FlatStorage
func NewFlatStorageFromNested(nestedData interface{}) *FlatStorage {
	fs := NewFlatStorage(make(map[string]interface{}))
	fs.flattenData("", nestedData)
	return fs
}

// Data 获取存储的原始数据
func (fs *FlatStorage) Data() map[string]interface{} {
	return fs.data
}

// Sub 获取子配置存储对象
func (fs *FlatStorage) Sub(key string) Storage {
	if key == "" {
		return fs
	}

	// 构建子存储的数据
	subData := make(map[string]interface{})
	prefix := key + fs.KeySeparator
	
	// 查找所有以指定前缀开头的键
	for k, v := range fs.data {
		if k == key {
			// 如果键完全匹配，说明这是一个叶子节点
			return NewFlatStorageWithOptions(map[string]interface{}{"": v}, fs.KeySeparator, fs.ArrayFormat)
		}
		if strings.HasPrefix(k, prefix) {
			// 移除前缀，保留后续部分作为子键
			subKey := k[len(prefix):]
			subData[subKey] = v
		}
	}
	
	// 特殊处理数组情况 - 查找数组格式的键
	if len(subData) == 0 {
		// 尝试查找数组格式的键，根据不同的数组格式
		for k, v := range fs.data {
			if strings.HasPrefix(k, key) && len(k) > len(key) {
				remaining := k[len(key):]
				// 检查是否是数组格式，如 [0], [1] 等或者 _0, _1 等
				if strings.HasPrefix(remaining, "[") || 
				   (strings.Contains(fs.ArrayFormat, "%d") && strings.HasPrefix(remaining, strings.Split(fs.ArrayFormat, "%d")[0])) {
					subData[remaining] = v
				}
			}
		}
	}
	
	if len(subData) == 0 {
		// 没有找到匹配的键，返回空存储
		return NewFlatStorageWithOptions(make(map[string]interface{}), fs.KeySeparator, fs.ArrayFormat)
	}
	
	return NewFlatStorageWithOptions(subData, fs.KeySeparator, fs.ArrayFormat)
}

// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
func (fs *FlatStorage) ConvertTo(object interface{}) error {
	if len(fs.data) == 0 {
		return nil
	}
	
	// 如果只有一个空键，说明这是一个叶子值
	if len(fs.data) == 1 {
		if value, exists := fs.data[""]; exists {
			return fs.convertValue(value, reflect.ValueOf(object))
		}
	}
	
	// 重建嵌套结构然后转换
	nested := fs.buildNestedStructure()
	
	// 特殊处理：如果目标是基本类型但嵌套结构是map，尝试提取单个值
	dst := reflect.ValueOf(object)
	if dst.Kind() == reflect.Ptr && dst.Elem().Kind() != reflect.Map && dst.Elem().Kind() != reflect.Slice && dst.Elem().Kind() != reflect.Struct {
		if nestedMap, ok := nested.(map[string]interface{}); ok && len(nestedMap) == 1 {
			for _, value := range nestedMap {
				return fs.convertValue(value, dst)
			}
		}
	}
	
	return fs.convertValue(nested, dst)
}

// flattenData 将嵌套数据打平存储
func (fs *FlatStorage) flattenData(prefix string, data interface{}) {
	rv := reflect.ValueOf(data)
	
	// 处理指针
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	
	switch rv.Kind() {
	case reflect.Map:
		for _, key := range rv.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			newPrefix := keyStr
			if prefix != "" {
				newPrefix = prefix + fs.KeySeparator + keyStr
			}
			fs.flattenData(newPrefix, rv.MapIndex(key).Interface())
		}
		
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			arrayKey := fmt.Sprintf(fs.ArrayFormat, i)
			newPrefix := arrayKey
			if prefix != "" {
				newPrefix = prefix + arrayKey
			}
			fs.flattenData(newPrefix, rv.Index(i).Interface())
		}
		
	default:
		// 基本类型，直接存储
		if prefix == "" {
			prefix = "root"
		}
		fs.data[prefix] = data
	}
}

// buildNestedStructure 从打平的数据重建嵌套结构
func (fs *FlatStorage) buildNestedStructure() interface{} {
	if len(fs.data) == 0 {
		return nil
	}
	
	// 如果只有一个键且为空，返回该值
	if len(fs.data) == 1 {
		if value, exists := fs.data[""]; exists {
			return value
		}
	}
	
	// 检查是否是纯数组（所有键都是 [0], [1], [2] 格式或者都以 [数字] 开头）
	arrayPattern := true
	maxIndex := -1
	hasDirectArray := false // 是否有直接的数组键如 [0], [1]
	hasNestedArray := false // 是否有嵌套的数组键如 [0].name, [1].name
	
	for key := range fs.data {
		if key == "" {
			continue
		}
		
		// 检查是否是直接数组格式 [0], [1], [2]
		if strings.HasPrefix(key, "[") {
			if dotIndex := strings.Index(key, "."); dotIndex == -1 {
				// 纯数组键，如 [0], [1]
				if strings.HasSuffix(key, "]") {
					indexStr := key[1 : len(key)-1]
					if index, err := strconv.Atoi(indexStr); err == nil {
						if index > maxIndex {
							maxIndex = index
						}
						hasDirectArray = true
					} else {
						arrayPattern = false
						break
					}
				} else {
					arrayPattern = false
					break
				}
			} else {
				// 嵌套数组键，如 [0].name, [1].email
				closeBracket := strings.Index(key, "]")
				if closeBracket > 0 && closeBracket < dotIndex {
					indexStr := key[1:closeBracket]
					if index, err := strconv.Atoi(indexStr); err == nil {
						if index > maxIndex {
							maxIndex = index
						}
						hasNestedArray = true
					} else {
						arrayPattern = false
						break
					}
				} else {
					arrayPattern = false
					break
				}
			}
		} else {
			arrayPattern = false
			break
		}
	}
	
	if arrayPattern && maxIndex >= 0 {
		if hasDirectArray && !hasNestedArray {
			// 构建简单数组结构
			result := make([]interface{}, maxIndex+1)
			for key, value := range fs.data {
				if key == "" {
					continue
				}
				indexStr := key[1 : len(key)-1]
				if index, err := strconv.Atoi(indexStr); err == nil {
					result[index] = value
				}
			}
			return result
		} else if hasNestedArray {
			// 构建嵌套对象数组结构
			result := make([]interface{}, maxIndex+1)
			for i := 0; i <= maxIndex; i++ {
				result[i] = make(map[string]interface{})
			}
			
			for key, value := range fs.data {
				if key == "" {
					continue
				}
				
				closeBracket := strings.Index(key, "]")
				dotIndex := strings.Index(key, ".")
				if closeBracket > 0 && dotIndex > closeBracket {
					indexStr := key[1:closeBracket]
					if index, err := strconv.Atoi(indexStr); err == nil && index <= maxIndex {
						fieldName := key[dotIndex+1:]
						if obj, ok := result[index].(map[string]interface{}); ok {
							obj[fieldName] = value
						}
					}
				}
			}
			return result
		}
	}
	
	// 构建对象结构
	result := make(map[string]interface{})
	for key, value := range fs.data {
		if key == "" {
			continue
		}
		fs.setNestedValue(result, key, value)
	}
	
	return result
}

// setNestedValue 在嵌套结构中设置值
func (fs *FlatStorage) setNestedValue(target map[string]interface{}, key string, value interface{}) {
	parts := fs.parseKey(key)
	current := target
	
	for i, part := range parts {
		if i == len(parts)-1 {
			// 最后一个部分，设置值
			current[part.key] = value
		} else {
			// 中间部分，确保存在容器
			if part.isArray {
				// 需要创建数组
				if _, exists := current[part.key]; !exists {
					current[part.key] = make([]interface{}, 0)
				}
				arr, ok := current[part.key].([]interface{})
				if !ok {
					current[part.key] = make([]interface{}, 0)
					arr = current[part.key].([]interface{})
				}
				
				// 确保数组足够大
				for len(arr) <= part.index {
					arr = append(arr, make(map[string]interface{}))
				}
				current[part.key] = arr
				
				// 获取数组元素
				if item, ok := arr[part.index].(map[string]interface{}); ok {
					current = item
				} else {
					newItem := make(map[string]interface{})
					arr[part.index] = newItem
					current[part.key] = arr
					current = newItem
				}
			} else {
				// 需要创建对象
				if _, exists := current[part.key]; !exists {
					current[part.key] = make(map[string]interface{})
				}
				if nextMap, ok := current[part.key].(map[string]interface{}); ok {
					current = nextMap
				} else {
					newMap := make(map[string]interface{})
					current[part.key] = newMap
					current = newMap
				}
			}
		}
	}
}

// KeyPart 表示解析后的键部分
type KeyPart struct {
	key     string
	isArray bool
	index   int
}

// parseKey 解析键名，支持点号和数组索引
func (fs *FlatStorage) parseKey(key string) []KeyPart {
	var parts []KeyPart
	var current string
	inBracket := false
	
	for _, char := range key {
		switch char {
		case '.':
			if !inBracket && current != "" {
				parts = append(parts, KeyPart{key: current, isArray: false})
				current = ""
			} else {
				current += string(char)
			}
		case '[':
			if current != "" {
				// 当前部分是数组名
				inBracket = true
			} else {
				current += string(char)
			}
		case ']':
			if inBracket {
				// 解析数组索引
				if index, err := strconv.Atoi(current); err == nil {
					if len(parts) > 0 {
						parts[len(parts)-1].isArray = true
						parts[len(parts)-1].index = index
					}
				}
				current = ""
				inBracket = false
			} else {
				current += string(char)
			}
		default:
			if inBracket {
				current += string(char)
			} else if char == '[' && current != "" {
				// 开始数组索引
				parts = append(parts, KeyPart{key: current})
				current = ""
				inBracket = true
			} else {
				current += string(char)
			}
		}
	}
	
	if current != "" {
		parts = append(parts, KeyPart{key: current, isArray: false})
	}
	
	return parts
}

// convertValue 将值转换为目标类型
func (fs *FlatStorage) convertValue(src interface{}, dst reflect.Value) error {
	if !dst.CanSet() && dst.Kind() != reflect.Ptr {
		return fmt.Errorf("destination is not settable")
	}
	
	srcValue := reflect.ValueOf(src)
	if !srcValue.IsValid() {
		return nil
	}
	
	// 处理目标为指针的情况
	if dst.Kind() == reflect.Ptr {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return fs.convertValue(src, dst.Elem())
	}
	
	// 处理源为指针的情况
	for srcValue.Kind() == reflect.Ptr {
		if srcValue.IsNil() {
			return nil
		}
		srcValue = srcValue.Elem()
	}
	
	// 类型完全匹配
	if srcValue.Type().AssignableTo(dst.Type()) {
		dst.Set(srcValue)
		return nil
	}
	
	// 特殊类型转换：time.Duration 和 time.Time
	if err := fs.convertTimeTypes(srcValue, dst); err == nil {
		return nil
	} else if err.Error() != "not a time type" {
		return err
	}
	
	// 类型转换
	switch dst.Kind() {
	case reflect.Map:
		return fs.convertToMap(srcValue, dst)
	case reflect.Slice:
		return fs.convertToSlice(srcValue, dst)
	case reflect.Struct:
		return fs.convertToStruct(srcValue, dst)
	case reflect.Interface:
		if dst.Type().NumMethod() == 0 {
			dst.Set(srcValue)
			return nil
		}
	}
	
	// 尝试直接转换
	if srcValue.Type().ConvertibleTo(dst.Type()) {
		dst.Set(srcValue.Convert(dst.Type()))
		return nil
	}
	
	return fmt.Errorf("cannot convert %v to %v", srcValue.Type(), dst.Type())
}

// convertTimeTypes 处理时间相关类型的转换
func (fs *FlatStorage) convertTimeTypes(src, dst reflect.Value) error {
	dstType := dst.Type()
	
	// 转换为 time.Duration
	if dstType == reflect.TypeOf(time.Duration(0)) {
		return fs.convertToDuration(src, dst)
	}
	
	// 转换为 time.Time
	if dstType == reflect.TypeOf(time.Time{}) {
		return fs.convertToTime(src, dst)
	}
	
	return fmt.Errorf("not a time type")
}

// convertToDuration 将源值转换为 time.Duration
func (fs *FlatStorage) convertToDuration(src, dst reflect.Value) error {
	switch src.Kind() {
	case reflect.String:
		str := src.String()
		duration, err := time.ParseDuration(str)
		if err != nil {
			return fmt.Errorf("failed to parse duration %q: %v", str, err)
		}
		dst.Set(reflect.ValueOf(duration))
		return nil
		
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		nanoseconds := src.Int()
		duration := time.Duration(nanoseconds)
		dst.Set(reflect.ValueOf(duration))
		return nil
		
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		nanoseconds := src.Uint()
		duration := time.Duration(nanoseconds)
		dst.Set(reflect.ValueOf(duration))
		return nil
		
	case reflect.Float32, reflect.Float64:
		seconds := src.Float()
		duration := time.Duration(seconds * float64(time.Second))
		dst.Set(reflect.ValueOf(duration))
		return nil
	}
	
	return fmt.Errorf("cannot convert %v to time.Duration", src.Type())
}

// convertToTime 将源值转换为 time.Time
func (fs *FlatStorage) convertToTime(src, dst reflect.Value) error {
	switch src.Kind() {
	case reflect.String:
		str := src.String()
		
		// 尝试多种时间格式
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
			"15:04:05",
		}
		
		for _, format := range formats {
			if t, err := time.Parse(format, str); err == nil {
				dst.Set(reflect.ValueOf(t))
				return nil
			}
		}
		
		return fmt.Errorf("failed to parse time %q", str)
		
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		timestamp := src.Int()
		t := time.Unix(timestamp, 0)
		dst.Set(reflect.ValueOf(t))
		return nil
		
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		timestamp := int64(src.Uint())
		t := time.Unix(timestamp, 0)
		dst.Set(reflect.ValueOf(t))
		return nil
		
	case reflect.Float32, reflect.Float64:
		timestamp := src.Float()
		seconds := int64(timestamp)
		nanoseconds := int64((timestamp - float64(seconds)) * 1e9)
		t := time.Unix(seconds, nanoseconds)
		dst.Set(reflect.ValueOf(t))
		return nil
	}
	
	return fmt.Errorf("cannot convert %v to time.Time", src.Type())
}

// convertToMap 转换为 map 类型
func (fs *FlatStorage) convertToMap(src, dst reflect.Value) error {
	if src.Kind() != reflect.Map {
		return fmt.Errorf("source is not a map")
	}
	
	if dst.IsNil() {
		dst.Set(reflect.MakeMap(dst.Type()))
	}
	
	for _, key := range src.MapKeys() {
		srcValue := src.MapIndex(key)
		dstValue := reflect.New(dst.Type().Elem()).Elem()
		
		if err := fs.convertValue(srcValue.Interface(), dstValue); err != nil {
			return err
		}
		
		convertedKey := key
		if !key.Type().AssignableTo(dst.Type().Key()) {
			if key.Type().ConvertibleTo(dst.Type().Key()) {
				convertedKey = key.Convert(dst.Type().Key())
			} else {
				return fmt.Errorf("cannot convert key %v to %v", key.Type(), dst.Type().Key())
			}
		}
		
		dst.SetMapIndex(convertedKey, dstValue)
	}
	
	return nil
}

// convertToSlice 转换为 slice 类型
func (fs *FlatStorage) convertToSlice(src, dst reflect.Value) error {
	if src.Kind() != reflect.Slice && src.Kind() != reflect.Array {
		return fmt.Errorf("source is not a slice or array")
	}
	
	length := src.Len()
	dst.Set(reflect.MakeSlice(dst.Type(), length, length))
	
	for i := 0; i < length; i++ {
		srcItem := src.Index(i)
		dstItem := dst.Index(i)
		
		if err := fs.convertValue(srcItem.Interface(), dstItem); err != nil {
			return err
		}
	}
	
	return nil
}

// convertToStruct 转换为 struct 类型
func (fs *FlatStorage) convertToStruct(src, dst reflect.Value) error {
	if src.Kind() != reflect.Map {
		return fmt.Errorf("source is not a map")
	}
	
	dstType := dst.Type()
	
	for i := 0; i < dstType.NumField(); i++ {
		field := dstType.Field(i)
		fieldValue := dst.Field(i)
		
		if !fieldValue.CanSet() {
			continue
		}
		
		// 获取字段名，优先使用 cfg tag，然后是其他标签
		fieldName := field.Name
		if tag := field.Tag.Get("cfg"); tag != "" {
			tagName := strings.Split(tag, ",")[0]
			if tagName != "-" && tagName != "" {
				fieldName = tagName
			}
		} else if tag := field.Tag.Get("json"); tag != "" {
			tagName := strings.Split(tag, ",")[0]
			if tagName != "-" && tagName != "" {
				fieldName = tagName
			}
		} else if tag := field.Tag.Get("yaml"); tag != "" {
			tagName := strings.Split(tag, ",")[0]
			if tagName != "-" && tagName != "" {
				fieldName = tagName
			}
		} else if tag := field.Tag.Get("toml"); tag != "" {
			tagName := strings.Split(tag, ",")[0]
			if tagName != "-" && tagName != "" {
				fieldName = tagName
			}
		} else if tag := field.Tag.Get("ini"); tag != "" {
			tagName := strings.Split(tag, ",")[0]
			if tagName != "-" && tagName != "" {
				fieldName = tagName
			}
		}
		
		// 查找对应的源值
		var srcFieldValue reflect.Value
		for _, key := range src.MapKeys() {
			if key.String() == fieldName {
				srcFieldValue = src.MapIndex(key)
				break
			}
		}
		
		if srcFieldValue.IsValid() {
			if err := fs.convertValue(srcFieldValue.Interface(), fieldValue); err != nil {
				return err
			}
		}
	}
	
	return nil
}