package refx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hatlonely/gox/cfg/storage"
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

// TestStorageSupport tests the Storage interface support in refx.New
func TestStorageSupport(t *testing.T) {
	namespace := "test-storage"

	// Register a constructor that expects a specific struct
	type DatabaseConfig struct {
		Host     string `cfg:"host"`
		Port     int    `cfg:"port"`
		Username string `cfg:"username"`
		Password string `cfg:"password"`
	}

	type Database struct {
		Config *DatabaseConfig
	}

	newDatabase := func(config *DatabaseConfig) *Database {
		return &Database{Config: config}
	}

	err := Register(namespace, "Database", newDatabase)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Create a MapStorage with configuration data
	configData := map[string]interface{}{
		"host":     "localhost",
		"port":     3306,
		"username": "root",
		"password": "secret",
	}
	mapStorage := storage.NewMapStorage(configData)

	// Test using Storage as options
	result, err := New(namespace, "Database", mapStorage)
	if err != nil {
		t.Fatalf("New() with Storage error = %v", err)
	}

	db, ok := result.(*Database)
	if !ok {
		t.Fatalf("New() result is not *Database type, got %T", result)
	}

	// Verify the configuration was properly converted
	if db.Config.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", db.Config.Host)
	}
	if db.Config.Port != 3306 {
		t.Errorf("Expected port 3306, got %d", db.Config.Port)
	}
	if db.Config.Username != "root" {
		t.Errorf("Expected username 'root', got '%s'", db.Config.Username)
	}
	if db.Config.Password != "secret" {
		t.Errorf("Expected password 'secret', got '%s'", db.Config.Password)
	}
}

// TestStorageSupportWithFlatStorage tests Storage support with FlatStorage
func TestStorageSupportWithFlatStorage(t *testing.T) {
	namespace := "test-flat-storage"

	// Register a constructor that expects a nested struct
	type ServerConfig struct {
		Database struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
		} `cfg:"database"`
		Redis struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
		} `cfg:"redis"`
	}

	type Server struct {
		Config *ServerConfig
	}

	newServer := func(config *ServerConfig) *Server {
		return &Server{Config: config}
	}

	err := Register(namespace, "Server", newServer)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Create a FlatStorage with flattened configuration data
	flatData := map[string]interface{}{
		"database.host": "db.example.com",
		"database.port": 5432,
		"redis.host":    "redis.example.com",
		"redis.port":    6379,
	}
	flatStorage := storage.NewFlatStorage(flatData)

	// Test using FlatStorage as options
	result, err := New(namespace, "Server", flatStorage)
	if err != nil {
		t.Fatalf("New() with FlatStorage error = %v", err)
	}

	server, ok := result.(*Server)
	if !ok {
		t.Fatalf("New() result is not *Server type, got %T", result)
	}

	// Verify the nested configuration was properly converted
	if server.Config.Database.Host != "db.example.com" {
		t.Errorf("Expected database host 'db.example.com', got '%s'", server.Config.Database.Host)
	}
	if server.Config.Database.Port != 5432 {
		t.Errorf("Expected database port 5432, got %d", server.Config.Database.Port)
	}
	if server.Config.Redis.Host != "redis.example.com" {
		t.Errorf("Expected redis host 'redis.example.com', got '%s'", server.Config.Redis.Host)
	}
	if server.Config.Redis.Port != 6379 {
		t.Errorf("Expected redis port 6379, got %d", server.Config.Redis.Port)
	}
}

// TestStorageSupportBackwardCompatibility ensures regular options still work
func TestStorageSupportBackwardCompatibility(t *testing.T) {
	namespace := "test-backward-compat"

	// Register a simple constructor
	type SimpleConfig struct {
		Name string
	}

	type SimpleObject struct {
		Config *SimpleConfig
	}

	newSimpleObject := func(config *SimpleConfig) *SimpleObject {
		return &SimpleObject{Config: config}
	}

	err := Register(namespace, "SimpleObject", newSimpleObject)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Test with regular (non-Storage) options - should work as before
	regularOptions := &SimpleConfig{Name: "regular"}
	result, err := New(namespace, "SimpleObject", regularOptions)
	if err != nil {
		t.Fatalf("New() with regular options error = %v", err)
	}

	obj, ok := result.(*SimpleObject)
	if !ok {
		t.Fatalf("New() result is not *SimpleObject type, got %T", result)
	}

	if obj.Config.Name != "regular" {
		t.Errorf("Expected name 'regular', got '%s'", obj.Config.Name)
	}
}

