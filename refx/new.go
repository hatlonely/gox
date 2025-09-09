package refx

import (
	"fmt"
	"reflect"
	"sync"
)

type constructor struct {
	originalFunc any
	newFunc      reflect.Value
	hasOptions   bool
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
		originalFunc: newFunc,
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

func isSameFunc(func1, func2 any) bool {
	if func1 == nil || func2 == nil {
		return func1 == func2
	}
	
	v1 := reflect.ValueOf(func1)
	v2 := reflect.ValueOf(func2)
	
	// 比较函数指针
	return v1.Pointer() == v2.Pointer()
}

func Register(namespace string, type_ string, newFunc any) error {
	key := namespace + ":" + type_
	
	// 检查是否已经注册
	if existingValue, ok := nameConstructorMap.Load(key); ok {
		if existingConstructor, ok := existingValue.(*constructor); ok {
			// 检查是否是相同的函数
			if isSameFunc(existingConstructor.originalFunc, newFunc) {
				// 相同函数，跳过注册
				return nil
			} else {
				// 不同函数，返回错误
				return fmt.Errorf("constructor for %s:%s already registered with different function", namespace, type_)
			}
		}
	}
	
	constructor, err := newConstructor(newFunc)
	if err != nil {
		return fmt.Errorf("failed to create constructor: %w", err)
	}
	
	nameConstructorMap.Store(key, constructor)
	return nil
}

func RegisterT[T any](newFunc any) error {
	var t T
	tType := reflect.TypeOf(t)
	
	// 如果是指针类型，获取其元素类型
	for tType.Kind() == reflect.Ptr {
		tType = tType.Elem()
	}
	
	// 从类型中提取包名和类型名作为默认的namespace和type
	pkgPath := tType.PkgPath()
	typeName := tType.Name()
	
	if pkgPath == "" || typeName == "" {
		return fmt.Errorf("cannot determine package path or type name for type %T", t)
	}
	
	return Register(pkgPath, typeName, newFunc)
}

func MustRegister(namespace string, type_ string, newFunc any) {
	err := Register(namespace, type_, newFunc)
	if err != nil {
		panic(err)
	}
}

func MustRegisterT[T any](newFunc any) {
	err := RegisterT[T](newFunc)
	if err != nil {
		panic(err)
	}
}

func New(namespace string, type_ string, options any) (any, error) {
	key := namespace + ":" + type_
	value, ok := nameConstructorMap.Load(key)
	if !ok {
		return nil, fmt.Errorf("constructor not found for %s:%s", namespace, type_)
	}
	
	constructor, ok := value.(*constructor)
	if !ok {
		return nil, fmt.Errorf("invalid constructor type for %s:%s", namespace, type_)
	}
	
	return constructor.new(options)
}

func NewT[T any](options any) (T, error) {
	var t T
	tType := reflect.TypeOf(t)
	
	// 如果是指针类型，获取其元素类型
	for tType.Kind() == reflect.Ptr {
		tType = tType.Elem()
	}
	
	// 从类型中提取包名和类型名作为默认的namespace和type
	pkgPath := tType.PkgPath()
	typeName := tType.Name()
	
	if pkgPath == "" || typeName == "" {
		return t, fmt.Errorf("cannot determine package path or type name for type %T", t)
	}
	
	obj, err := New(pkgPath, typeName, options)
	if err != nil {
		return t, err
	}
	
	result, ok := obj.(T)
	if !ok {
		return t, fmt.Errorf("created object is not of type %T", t)
	}
	
	return result, nil
}
