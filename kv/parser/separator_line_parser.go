package parser

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type SeparatorLineParserOptions struct {
	Separator string `cfg:"separator" def:"\t"`
}

type SeparatorLineParser[K, V any] struct {
	separator string
}

func NewSeparatorLineParserWithOptions[K, V any](options *SeparatorLineParserOptions) (*SeparatorLineParser[K, V], error) {
	return &SeparatorLineParser[K, V]{
		separator: options.Separator,
	}, nil
}

func (p *SeparatorLineParser[K, V]) Parse(line string) (ChangeType, K, V, error) {
	var zeroK K
	var zeroV V
	
	parts := strings.Split(line, p.separator)
	
	if len(parts) < 2 {
		return ChangeTypeUnknown, zeroK, zeroV, nil
	}
	
	keyStr := parts[0]
	valueStr := parts[1]
	changeType := ChangeTypeAdd
	
	if len(parts) >= 3 && parts[2] != "" {
		if val, err := strconv.Atoi(parts[2]); err == nil {
			changeType = ChangeType(val)
		} else {
			switch strings.ToLower(parts[2]) {
			case "add":
				changeType = ChangeTypeAdd
			case "update":
				changeType = ChangeTypeUpdate
			case "delete":
				changeType = ChangeTypeDelete
			default:
				changeType = ChangeTypeAdd
			}
		}
	}
	
	key, err := parseValue[K](keyStr)
	if err != nil {
		return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to parse key: %w", err)
	}
	
	value, err := parseValue[V](valueStr)
	if err != nil {
		return ChangeTypeUnknown, zeroK, zeroV, fmt.Errorf("failed to parse value: %w", err)
	}
	
	return changeType, key, value, nil
}

func parseValue[T any](str string) (T, error) {
	var result T
	var zeroT T
	
	rt := reflect.TypeOf(result)
	
	switch rt.Kind() {
	case reflect.String:
		if v, ok := any(str).(T); ok {
			return v, nil
		}
	case reflect.Int:
		if val, err := strconv.Atoi(str); err == nil {
			if v, ok := any(val).(T); ok {
				return v, nil
			}
		}
	case reflect.Int8:
		if val, err := strconv.ParseInt(str, 10, 8); err == nil {
			if v, ok := any(int8(val)).(T); ok {
				return v, nil
			}
		}
	case reflect.Int16:
		if val, err := strconv.ParseInt(str, 10, 16); err == nil {
			if v, ok := any(int16(val)).(T); ok {
				return v, nil
			}
		}
	case reflect.Int32:
		if val, err := strconv.ParseInt(str, 10, 32); err == nil {
			if v, ok := any(int32(val)).(T); ok {
				return v, nil
			}
		}
	case reflect.Int64:
		if val, err := strconv.ParseInt(str, 10, 64); err == nil {
			if v, ok := any(val).(T); ok {
				return v, nil
			}
		}
	case reflect.Uint:
		if val, err := strconv.ParseUint(str, 10, 0); err == nil {
			if v, ok := any(uint(val)).(T); ok {
				return v, nil
			}
		}
	case reflect.Uint8:
		if val, err := strconv.ParseUint(str, 10, 8); err == nil {
			if v, ok := any(uint8(val)).(T); ok {
				return v, nil
			}
		}
	case reflect.Uint16:
		if val, err := strconv.ParseUint(str, 10, 16); err == nil {
			if v, ok := any(uint16(val)).(T); ok {
				return v, nil
			}
		}
	case reflect.Uint32:
		if val, err := strconv.ParseUint(str, 10, 32); err == nil {
			if v, ok := any(uint32(val)).(T); ok {
				return v, nil
			}
		}
	case reflect.Uint64:
		if val, err := strconv.ParseUint(str, 10, 64); err == nil {
			if v, ok := any(val).(T); ok {
				return v, nil
			}
		}
	case reflect.Float32:
		if val, err := strconv.ParseFloat(str, 32); err == nil {
			if v, ok := any(float32(val)).(T); ok {
				return v, nil
			}
		}
	case reflect.Float64:
		if val, err := strconv.ParseFloat(str, 64); err == nil {
			if v, ok := any(val).(T); ok {
				return v, nil
			}
		}
	case reflect.Bool:
		if val, err := strconv.ParseBool(str); err == nil {
			if v, ok := any(val).(T); ok {
				return v, nil
			}
		}
	default:
		if err := json.Unmarshal([]byte(str), &result); err == nil {
			return result, nil
		}
		return zeroT, fmt.Errorf("unsupported type conversion from string to %T", result)
	}
	
	return zeroT, fmt.Errorf("failed to convert string %q to type %T", str, result)
}
