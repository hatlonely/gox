package refx

import (
	"errors"
	"testing"
)

type Value struct {
	Name string
}

type Options struct {
	Name string
}

// 测试构造函数：接收options参数，返回对象和错误
func NewValue(options *Options) (*Value, error) {
	if options == nil {
		return nil, errors.New("options cannot be nil")
	}
	if options.Name == "" {
		return nil, errors.New("name cannot be empty")
	}
	return &Value{Name: options.Name}, nil
}

// 测试构造函数：不接收参数，只返回对象
func NewDefaultValue() *Value {
	return &Value{Name: "default"}
}

// 测试构造函数：接收options参数，只返回对象
func NewSimpleValue(options *Options) *Value {
	if options == nil {
		return &Value{Name: "nil-options"}
	}
	return &Value{Name: options.Name}
}

// 测试构造函数：不接收参数，返回对象和错误
func NewErrorValue() (*Value, error) {
	return &Value{Name: "error-test"}, nil
}

func TestRegisterAndNew(t *testing.T) {
	// 注册构造函数
	err := Register("test", "Value", NewValue)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	err = Register("test", "DefaultValue", NewDefaultValue)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// 测试 New 方法
	tests := []struct {
		name      string
		namespace string
		type_     string
		options   any
		wantErr   bool
		expected  string
	}{
		{
			name:      "Create Value with options",
			namespace: "test",
			type_:     "Value",
			options:   &Options{Name: "registered"},
			wantErr:   false,
			expected:  "registered",
		},
		{
			name:      "Create DefaultValue without options",
			namespace: "test",
			type_:     "DefaultValue",
			options:   nil,
			wantErr:   false,
			expected:  "default",
		},
		{
			name:      "Not found constructor",
			namespace: "test",
			type_:     "NotExist",
			options:   nil,
			wantErr:   true,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := New(tt.namespace, tt.type_, tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if value, ok := result.(*Value); ok {
					if value.Name != tt.expected {
						t.Errorf("New() got name = %v, want %v", value.Name, tt.expected)
					}
				} else {
					t.Errorf("New() result is not *Value type")
				}
			}
		})
	}
}

func TestNewT(t *testing.T) {
	// 注册构造函数，使用完整的包路径作为namespace
	err := Register("github.com/hatlonely/gox/refx", "Value", NewValue)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// 测试 NewT 方法
	result, err := NewT[*Value](&Options{Name: "generic"})
	if err != nil {
		t.Fatalf("NewT() error = %v", err)
	}

	if result.Name != "generic" {
		t.Errorf("NewT() got name = %v, want %v", result.Name, "generic")
	}
}

func TestRegisterT(t *testing.T) {
	// 使用 RegisterT 注册构造函数（只注册一个避免覆盖）
	err := RegisterT[*Value](NewValue)
	if err != nil {
		t.Fatalf("RegisterT() error = %v", err)
	}

	// 测试通过 NewT 创建对象
	result1, err := NewT[*Value](&Options{Name: "registerT-test"})
	if err != nil {
		t.Fatalf("NewT() error = %v", err)
	}

	if result1.Name != "registerT-test" {
		t.Errorf("NewT() got name = %v, want %v", result1.Name, "registerT-test")
	}

	// 测试通过 New 创建对象  
	result2, err := New("github.com/hatlonely/gox/refx", "Value", &Options{Name: "new-test"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if value, ok := result2.(*Value); ok {
		if value.Name != "new-test" {
			t.Errorf("New() got name = %v, want %v", value.Name, "new-test")
		}
	} else {
		t.Errorf("New() result is not *Value type")
	}

	// 测试注册另一种类型的构造函数
	type AnotherValue struct {
		Name string
	}

	newAnotherValue := func(options *Options) *AnotherValue {
		if options == nil {
			return &AnotherValue{Name: "another-default"}
		}
		return &AnotherValue{Name: "another-" + options.Name}
	}

	err = RegisterT[*AnotherValue](newAnotherValue)
	if err != nil {
		t.Fatalf("RegisterT[AnotherValue]() error = %v", err)
	}

	result3, err := NewT[*AnotherValue](&Options{Name: "test"})
	if err != nil {
		t.Fatalf("NewT[AnotherValue]() error = %v", err)
	}

	if result3.Name != "another-test" {
		t.Errorf("NewT[AnotherValue]() got name = %v, want %v", result3.Name, "another-test")
	}
}

func TestNewConstructor(t *testing.T) {
	tests := []struct {
		name     string
		newFunc  any
		options  any
		wantErr  bool
		expected string
	}{
		{
			name:     "NewValue with options",
			newFunc:  NewValue,
			options:  &Options{Name: "test"},
			wantErr:  false,
			expected: "test",
		},
		{
			name:     "NewValue with nil options",
			newFunc:  NewValue,
			options:  (*Options)(nil),
			wantErr:  true,
			expected: "",
		},
		{
			name:     "NewDefaultValue without options",
			newFunc:  NewDefaultValue,
			options:  nil,
			wantErr:  false,
			expected: "default",
		},
		{
			name:     "NewSimpleValue with options",
			newFunc:  NewSimpleValue,
			options:  &Options{Name: "simple"},
			wantErr:  false,
			expected: "simple",
		},
		{
			name:     "NewErrorValue without options",
			newFunc:  NewErrorValue,
			options:  nil,
			wantErr:  false,
			expected: "error-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constructor, err := newConstructor(tt.newFunc)
			if err != nil {
				t.Fatalf("newConstructor() error = %v", err)
			}

			result, err := constructor.new(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("constructor.new() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if value, ok := result.(*Value); ok {
					if value.Name != tt.expected {
						t.Errorf("constructor.new() got name = %v, want %v", value.Name, tt.expected)
					}
				} else {
					t.Errorf("constructor.new() result is not *Value type")
				}
			}
		})
	}
}