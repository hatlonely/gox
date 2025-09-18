package validator

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

// ValidateStruct 使用 validator 校验结构体
// 这是一个通用的结构体校验函数，提供了比直接使用 validator.Struct() 更好的容错性和类型检查
func ValidateStruct(object interface{}) error {
	if object == nil {
		return nil
	}

	// 只对结构体进行校验
	rv := reflect.ValueOf(object)
	if !rv.IsValid() {
		return nil
	}

	// 处理指针类型，检查是否有任何层级的 nil 指针
	currentValue := rv
	for currentValue.Kind() == reflect.Ptr {
		if currentValue.IsNil() {
			return nil
		}
		currentValue = currentValue.Elem()
	}

	if currentValue.Kind() != reflect.Struct {
		return nil
	}

	// 跳过对某些内置类型的校验，如 time.Time
	rt := currentValue.Type()
	if rt.PkgPath() == "time" && rt.Name() == "Time" {
		return nil
	}

	// 只有当目标是一个有效的结构体实例时才进行校验
	// 对于双重指针或更深层的指针，需要确保最终指向的是一个有效的结构体
	if rv.Kind() == reflect.Ptr {
		// 如果是指针，检查指向的值
		elem := rv.Elem()
		if !elem.IsValid() || (elem.Kind() == reflect.Ptr && elem.IsNil()) {
			return nil
		}
		// 传递实际的结构体值进行校验
		if elem.Kind() == reflect.Ptr && !elem.IsNil() {
			return ValidateStruct(elem.Interface())
		} else if elem.Kind() == reflect.Struct {
			validate := validator.New()
			return validate.Struct(elem.Interface())
		}
	}

	// 对于非指针的结构体
	validate := validator.New()
	return validate.Struct(object)
}