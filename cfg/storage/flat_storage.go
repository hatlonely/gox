package storage

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hatlonely/gox/cfg/def"
)

//	data := map[string]interface{}{
//		"name": "test-app",
//		"database-host": "localhost",
//		"database-port": 3306,
//		"servers-0-host": "server1",
//		"servers-0-port": 8080,
//		"servers-1-host": "server2",
//		"servers-1-port": 8080,
//	}
type FlatStorage struct {
	data           map[string]interface{}
	separator      string
	enableDefaults bool
	uppercase      bool
	lowercase      bool

	parent *FlatStorage
	prefix string
}

func NewFlatStorage(data map[string]interface{}) *FlatStorage {
	return &FlatStorage{
		data:      data,
		separator: ".",
	}
}

func (fs *FlatStorage) WithDefaults(enable bool) *FlatStorage {
	fs.enableDefaults = enable
	return fs
}

func (fs *FlatStorage) WithSeparator(sep string) *FlatStorage {
	fs.separator = sep
	return fs
}

func (fs *FlatStorage) WithUppercase(enable bool) *FlatStorage {
	fs.uppercase = enable
	return fs
}

func (fs *FlatStorage) WithLowercase(enable bool) *FlatStorage {
	fs.lowercase = enable
	return fs
}

func (fs *FlatStorage) Data() map[string]interface{} {
	return fs.data
}

func (fs *FlatStorage) Sub(key string) Storage {
	if key == "" {
		return fs
	}

	if fs.parent != nil {
		return fs.parent.Sub(fs.prefix + "." + key)
	}

	keys := fs.parseKey(key)

	return &FlatStorage{
		parent: fs,
		prefix: strings.Join(keys, fs.separator),
	}
}

func (fs *FlatStorage) ConvertTo(object interface{}) error {
	if fs == nil {
		return nil
	}
	
	// 首先设置默认值
	if fs.enableDefaults {
		err := def.SetDefaults(object)
		if err != nil {
			return fmt.Errorf("failed to set defaults: %v", err)
		}
	}
	
	// 转换值
	return fs.convertValue("", reflect.ValueOf(object))
}

func (fs *FlatStorage) get(key string) interface{} {
	if fs.prefix != "" {
		return fs.parent.get(fs.prefix + "." + key)
	}

	// 应用大小写转换
	actualKey := key
	if fs.uppercase {
		actualKey = strings.ToUpper(key)
	} else if fs.lowercase {
		actualKey = strings.ToLower(key)
	}

	return fs.data[actualKey]
}

// convertValue 将扁平存储的数据转换为目标类型
func (fs *FlatStorage) convertValue(keyPath string, dst reflect.Value) error {
	if !dst.CanSet() && dst.Kind() != reflect.Ptr {
		return fmt.Errorf("destination is not settable")
	}

	// 处理目标为指针的情况
	if dst.Kind() == reflect.Ptr {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
			
			// 新分配的结构体指针需要设置默认值
			if fs.enableDefaults && dst.Type().Elem().Kind() == reflect.Struct {
				err := def.SetDefaults(dst.Interface())
				if err != nil {
					return fmt.Errorf("failed to set defaults for new pointer: %v", err)
				}
			}
		}
		return fs.convertValue(keyPath, dst.Elem())
	}

	// 根据目标类型处理转换
	switch dst.Kind() {
	case reflect.Map:
		return fs.convertToMap(keyPath, dst)
	case reflect.Slice:
		return fs.convertToSlice(keyPath, dst)
	case reflect.Struct:
		return fs.convertToStruct(keyPath, dst)
	case reflect.Interface:
		if dst.Type().NumMethod() == 0 {
			// 对于 interface{} 类型，直接获取值
			if keyPath != "" {
				value := fs.get(keyPath)
				if value != nil {
					dst.Set(reflect.ValueOf(value))
				}
			}
			return nil
		}
	default:
		// 基本类型，直接从扁平存储中获取值
		if keyPath != "" {
			value := fs.get(keyPath)
			if value != nil {
				return fs.convertBasicValue(value, dst)
			}
		}
	}

	return nil
}

