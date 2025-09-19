package ref

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

		// 检查是否需要进行 Storage 转换
		processedOptions, err := c.processStorageOptions(options)
		if err != nil {
			return nil, fmt.Errorf("failed to process storage options: %w", err)
		}

		args = []reflect.Value{reflect.ValueOf(processedOptions)}
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

// Convertable 接口定义，用于支持配置数据的自动转换
// 任何实现了此接口的类型都可以作为 options 参数传递给 New 方法
// 并自动转换为构造函数期望的参数类型
type Convertable interface {
	// ConvertTo 将配置数据转换为指定的对象类型
	// object 应该是指向目标对象的指针
	ConvertTo(object interface{}) error
}

// processStorageOptions 处理 Convertable 类型的 options，如果是 Convertable 类型则转换为目标类型
func (c *constructor) processStorageOptions(options any) (any, error) {
	// 检查 options 是否实现了 Convertable 接口
	if convertable, ok := options.(Convertable); ok {
		// 获取构造函数的参数类型
		funcType := c.newFunc.Type()
		if funcType.NumIn() != 1 {
			return nil, fmt.Errorf("unexpected constructor parameter count: %d", funcType.NumIn())
		}

		// 获取目标参数类型
		paramType := funcType.In(0)

		// 创建目标类型的实例
		var targetValue reflect.Value
		if paramType.Kind() == reflect.Ptr {
			// 如果是指针类型，创建新实例
			targetValue = reflect.New(paramType.Elem())
			// 使用 Convertable.ConvertTo 进行转换，传入指针
			if err := convertable.ConvertTo(targetValue.Interface()); err != nil {
				return nil, fmt.Errorf("failed to convert convertable to target type %v: %w", paramType, err)
			}
			return targetValue.Interface(), nil
		} else {
			// 如果是值类型，创建零值并获取其指针进行转换
			targetValue = reflect.New(paramType)
			// 使用 Convertable.ConvertTo 进行转换，传入指针
			if err := convertable.ConvertTo(targetValue.Interface()); err != nil {
				return nil, fmt.Errorf("failed to convert convertable to target type %v: %w", paramType, err)
			}
			// 返回解引用后的值
			return targetValue.Elem().Interface(), nil
		}
	}

	// 如果不是 Convertable 类型，直接返回原始 options
	return options, nil
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

type TypeOptions struct {
	Namespace string `cfg:"namespace"`
	Type      string `cfg:"type"`
	Options   any    `cfg:"options"`
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
