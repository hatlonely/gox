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
	
	return fs.convertValue(fs.data, reflect.ValueOf(object))
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
	
	// 处理目标为指针的情况
	if dst.Kind() == reflect.Ptr {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return fs.convertValue(src, dst.Elem())
	}
	
	// 如果源数据是打平的map，根据目标类型进行处理
	if flatData, ok := src.(map[string]interface{}); ok {
		switch dst.Kind() {
		case reflect.Struct:
			return fs.convertToStruct(flatData, dst)
		case reflect.Map:
			return fs.convertToMap(flatData, dst)
		case reflect.Slice:
			return fs.convertToSlice(flatData, dst)
		default:
			// 对于基本类型，尝试查找单个匹配的值
			if len(flatData) == 1 {
				for _, value := range flatData {
					return fs.convertDirectValue(value, dst)
				}
			}
		}
	}
	
	// 直接值转换
	return fs.convertDirectValue(src, dst)
}

// convertDirectValue 直接值转换，处理基本类型和时间类型
func (fs *FlatStorage) convertDirectValue(src interface{}, dst reflect.Value) error {
	srcValue := reflect.ValueOf(src)
	if !srcValue.IsValid() {
		return nil
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
	
	// 接口类型
	if dst.Kind() == reflect.Interface && dst.Type().NumMethod() == 0 {
		dst.Set(srcValue)
		return nil
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
func (fs *FlatStorage) convertToMap(flatData map[string]interface{}, dst reflect.Value) error {
	if dst.IsNil() {
		dst.Set(reflect.MakeMap(dst.Type()))
	}
	
	// 处理扁平数据到map的转换，移除公共前缀
	processedData := fs.removePrefixFromKeys(flatData)
	
	for key, value := range processedData {
		// 转换键
		keyValue := reflect.ValueOf(key)
		convertedKey := keyValue
		if !keyValue.Type().AssignableTo(dst.Type().Key()) {
			if keyValue.Type().ConvertibleTo(dst.Type().Key()) {
				convertedKey = keyValue.Convert(dst.Type().Key())
			} else {
				return fmt.Errorf("cannot convert key %v to %v", keyValue.Type(), dst.Type().Key())
			}
		}
		
		// 转换值
		dstValue := reflect.New(dst.Type().Elem()).Elem()
		if err := fs.convertDirectValue(value, dstValue); err != nil {
			return err
		}
		
		dst.SetMapIndex(convertedKey, dstValue)
	}
	
	return nil
}

// removePrefixFromKeys 移除键名中的公共前缀，用于map转换
func (fs *FlatStorage) removePrefixFromKeys(flatData map[string]interface{}) map[string]interface{} {
	if len(flatData) == 0 {
		return flatData
	}
	
	// 查找最长公共前缀
	var commonPrefix string
	keys := make([]string, 0, len(flatData))
	for key := range flatData {
		keys = append(keys, key)
	}
	
	if len(keys) > 0 {
		commonPrefix = keys[0]
		for _, key := range keys[1:] {
			commonPrefix = fs.longestCommonPrefix(commonPrefix, key)
		}
		
		// 如果公共前缀以分隔符结尾，移除分隔符后的所有内容作为前缀
		if idx := strings.LastIndex(commonPrefix, fs.KeySeparator); idx >= 0 {
			commonPrefix = commonPrefix[:idx+len(fs.KeySeparator)]
		} else {
			commonPrefix = ""
		}
	}
	
	// 构建移除前缀后的map
	result := make(map[string]interface{})
	for key, value := range flatData {
		cleanKey := key
		if commonPrefix != "" && strings.HasPrefix(key, commonPrefix) {
			cleanKey = key[len(commonPrefix):]
		}
		result[cleanKey] = value
	}
	
	return result
}

// longestCommonPrefix 计算两个字符串的最长公共前缀
func (fs *FlatStorage) longestCommonPrefix(a, b string) string {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	
	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	
	return a[:minLen]
}

// convertToSlice 转换为 slice 类型，支持直接从打平数据构建
func (fs *FlatStorage) convertToSlice(flatData map[string]interface{}, dst reflect.Value) error {
	// 首先尝试从打平的数据直接构建数组
	arrayItems := fs.extractArrayFromFlatData(flatData)
	
	if len(arrayItems) > 0 {
		dst.Set(reflect.MakeSlice(dst.Type(), len(arrayItems), len(arrayItems)))
		
		for i, itemData := range arrayItems {
			dstItem := dst.Index(i)
			
			// 根据目标元素类型进行转换
			if dstItem.Kind() == reflect.Struct {
				if err := fs.convertToStruct(itemData, dstItem); err != nil {
					return err
				}
			} else if dstItem.Kind() == reflect.Map || (dstItem.Type().Kind() == reflect.Map) {
				// 处理 map 类型，包括 map[string]interface{} 和其别名类型
				if dstItem.IsNil() {
					dstItem.Set(reflect.MakeMap(dstItem.Type()))
				}
				
				for k, v := range itemData {
					keyValue := reflect.ValueOf(k)
					convertedKey := keyValue
					if !keyValue.Type().AssignableTo(dstItem.Type().Key()) {
						if keyValue.Type().ConvertibleTo(dstItem.Type().Key()) {
							convertedKey = keyValue.Convert(dstItem.Type().Key())
						} else {
							return fmt.Errorf("cannot convert key %v to %v", keyValue.Type(), dstItem.Type().Key())
						}
					}
					
					valueReflect := reflect.ValueOf(v)
					convertedValue := valueReflect
					if !valueReflect.Type().AssignableTo(dstItem.Type().Elem()) {
						if valueReflect.Type().ConvertibleTo(dstItem.Type().Elem()) {
							convertedValue = valueReflect.Convert(dstItem.Type().Elem())
						} else {
							// 对于 interface{} 目标类型，直接赋值
							if dstItem.Type().Elem().Kind() == reflect.Interface {
								convertedValue = valueReflect
							} else {
								return fmt.Errorf("cannot convert value %v to %v", valueReflect.Type(), dstItem.Type().Elem())
							}
						}
					}
					
					dstItem.SetMapIndex(convertedKey, convertedValue)
				}
			} else {
				return fmt.Errorf("unsupported slice element type: %v", dstItem.Type())
			}
		}
		return nil
	}
	
	// 回退到重建嵌套结构的方法
	nested := fs.buildNestedStructure()
	
	if nestedSlice, ok := nested.([]interface{}); ok {
		length := len(nestedSlice)
		dst.Set(reflect.MakeSlice(dst.Type(), length, length))
		
		for i := 0; i < length; i++ {
			dstItem := dst.Index(i)
			if dstItem.Kind() == reflect.Struct {
				if itemMap, ok := nestedSlice[i].(map[string]interface{}); ok {
					if err := fs.convertToStruct(itemMap, dstItem); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("cannot convert array item to struct: expected map, got %T", nestedSlice[i])
				}
			} else {
				if err := fs.convertDirectValue(nestedSlice[i], dstItem); err != nil {
					return err
				}
			}
		}
		return nil
	}
	
	return fmt.Errorf("cannot convert flat data to slice")
}

// extractArrayFromFlatData 从打平的数据中提取数组数据
func (fs *FlatStorage) extractArrayFromFlatData(flatData map[string]interface{}) []map[string]interface{} {
	// 按索引分组数据
	indexedData := make(map[int]map[string]interface{})
	maxIndex := -1
	
	for key, value := range flatData {
		// 解析类似 "pools_0_name", "pools_1_host" 的键名
		if parts := strings.Split(key, fs.KeySeparator); len(parts) >= 2 {
			// 查找索引部分
			for i := 1; i < len(parts); i++ {
				if index, err := strconv.Atoi(parts[i]); err == nil {
					if index > maxIndex {
						maxIndex = index
					}
					
					// 构建子键名
					var subKey string
					if i+1 < len(parts) {
						subKey = strings.Join(parts[i+1:], fs.KeySeparator)
					} else {
						subKey = parts[0] // 如果没有后续部分，使用前缀作为键
					}
					
					if _, exists := indexedData[index]; !exists {
						indexedData[index] = make(map[string]interface{})
					}
					indexedData[index][subKey] = value
					break
				}
			}
		}
	}
	
	// 构建结果数组
	if maxIndex >= 0 {
		result := make([]map[string]interface{}, maxIndex+1)
		for i := 0; i <= maxIndex; i++ {
			if data, exists := indexedData[i]; exists {
				result[i] = data
			} else {
				result[i] = make(map[string]interface{})
			}
		}
		return result
	}
	
	return nil
}

// convertToStruct 转换为 struct 类型，支持基于字段路径的智能匹配
func (fs *FlatStorage) convertToStruct(flatData map[string]interface{}, dst reflect.Value) error {
	return fs.convertToStructWithPrefix(flatData, dst, "")
}

// convertToStructWithPrefix 递归转换结构体，支持路径前缀
func (fs *FlatStorage) convertToStructWithPrefix(flatData map[string]interface{}, dst reflect.Value, prefix string) error {
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
		
		// 构建完整的字段路径
		fullPath := fieldName
		if prefix != "" {
			fullPath = prefix + fs.KeySeparator + fieldName
		}
		
		// 查找匹配的键值
		value, found := fs.findMatchingValue(flatData, fullPath)
		if found {
			if fieldValue.Kind() == reflect.Struct && fieldValue.Type() != reflect.TypeOf(time.Time{}) {
				// 如果是嵌套结构体（但不是 time.Time），递归处理
				if err := fs.convertToStructWithPrefix(flatData, fieldValue, fullPath); err != nil {
					return err
				}
			} else {
				// 直接赋值（包括 time.Time）
				if err := fs.convertDirectValue(value, fieldValue); err != nil {
					return err
				}
			}
		} else {
			// 没有直接匹配时，根据字段类型进行特殊处理
			switch fieldValue.Kind() {
			case reflect.Struct:
				// 嵌套结构体，递归处理
				if err := fs.convertToStructWithPrefix(flatData, fieldValue, fullPath); err != nil {
					return err
				}
			case reflect.Slice:
				// 切片类型，查找相关的数组数据
				arrayData := fs.filterDataWithPrefix(flatData, fullPath)
				if len(arrayData) > 0 {
					if err := fs.convertToSlice(arrayData, fieldValue); err != nil {
						return err
					}
				}
			case reflect.Map:
				// Map类型，查找相关的map数据
				mapData := fs.filterDataWithPrefix(flatData, fullPath)
				if len(mapData) > 0 {
					if err := fs.convertToMap(mapData, fieldValue); err != nil {
						return err
					}
				}
			}
		}
	}
	
	return nil
}

// findMatchingValue 查找与字段路径匹配的值，支持模糊匹配
func (fs *FlatStorage) findMatchingValue(flatData map[string]interface{}, fieldPath string) (interface{}, bool) {
	// 1. 精确匹配
	if value, exists := flatData[fieldPath]; exists {
		return value, true
	}
	
	// 2. 尝试不同的分隔符组合进行匹配
	// 将字段路径转换为可能的键名模式
	patterns := fs.generateKeyPatterns(fieldPath)
	
	// 3. 尝试添加常见前缀
	patterns = append(patterns, fs.generatePrefixedPatterns(fieldPath)...)
	
	for _, pattern := range patterns {
		if value, exists := flatData[pattern]; exists {
			return value, true
		}
	}
	return nil, false
}

// filterDataWithPrefix 过滤出具有指定前缀的数据
func (fs *FlatStorage) filterDataWithPrefix(flatData map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	
	// 生成可能的前缀模式
	prefixPatterns := fs.generateKeyPatterns(prefix)
	prefixPatterns = append(prefixPatterns, fs.generatePrefixedPatterns(prefix)...)
	
	for _, pattern := range prefixPatterns {
		prefixWithSep := pattern + fs.KeySeparator
		
		for key, value := range flatData {
			if strings.HasPrefix(key, prefixWithSep) {
				result[key] = value
			}
		}
	}
	
	return result
}

// generateKeyPatterns 生成字段路径的可能键名模式
func (fs *FlatStorage) generateKeyPatterns(fieldPath string) []string {
	var patterns []string
	
	// 原始路径
	patterns = append(patterns, fieldPath)
	
	// 如果使用下划线分隔符
	if fs.KeySeparator == "_" {
		// 也尝试点号分隔符
		dotPath := strings.ReplaceAll(fieldPath, "_", ".")
		patterns = append(patterns, dotPath)
		
		// 全大写版本
		patterns = append(patterns, strings.ToUpper(fieldPath))
		
		// 全小写版本  
		patterns = append(patterns, strings.ToLower(fieldPath))
	}
	
	// 如果使用点号分隔符，也尝试下划线
	if fs.KeySeparator == "." {
		underscorePath := strings.ReplaceAll(fieldPath, ".", "_")
		patterns = append(patterns, underscorePath)
		
		// 全大写版本
		patterns = append(patterns, strings.ToUpper(underscorePath))
		
		// 全小写版本  
		patterns = append(patterns, strings.ToLower(underscorePath))
	}
	
	return patterns
}

// generatePrefixedPatterns 生成带前缀的模式，用于匹配环境变量风格的键名
func (fs *FlatStorage) generatePrefixedPatterns(fieldPath string) []string {
	var patterns []string
	
	// 常见前缀列表
	prefixes := []string{"APP", "app"}
	
	for _, prefix := range prefixes {
		// 生成前缀 + 分隔符 + 字段路径的模式
		prefixed := prefix + fs.KeySeparator + fieldPath
		patterns = append(patterns, prefixed)
		patterns = append(patterns, strings.ToUpper(prefixed))
		patterns = append(patterns, strings.ToLower(prefixed))
		
		// 如果分隔符是下划线，也尝试点号
		if fs.KeySeparator == "_" {
			dotPrefixed := strings.ReplaceAll(prefixed, "_", ".")
			patterns = append(patterns, dotPrefixed)
		}
		
		// 如果分隔符是点号，也尝试下划线
		if fs.KeySeparator == "." {
			underscorePrefixed := strings.ReplaceAll(prefixed, ".", "_")
			patterns = append(patterns, underscorePrefixed)
			patterns = append(patterns, strings.ToUpper(underscorePrefixed))
			patterns = append(patterns, strings.ToLower(underscorePrefixed))
		}
	}
	
	return patterns
}