// convertBasicValue 转换基本类型的值
func (fs *FlatStorage) convertBasicValue(src interface{}, dst reflect.Value) error {
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

	// 特殊类型转换：time.Duration 和 time.Time
	if err := fs.convertTimeTypes(srcValue, dst); err == nil {
		return nil
	} else if err.Error() != "not a time type" {
		return err
	}

	// 类型完全匹配
	if srcValue.Type().AssignableTo(dst.Type()) {
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

// convertToStruct 转换为结构体类型
func (fs *FlatStorage) convertToStruct(keyPath string, dst reflect.Value) error {
	dstType := dst.Type()

	for i := 0; i < dstType.NumField(); i++ {
		field := dstType.Field(i)
		fieldValue := dst.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// 获取字段名，优先使用 cfg tag，然后是 json/yaml/toml/ini tag
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

		// 应用大小写转换
		if fs.uppercase {
			fieldName = strings.ToUpper(fieldName)
		} else if fs.lowercase {
			fieldName = strings.ToLower(fieldName)
		}

		// 构建字段的完整路径
		var fieldPath string
		if keyPath == "" {
			fieldPath = fieldName
		} else {
			fieldPath = keyPath + fs.separator + fieldName
		}

		// 递归转换字段值
		if err := fs.convertValue(fieldPath, fieldValue); err != nil {
			return err
		}
	}

	return nil
}

// convertToSlice 转换为切片类型
func (fs *FlatStorage) convertToSlice(keyPath string, dst reflect.Value) error {
	// 查找所有以 keyPath 开头的索引项
	var maxIndex = -1
	prefix := keyPath
	if prefix != "" {
		prefix += fs.separator
	}

	// 扫描所有 key 找出最大索引
	actualPrefix := prefix
	if fs.uppercase {
		actualPrefix = strings.ToUpper(prefix)
	} else if fs.lowercase {
		actualPrefix = strings.ToLower(prefix)
	}
	
	for key := range fs.data {
		if strings.HasPrefix(key, actualPrefix) {
			remaining := strings.TrimPrefix(key, actualPrefix)
			// 查找第一个分隔符或结束
			parts := strings.SplitN(remaining, fs.separator, 2)
			if len(parts) > 0 {
				if index, err := strconv.Atoi(parts[0]); err == nil {
					if index > maxIndex {
						maxIndex = index
					}
				}
			}
		}
	}

	// 如果没有找到索引，返回空切片
	if maxIndex < 0 {
		dst.Set(reflect.MakeSlice(dst.Type(), 0, 0))
		return nil
	}

	// 创建切片
	length := maxIndex + 1
	dst.Set(reflect.MakeSlice(dst.Type(), length, length))

	// 填充切片元素
	for i := 0; i < length; i++ {
		var itemPath string
		if keyPath == "" {
			itemPath = strconv.Itoa(i)
		} else {
			itemPath = keyPath + fs.separator + strconv.Itoa(i)
		}

		dstItem := dst.Index(i)
		
		// 如果是结构体类型，为切片元素设置默认值
		if fs.enableDefaults && dstItem.Kind() == reflect.Struct {
			if err := def.SetDefaults(dstItem.Addr().Interface()); err != nil {
				return fmt.Errorf("failed to set defaults for slice element %d: %v", i, err)
			}
		}

		if err := fs.convertValue(itemPath, dstItem); err != nil {
			return err
		}
	}

	return nil
}

// convertToMap 转换为 map 类型
func (fs *FlatStorage) convertToMap(keyPath string, dst reflect.Value) error {
	if dst.IsNil() {
		dst.Set(reflect.MakeMap(dst.Type()))
	}

	// 查找所有以 keyPath 开头的键
	prefix := keyPath
	if prefix != "" {
		prefix += fs.separator
	}

	// 收集所有直接子键
	actualPrefix := prefix
	if fs.uppercase {
		actualPrefix = strings.ToUpper(prefix)
	} else if fs.lowercase {
		actualPrefix = strings.ToLower(prefix)
	}
	
	subKeys := make(map[string]bool)
	for key := range fs.data {
		if strings.HasPrefix(key, actualPrefix) {
			remaining := strings.TrimPrefix(key, actualPrefix)
			if remaining != "" {
				// 获取第一级子键
				parts := strings.SplitN(remaining, fs.separator, 2)
				subKeys[parts[0]] = true
			}
		}
	}

	// 处理每个子键
	for subKey := range subKeys {
		var subKeyPath string
		if keyPath == "" {
			subKeyPath = subKey
		} else {
			subKeyPath = keyPath + fs.separator + subKey
		}

		// 创建 map 值
		dstValue := reflect.New(dst.Type().Elem()).Elem()
		
		// 如果是结构体类型，为新创建的对象设置默认值
		if fs.enableDefaults && dstValue.Kind() == reflect.Struct {
			if err := def.SetDefaults(dstValue.Addr().Interface()); err != nil {
				return fmt.Errorf("failed to set defaults for new map value: %v", err)
			}
		}

		// 递归转换值
		if err := fs.convertValue(subKeyPath, dstValue); err != nil {
			return err
		}

		// 转换键类型
		keyValue := reflect.ValueOf(subKey)
		if !keyValue.Type().AssignableTo(dst.Type().Key()) {
			if keyValue.Type().ConvertibleTo(dst.Type().Key()) {
				keyValue = keyValue.Convert(dst.Type().Key())
			} else {
				return fmt.Errorf("cannot convert key %v to %v", keyValue.Type(), dst.Type().Key())
			}
		}

		dst.SetMapIndex(keyValue, dstValue)
	}

	return nil
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

func (fs *FlatStorage) Equals(other Storage) bool {
	if otherFs, ok := other.(*FlatStorage); ok {
		return fs.separator == otherFs.separator &&
			fs.enableDefaults == otherFs.enableDefaults &&
			fs.uppercase == otherFs.uppercase &&
			fs.lowercase == otherFs.lowercase &&
			fs.parent == otherFs.parent &&
			fs.prefix == otherFs.prefix &&
			reflect.DeepEqual(fs.data, otherFs.data)
	}
	return false
}

// parseKey 解析 key 字符串，支持点号和数组索引
func (ms *FlatStorage) parseKey(key string) []string {
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
