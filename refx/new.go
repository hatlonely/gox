package refx

import (
	"fmt"
	"reflect"
	"sync"
)

type constructor struct {
	newFunc reflect.Value
	hasOptions bool
	returnsError bool
}

func newConstructor(newFunc any) (*constructor, error) {
	funcValue := reflect.ValueOf(newFunc)
	if funcValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("newFunc must be a function")
	}

	funcType := funcValue.Type()
	numIn := funcType.NumIn()
	numOut := funcType.NumOut()

	// 验证参数数量：0个或1个参数
	if numIn != 0 && numIn != 1 {
		return nil, fmt.Errorf("newFunc must have 0 or 1 input parameters, got %d", numIn)
	}

	// 验证返回值数量：1个或2个返回值
	if numOut != 1 && numOut != 2 {
		return nil, fmt.Errorf("newFunc must have 1 or 2 return values, got %d", numOut)
	}

	hasOptions := numIn == 1
	returnsError := false

	// 如果有2个返回值，第二个必须是error类型
	if numOut == 2 {
		errorInterface := reflect.TypeOf((*error)(nil)).Elem()
		if !funcType.Out(1).Implements(errorInterface) {
			return nil, fmt.Errorf("second return value must be error type")
		}
		returnsError = true
	}

	return &constructor{
		newFunc:      funcValue,
		hasOptions:   hasOptions,
		returnsError: returnsError,
	}, nil
}

func (c *constructor) new(options any) (any, error) {
	var args []reflect.Value

	// 根据构造函数是否需要参数来准备调用参数
	if c.hasOptions {
		if options == nil {
			return nil, fmt.Errorf("constructor requires options but got nil")
		}
		args = []reflect.Value{reflect.ValueOf(options)}
	} else {
		args = []reflect.Value{}
	}

	// 调用构造函数
	results := c.newFunc.Call(args)

	// 处理返回值
	if c.returnsError {
		// 有错误返回值的情况
		obj := results[0].Interface()
		errResult := results[1].Interface()
		
		if errResult != nil {
			if err, ok := errResult.(error); ok {
				return nil, err
			}
			return nil, fmt.Errorf("second return value is not an error")
		}
		
		return obj, nil
	} else {
		// 只有对象返回值的情况
		return results[0].Interface(), nil
	}
}

var nameConstructorMap sync.Map

func Register(namespace string, type_ string, constructor any) error {
	return nil
}

func New(namespace string, type_ string, options any) (any, error) {
	return nil, nil
}

func NewT[T any](options any) (T, error) {
	var t T
	return t, nil
}
