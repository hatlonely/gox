package cfg

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// SetDefaults 为结构体设置默认值，基于 def tag
func SetDefaults(object interface{}) error {
	if object == nil {
		return fmt.Errorf("object cannot be nil")
	}

	rv := reflect.ValueOf(object)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("object must be a pointer")
	}

	if rv.IsNil() {
		return fmt.Errorf("object cannot be nil")
	}

	return setDefaults(rv.Elem())
}

// setDefaults 递归地为结构体字段设置默认值
func setDefaults(rv reflect.Value) error {
	if !rv.IsValid() {
		return nil
	}

	// 处理指针类型
	if rv.Kind() == reflect.Ptr {
		// 如果是空指针，需要分配内存
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		return setDefaults(rv.Elem())
	}

	// 只处理结构体类型
	if rv.Kind() != reflect.Struct {
		return nil
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fieldValue := rv.Field(i)

		// 跳过不可设置的字段
		if !fieldValue.CanSet() {
			continue
		}

		// 获取 def tag
		defTag := field.Tag.Get("def")

		// 处理嵌套结构体（递归处理）
		if fieldValue.Kind() == reflect.Struct || 
		   (fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct) {
			if err := setDefaults(fieldValue); err != nil {
				return fmt.Errorf("failed to set defaults for field %s: %v", field.Name, err)
			}
		}

		// 如果没有 def tag，跳过
		if defTag == "" {
			continue
		}

		// 只有在字段为零值时才设置默认值
		if !isZeroValue(fieldValue) {
			continue
		}

		// 如果字段是指针且为 nil，需要分配内存
		if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
			fieldValue = fieldValue.Elem()
		}

		// 设置默认值
		if err := setDefaultValue(fieldValue, defTag); err != nil {
			return fmt.Errorf("failed to set default value for field %s: %v", field.Name, err)
		}
	}

	return nil
}

// isZeroValue 检查值是否为零值
func isZeroValue(rv reflect.Value) bool {
	return rv.IsZero()
}

// setDefaultValue 根据字段类型和 def tag 设置默认值
func setDefaultValue(rv reflect.Value, defValue string) error {
	switch rv.Kind() {
	case reflect.String:
		rv.SetString(defValue)
		return nil

	case reflect.Bool:
		val, err := strconv.ParseBool(defValue)
		if err != nil {
			return fmt.Errorf("invalid bool value %q: %v", defValue, err)
		}
		rv.SetBool(val)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 特殊处理 time.Duration
		if rv.Type() == reflect.TypeOf(time.Duration(0)) {
			return setDurationDefault(rv, defValue)
		}
		val, err := strconv.ParseInt(defValue, 0, rv.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid int value %q: %v", defValue, err)
		}
		rv.SetInt(val)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(defValue, 0, rv.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid uint value %q: %v", defValue, err)
		}
		rv.SetUint(val)
		return nil

	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(defValue, rv.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid float value %q: %v", defValue, err)
		}
		rv.SetFloat(val)
		return nil

	case reflect.Struct:
		// 特殊处理 time.Time
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			return setTimeDefault(rv, defValue)
		}

	case reflect.Slice:
		return setSliceDefault(rv, defValue)

	case reflect.Map:
		// Map 类型的默认值处理比较复杂，暂不支持
		return fmt.Errorf("map default values are not supported")
	}

	return fmt.Errorf("unsupported type %v", rv.Type())
}

// setDurationDefault 设置 time.Duration 类型的默认值
func setDurationDefault(rv reflect.Value, defValue string) error {
	duration, err := time.ParseDuration(defValue)
	if err != nil {
		// 尝试解析为纳秒数
		if val, numErr := strconv.ParseInt(defValue, 10, 64); numErr == nil {
			duration = time.Duration(val)
		} else {
			return fmt.Errorf("invalid duration value %q: %v", defValue, err)
		}
	}
	rv.Set(reflect.ValueOf(duration))
	return nil
}

// setTimeDefault 设置 time.Time 类型的默认值
func setTimeDefault(rv reflect.Value, defValue string) error {
	// 支持多种时间格式
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"15:04:05",
	}

	// 首先尝试字符串解析
	for _, format := range formats {
		if t, err := time.Parse(format, defValue); err == nil {
			rv.Set(reflect.ValueOf(t))
			return nil
		}
	}

	// 尝试解析为 Unix 时间戳
	if timestamp, err := strconv.ParseInt(defValue, 10, 64); err == nil {
		t := time.Unix(timestamp, 0)
		rv.Set(reflect.ValueOf(t))
		return nil
	}

	// 尝试解析为浮点数时间戳（支持小数）
	if timestamp, err := strconv.ParseFloat(defValue, 64); err == nil {
		seconds := int64(timestamp)
		nanoseconds := int64((timestamp - float64(seconds)) * 1e9)
		t := time.Unix(seconds, nanoseconds)
		rv.Set(reflect.ValueOf(t))
		return nil
	}

	return fmt.Errorf("invalid time value %q", defValue)
}

// setSliceDefault 设置切片类型的默认值
func setSliceDefault(rv reflect.Value, defValue string) error {
	// 简单支持逗号分隔的字符串列表
	if rv.Type().Elem().Kind() == reflect.String {
		parts := strings.Split(defValue, ",")
		slice := reflect.MakeSlice(rv.Type(), len(parts), len(parts))
		for i, part := range parts {
			slice.Index(i).SetString(strings.TrimSpace(part))
		}
		rv.Set(slice)
		return nil
	}

	// 对于其他类型的切片，支持逗号分隔的值
	parts := strings.Split(defValue, ",")
	slice := reflect.MakeSlice(rv.Type(), len(parts), len(parts))
	
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if err := setDefaultValue(slice.Index(i), part); err != nil {
			return fmt.Errorf("failed to set slice element %d: %v", i, err)
		}
	}
	
	rv.Set(slice)
	return nil
}