// TestStorageSupportWithNewT tests Storage support with NewT method
func TestStorageSupportWithNewT(t *testing.T) {
	// Define a config struct for testing
	type APIConfig struct {
		Endpoint string `cfg:"endpoint"`
		Timeout  int    `cfg:"timeout"`
		APIKey   string `cfg:"api_key"`
	}

	type APIClient struct {
		Config *APIConfig
	}

	newAPIClient := func(config *APIConfig) *APIClient {
		return &APIClient{Config: config}
	}

	// Register using RegisterT
	err := RegisterT[*APIClient](newAPIClient)
	if err != nil {
		t.Fatalf("RegisterT() error = %v", err)
	}

	// Create a MapStorage with configuration data
	configData := map[string]interface{}{
		"endpoint": "https://api.example.com",
		"timeout":  30,
		"api_key":  "secret-key-123",
	}
	mapStorage := storage.NewMapStorage(configData)

	// Test using NewT with Storage
	client, err := NewT[*APIClient](mapStorage)
	if err != nil {
		t.Fatalf("NewT() with Storage error = %v", err)
	}

	// Verify the configuration was properly converted
	if client.Config.Endpoint != "https://api.example.com" {
		t.Errorf("Expected endpoint 'https://api.example.com', got '%s'", client.Config.Endpoint)
	}
	if client.Config.Timeout != 30 {
		t.Errorf("Expected timeout 30, got %d", client.Config.Timeout)
	}
	if client.Config.APIKey != "secret-key-123" {
		t.Errorf("Expected api_key 'secret-key-123', got '%s'", client.Config.APIKey)
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

func TestDuplicateRegister(t *testing.T) {
	// 清理之前的注册（通过使用不同的namespace避免冲突）
	namespace := "test-duplicate"

	// 第一次注册
	err := Register(namespace, "Value", NewValue)
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	// 第二次注册相同的函数应该成功（跳过）
	err = Register(namespace, "Value", NewValue)
	if err != nil {
		t.Errorf("Second Register() with same function should not error, got: %v", err)
	}

	// 注册不同的函数应该失败
	err = Register(namespace, "Value", NewDefaultValue)
	if err == nil {
		t.Error("Register() with different function should return error")
	} else {
		expectedMsg := "constructor for test-duplicate:Value already registered with different function"
		if err.Error() != expectedMsg {
			t.Errorf("Register() error message = %v, want %v", err.Error(), expectedMsg)
		}
	}

	// 验证原始注册的函数仍然有效
	result, err := New(namespace, "Value", &Options{Name: "duplicate-test"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if value, ok := result.(*Value); ok {
		if value.Name != "duplicate-test" {
			t.Errorf("New() got name = %v, want %v", value.Name, "duplicate-test")
		}
	} else {
		t.Errorf("New() result is not *Value type")
	}
}

func TestDuplicateRegisterT(t *testing.T) {
	// 定义一个新的类型避免与其他测试冲突
	type TestValue struct {
		Name string
	}

	newTestValue := func(options *Options) *TestValue {
		if options == nil {
			return &TestValue{Name: "test-default"}
		}
		return &TestValue{Name: "test-" + options.Name}
	}

	anotherNewTestValue := func(options *Options) *TestValue {
		if options == nil {
			return &TestValue{Name: "another-default"}
		}
		return &TestValue{Name: "another-" + options.Name}
	}

	// 第一次注册
	err := RegisterT[*TestValue](newTestValue)
	if err != nil {
		t.Fatalf("First RegisterT() error = %v", err)
	}

	// 第二次注册相同的函数应该成功（跳过）
	err = RegisterT[*TestValue](newTestValue)
	if err != nil {
		t.Errorf("Second RegisterT() with same function should not error, got: %v", err)
	}

	// 注册不同的函数应该失败
	err = RegisterT[*TestValue](anotherNewTestValue)
	if err == nil {
		t.Error("RegisterT() with different function should return error")
	}

	// 验证原始注册的函数仍然有效
	result, err := NewT[*TestValue](&Options{Name: "registerT"})
	if err != nil {
		t.Fatalf("NewT() error = %v", err)
	}

	if result.Name != "test-registerT" {
		t.Errorf("NewT() got name = %v, want %v", result.Name, "test-registerT")
	}
}

func TestMustRegister(t *testing.T) {
	namespace := "test-must"

	// 测试正常注册不会panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustRegister() should not panic on valid registration, got: %v", r)
		}
	}()

	MustRegister(namespace, "Value", NewValue)

	// 验证注册成功
	result, err := New(namespace, "Value", &Options{Name: "must-test"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if value, ok := result.(*Value); ok {
		if value.Name != "must-test" {
			t.Errorf("New() got name = %v, want %v", value.Name, "must-test")
		}
	} else {
		t.Errorf("New() result is not *Value type")
	}
}

func TestMustRegisterPanic(t *testing.T) {
	namespace := "test-must-panic"

	// 先注册一个构造函数
	MustRegister(namespace, "Value", NewValue)

	// 测试重复注册不同函数会panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRegister() should panic when registering different function")
		} else {
			// 验证panic信息包含预期的错误信息
			errorStr := fmt.Sprintf("%v", r)
			expected := "constructor for test-must-panic:Value already registered with different function"
			if errorStr != expected {
				t.Errorf("MustRegister() panic message = %v, want %v", errorStr, expected)
			}
		}
	}()

	// 这应该会panic
	MustRegister(namespace, "Value", NewDefaultValue)
}

func TestMustRegisterT(t *testing.T) {
	// 定义新类型避免与其他测试冲突
	type MustTestValue struct {
		Name string
	}

	newMustTestValue := func(options *Options) *MustTestValue {
		if options == nil {
			return &MustTestValue{Name: "must-default"}
		}
		return &MustTestValue{Name: "must-" + options.Name}
	}

	// 测试正常注册不会panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustRegisterT() should not panic on valid registration, got: %v", r)
		}
	}()

	MustRegisterT[*MustTestValue](newMustTestValue)

	// 验证注册成功
	result, err := NewT[*MustTestValue](&Options{Name: "registerT"})
	if err != nil {
		t.Fatalf("NewT() error = %v", err)
	}

	if result.Name != "must-registerT" {
		t.Errorf("NewT() got name = %v, want %v", result.Name, "must-registerT")
	}
}

func TestMustRegisterTPanic(t *testing.T) {
	// 定义新类型避免与其他测试冲突
	type MustPanicTestValue struct {
		Name string
	}

	newFunc1 := func(options *Options) *MustPanicTestValue {
		return &MustPanicTestValue{Name: "func1"}
	}

	newFunc2 := func(options *Options) *MustPanicTestValue {
		return &MustPanicTestValue{Name: "func2"}
	}

	// 先注册一个构造函数
	MustRegisterT[*MustPanicTestValue](newFunc1)

	// 测试重复注册不同函数会panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRegisterT() should panic when registering different function")
		}
	}()

	// 这应该会panic
	MustRegisterT[*MustPanicTestValue](newFunc2)
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
