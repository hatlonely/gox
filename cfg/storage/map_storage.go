package storage

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// MapStorage 基于 map 和 slice 的存储实现
type MapStorage struct {
	data interface{}
}

// Data 获取存储的原始数据
func (ms *MapStorage) Data() interface{} {
	return ms.data
}

// NewMapStorage 创建一个新的 MapStorage 实例
func NewMapStorage(data interface{}) *MapStorage {
	return &MapStorage{data: data}
}

// Sub 获取子配置存储对象
// key 可以包含点号（.）表示多级嵌套，[]表示数组索引
// 例如 "database.connections[0].host"
func (ms *MapStorage) Sub(key string) Storage {
	if key == "" {
		return ms
	}
	
	result := ms.getValue(key)
	return NewMapStorage(result)
}

// ConvertTo 将配置数据转成结构体或者 map/slice 等任意结构
func (ms *MapStorage) ConvertTo(object interface{}) error {
	return ms.convertValue(ms.data, reflect.ValueOf(object))
}

// getValue 根据 key 获取嵌套的值
func (ms *MapStorage) getValue(key string) interface{} {
	keys := ms.parseKey(key)
	current := ms.data
	
	for _, k := range keys {
		current = ms.getValueByKey(current, k)
		if current == nil {
			return nil
		}
	}
	
	return current
}

// parseKey 解析 key 字符串，支持点号和数组索引
func (ms *MapStorage) parseKey(key string) []string {
	var keys []string
	var current string
	inBracket := false
	
	for _, char := range key {
		switch char {
		case '.':
			if !inBracket {
				if current != "" {
					keys = append(keys, current)
					current = ""
				}
			} else {
				current += string(char)
			}
		case '[':
			if current != "" {
				keys = append(keys, current)
				current = ""
			}
			inBracket = true
		case ']':
			if inBracket {
				if current != "" {
					keys = append(keys, current)
					current = ""
				}
				inBracket = false
			} else {
				current += string(char)
			}
		default:
			current += string(char)
		}
	}
	
	// 添加最后的部分
	if current != "" {
		keys = append(keys, current)
	}
	
	return keys
}

// getValueByKey 通过单个 key 获取值
func (ms *MapStorage) getValueByKey(data interface{}, key string) interface{} {
	if data == nil {
		return nil
	}
	
	rv := reflect.ValueOf(data)
	
	// 处理指针
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	
	switch rv.Kind() {
	case reflect.Map:
		// 处理 map 访问
		keyValue := reflect.ValueOf(key)
		value := rv.MapIndex(keyValue)
		if !value.IsValid() {
			return nil
		}
		return value.Interface()
		
	case reflect.Slice, reflect.Array:
		// 处理数组/切片访问
		index, err := strconv.Atoi(key)
		if err != nil {
			return nil
		}
		if index < 0 || index >= rv.Len() {
			return nil
		}
		return rv.Index(index).Interface()
		
	case reflect.Struct:
		// 处理结构体字段访问
		field := rv.FieldByName(key)
		if !field.IsValid() {
			// 尝试查找 json tag
			rt := rv.Type()
			for i := 0; i < rt.NumField(); i++ {
				fieldType := rt.Field(i)
				if tag := fieldType.Tag.Get("json"); tag != "" {
					tagName := strings.Split(tag, ",")[0]
					if tagName == key {
						field = rv.Field(i)
						break
					}
				}
			}
		}
		if !field.IsValid() {
			return nil
		}
		return field.Interface()
	}
	
	return nil
}

// convertValue 将数据转换为目标类型
func (ms *MapStorage) convertValue(src interface{}, dst reflect.Value) error {
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
		return ms.convertValue(src, dst.Elem())
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
	if err := ms.convertTimeTypes(srcValue, dst); err == nil {
		return nil
	} else if err.Error() != "not a time type" {
		return err
	}
	
	// 类型转换
	switch dst.Kind() {
	case reflect.Map:
		return ms.convertToMap(srcValue, dst)
	case reflect.Slice:
		return ms.convertToSlice(srcValue, dst)
	case reflect.Struct:
		return ms.convertToStruct(srcValue, dst)
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
func (ms *MapStorage) convertTimeTypes(src, dst reflect.Value) error {
	dstType := dst.Type()
	
	// 转换为 time.Duration
	if dstType == reflect.TypeOf(time.Duration(0)) {
		return ms.convertToDuration(src, dst)
	}
	
	// 转换为 time.Time
	if dstType == reflect.TypeOf(time.Time{}) {
		return ms.convertToTime(src, dst)
	}
	
	return fmt.Errorf("not a time type")
}

// convertToDuration 将源值转换为 time.Duration
func (ms *MapStorage) convertToDuration(src, dst reflect.Value) error {
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
		// 将整数视为纳秒
		nanoseconds := src.Int()
		duration := time.Duration(nanoseconds)
		dst.Set(reflect.ValueOf(duration))
		return nil
		
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// 将无符号整数视为纳秒
		nanoseconds := src.Uint()
		duration := time.Duration(nanoseconds)
		dst.Set(reflect.ValueOf(duration))
		return nil
		
	case reflect.Float32, reflect.Float64:
		// 将浮点数视为秒
		seconds := src.Float()
		duration := time.Duration(seconds * float64(time.Second))
		dst.Set(reflect.ValueOf(duration))
		return nil
	}
	
	return fmt.Errorf("cannot convert %v to time.Duration", src.Type())
}

// convertToTime 将源值转换为 time.Time
func (ms *MapStorage) convertToTime(src, dst reflect.Value) error {
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
		// Unix 时间戳（秒）
		timestamp := src.Int()
		t := time.Unix(timestamp, 0)
		dst.Set(reflect.ValueOf(t))
		return nil
		
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Unix 时间戳（秒）
		timestamp := int64(src.Uint())
		t := time.Unix(timestamp, 0)
		dst.Set(reflect.ValueOf(t))
		return nil
		
	case reflect.Float32, reflect.Float64:
		// Unix 时间戳（秒，支持小数）
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
func (ms *MapStorage) convertToMap(src, dst reflect.Value) error {
	if src.Kind() != reflect.Map {
		return fmt.Errorf("source is not a map")
	}
	
	if dst.IsNil() {
		dst.Set(reflect.MakeMap(dst.Type()))
	}
	
	for _, key := range src.MapKeys() {
		srcValue := src.MapIndex(key)
		dstValue := reflect.New(dst.Type().Elem()).Elem()
		
		if err := ms.convertValue(srcValue.Interface(), dstValue); err != nil {
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
func (ms *MapStorage) convertToSlice(src, dst reflect.Value) error {
	if src.Kind() != reflect.Slice && src.Kind() != reflect.Array {
		return fmt.Errorf("source is not a slice or array")
	}
	
	length := src.Len()
	dst.Set(reflect.MakeSlice(dst.Type(), length, length))
	
	for i := 0; i < length; i++ {
		srcItem := src.Index(i)
		dstItem := dst.Index(i)
		
		if err := ms.convertValue(srcItem.Interface(), dstItem); err != nil {
			return err
		}
	}
	
	return nil
}

// convertToStruct 转换为 struct 类型
func (ms *MapStorage) convertToStruct(src, dst reflect.Value) error {
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
		
		// 获取字段名，优先使用 json tag，然后是 yaml tag，toml tag，最后是 ini tag
		fieldName := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
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
			if err := ms.convertValue(srcFieldValue.Interface(), fieldValue); err != nil {
				return err
			}
		}
	}
	
	return nil
}