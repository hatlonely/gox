package storage

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// MapStorage 基于 map 和 slice 的存储实现
type MapStorage struct {
	data interface{}
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
		
		// 获取字段名，优先使用 json tag
		fieldName := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
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