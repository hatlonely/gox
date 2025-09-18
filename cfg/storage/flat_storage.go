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
		data:           data,
		separator:      ".",
		enableDefaults: true,
	}
}

func (fs *FlatStorage) WithDefaults(enable bool) *FlatStorage {
	if fs == nil {
		return nil
	}
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
		parent:    fs,
		prefix:    strings.Join(keys, fs.separator),
		separator: fs.separator,
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

// prepareKey 构建完整的键路径并应用大小写转换，同时返回数据源
func (fs *FlatStorage) prepareKey(key string) (dataSource map[string]interface{}, actualKey string) {
	// 构建完整的键路径
	fullKey := key
	if fs.prefix != "" {
		if key != "" {
			fullKey = fs.prefix + fs.separator + key
		} else {
			fullKey = fs.prefix
		}
	}

	// 获取数据源和大小写配置
	var useUppercase, useLowercase bool
	if fs.parent != nil {
		dataSource = fs.parent.data
		useUppercase = fs.parent.uppercase
		useLowercase = fs.parent.lowercase
	} else {
		dataSource = fs.data
		useUppercase = fs.uppercase
		useLowercase = fs.lowercase
	}

	// 应用大小写转换
	if useUppercase {
		actualKey = strings.ToUpper(fullKey)
	} else if useLowercase {
		actualKey = strings.ToLower(fullKey)
	} else {
		actualKey = fullKey
	}

	return dataSource, actualKey
}

func (fs *FlatStorage) get(key string) interface{} {
	dataSource, actualKey := fs.prepareKey(key)
	return dataSource[actualKey]
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

	// 先检查时间类型（在检查 struct 之前）
	value := fs.get(keyPath)
	if value != nil {
		// 尝试时间类型转换
		if err := fs.convertTimeTypes(reflect.ValueOf(value), dst); err == nil {
			return nil
		} else if err.Error() != "not a time type" {
			return err
		}
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
			value := fs.get(keyPath)
			if value != nil {
				dst.Set(reflect.ValueOf(value))
			}
			return nil
		}
	default:
		// 基本类型，直接从扁平存储中获取值
		value := fs.get(keyPath)
		if value != nil {
			return fs.convertBasicValue(value, dst)
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

	// 构建完整的前缀路径并应用大小写转换
	dataSource, actualPrefix := fs.prepareKey(keyPath)
	if actualPrefix != "" {
		actualPrefix += fs.separator
	}

	for key := range dataSource {
		if strings.HasPrefix(key, actualPrefix) {
			remaining := strings.TrimPrefix(key, actualPrefix)
			// 如果remaining以分隔符开头，去掉它
			remaining = strings.TrimPrefix(remaining, fs.separator)
			// 查找第一个分隔符或结束
			parts := strings.SplitN(remaining, fs.separator, 2)
			if len(parts) > 0 && parts[0] != "" {
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

	// 构建完整的前缀路径并应用大小写转换
	dataSource, finalPrefix := fs.prepareKey(keyPath)
	if finalPrefix != "" {
		finalPrefix += fs.separator
	}

	// 检查目标值类型，如果是interface{}，则保留完整键名
	isInterfaceValue := dst.Type().Elem().Kind() == reflect.Interface && dst.Type().Elem().NumMethod() == 0

	keyValueMap := make(map[string]interface{})
	for key, value := range dataSource {
		if strings.HasPrefix(key, finalPrefix) {
			remaining := strings.TrimPrefix(key, finalPrefix)
			if remaining != "" {
				if isInterfaceValue {
					// 对于interface{}类型，保留完整的剩余键名
					keyValueMap[remaining] = value
				} else {
					// 对于其他类型，只取第一级键名
					parts := strings.SplitN(remaining, fs.separator, 2)
					if _, exists := keyValueMap[parts[0]]; !exists {
						keyValueMap[parts[0]] = nil // 占位符，后续会被正确值替换
					}
				}
			}
		}
	}

	// 处理每个键值对
	for mapKey, mapValue := range keyValueMap {
		if isInterfaceValue {
			// 对于interface{}类型，直接设置值
			keyValue := reflect.ValueOf(mapKey)
			if !keyValue.Type().AssignableTo(dst.Type().Key()) {
				if keyValue.Type().ConvertibleTo(dst.Type().Key()) {
					keyValue = keyValue.Convert(dst.Type().Key())
				} else {
					return fmt.Errorf("cannot convert key %v to %v", keyValue.Type(), dst.Type().Key())
				}
			}

			dst.SetMapIndex(keyValue, reflect.ValueOf(mapValue))
		} else {
			// 对于其他类型，递归转换
			var subKeyPath string
			if keyPath == "" {
				subKeyPath = mapKey
			} else {
				subKeyPath = keyPath + fs.separator + mapKey
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
			keyValue := reflect.ValueOf(mapKey)
			if !keyValue.Type().AssignableTo(dst.Type().Key()) {
				if keyValue.Type().ConvertibleTo(dst.Type().Key()) {
					keyValue = keyValue.Convert(dst.Type().Key())
				} else {
					return fmt.Errorf("cannot convert key %v to %v", keyValue.Type(), dst.Type().Key())
				}
			}

			dst.SetMapIndex(keyValue, dstValue)
		}
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
		// 比较基本配置
		if fs.separator != otherFs.separator ||
			fs.enableDefaults != otherFs.enableDefaults ||
			fs.uppercase != otherFs.uppercase ||
			fs.lowercase != otherFs.lowercase ||
			fs.prefix != otherFs.prefix {
			return false
		}

		// 比较数据源：如果都有 parent，比较 parent 的数据；否则比较自身数据
		var fsData, otherData map[string]interface{}
		if fs.parent != nil {
			fsData = fs.parent.data
		} else {
			fsData = fs.data
		}
		if otherFs.parent != nil {
			otherData = otherFs.parent.data
		} else {
			otherData = otherFs.data
		}

		return reflect.DeepEqual(fsData, otherData)
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
