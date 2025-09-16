package storage

import (
	"reflect"
	"testing"
	"time"
)

// 测试数据集
var testData = map[string]interface{}{
	"database": map[string]interface{}{
		"host": "localhost",
		"port": 3306,
		"connections": []interface{}{
			map[string]interface{}{
				"name": "primary",
				"user": "admin",
			},
			map[string]interface{}{
				"name": "secondary", 
				"user": "readonly",
			},
		},
	},
	"servers": []interface{}{"server1", "server2"},
	"config": map[string]interface{}{
		"timeout":    "30s",
		"created_at": "2023-12-25T15:30:45Z",
		"enabled":    true,
	},
}

// TestMapStorage_Creation 测试 MapStorage 的各种创建方式
func TestMapStorage_Creation(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		defaults bool
	}{
		{
			name:     "创建带默认值的 MapStorage",
			data:     testData,
			defaults: true,
		},
		{
			name:     "创建不带默认值的 MapStorage",
			data:     testData,
			defaults: false,
		},
		{
			name:     "创建空数据的 MapStorage",
			data:     nil,
			defaults: true,
		},
		{
			name:     "创建空 map 的 MapStorage",
			data:     map[string]interface{}{},
			defaults: true,
		},
		{
			name:     "创建简单类型的 MapStorage",
			data:     "simple string",
			defaults: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var storage *MapStorage
			if tt.defaults {
				storage = NewMapStorage(tt.data)
			} else {
				storage = NewMapStorageWithoutDefaults(tt.data)
			}

			if storage == nil {
				t.Fatal("Expected storage to be created, got nil")
			}

			// 测试 Data() 方法
			if !deepEqual(storage.Data(), tt.data) {
				t.Errorf("Expected Data() to return original data")
			}

			// 测试 enableDefaults 设置
			expectedDefaults := tt.defaults
			if storage.enableDefaults != expectedDefaults {
				t.Errorf("Expected enableDefaults to be %v, got %v", expectedDefaults, storage.enableDefaults)
			}
		})
	}
}

// TestMapStorage_WithDefaults 测试默认值开关功能
func TestMapStorage_WithDefaults(t *testing.T) {
	storage := NewMapStorage(testData)
	
	// 测试开启默认值
	storage = storage.WithDefaults(true)
	if !storage.enableDefaults {
		t.Error("Expected enableDefaults to be true after WithDefaults(true)")
	}
	
	// 测试关闭默认值
	storage = storage.WithDefaults(false)
	if storage.enableDefaults {
		t.Error("Expected enableDefaults to be false after WithDefaults(false)")
	}
	
	// 测试 nil storage 的处理
	var nilStorage *MapStorage = nil
	result := nilStorage.WithDefaults(true)
	if result != nil {
		t.Error("Expected WithDefaults on nil storage to return nil")
	}
}

// TestMapStorage_Data 测试数据获取功能
func TestMapStorage_Data(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{"map 数据", testData},
		{"nil 数据", nil},
		{"字符串数据", "test"},
		{"数字数据", 123},
		{"切片数据", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMapStorage(tt.data)
			if !deepEqual(storage.Data(), tt.data) {
				t.Errorf("Expected Data() to return %v, got %v", tt.data, storage.Data())
			}
		})
	}
}

// TestMapStorage_Sub_Basic 测试基本路径访问功能
func TestMapStorage_Sub_Basic(t *testing.T) {
	storage := NewMapStorage(testData)

	tests := []struct {
		name        string
		key         string
		shouldExist bool
		expected    interface{}
	}{
		{
			name:        "空key返回自身",
			key:         "",
			shouldExist: true,
			expected:    testData,
		},
		{
			name:        "简单字段访问",
			key:         "servers",
			shouldExist: true,
			expected:    []interface{}{"server1", "server2"},
		},
		{
			name:        "嵌套map访问",
			key:         "database",
			shouldExist: true,
			expected: map[string]interface{}{
				"host": "localhost",
				"port": 3306,
				"connections": []interface{}{
					map[string]interface{}{"name": "primary", "user": "admin"},
					map[string]interface{}{"name": "secondary", "user": "readonly"},
				},
			},
		},
		{
			name:        "不存在的key",
			key:         "nonexistent",
			shouldExist: false,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := storage.Sub(tt.key)

			if !tt.shouldExist {
				// 检查返回的是否是 nil MapStorage
				if result == nil {
					t.Error("Sub should return nil MapStorage, not nil interface")
				}
				if ms, ok := result.(*MapStorage); !ok || ms != nil {
					t.Errorf("Expected nil *MapStorage, got %T: %v", result, ms)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result for existing key")
			}

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			if err != nil {
				t.Fatalf("ConvertTo failed: %v", err)
			}

			// 使用深度比较验证数据
			if !deepEqual(actualData, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, actualData)
			}
		})
	}
}

// TestMapStorage_Sub_NestedPath 测试嵌套路径访问
func TestMapStorage_Sub_NestedPath(t *testing.T) {
	storage := NewMapStorage(testData)

	tests := []struct {
		name        string
		path        string
		shouldExist bool
		expected    interface{}
	}{
		{
			name:        "两级嵌套访问",
			path:        "database.host",
			shouldExist: true,
			expected:    "localhost",
		},
		{
			name:        "两级嵌套数字访问",
			path:        "database.port",
			shouldExist: true,
			expected:    3306,
		},
		{
			name:        "三级嵌套访问",
			path:        "config.timeout",
			shouldExist: true,
			expected:    "30s",
		},
		{
			name:        "不存在的嵌套路径",
			path:        "database.nonexistent",
			shouldExist: false,
		},
		{
			name:        "部分存在的路径",
			path:        "nonexistent.field",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := storage.Sub(tt.path)

			if !tt.shouldExist {
				if result == nil {
					t.Error("Sub should return nil MapStorage, not nil interface")
				}
				if ms, ok := result.(*MapStorage); !ok || ms != nil {
					t.Errorf("Expected nil *MapStorage, got %T: %v", result, ms)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result for existing path")
			}

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			if err != nil {
				t.Fatalf("ConvertTo failed: %v", err)
			}

			if actualData != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, actualData)
			}
		})
	}
}

// TestMapStorage_Sub_ArrayIndex 测试数组索引访问
func TestMapStorage_Sub_ArrayIndex(t *testing.T) {
	storage := NewMapStorage(testData)

	tests := []struct {
		name        string
		path        string
		shouldExist bool
		expected    interface{}
	}{
		{
			name:        "数组第一个元素",
			path:        "servers[0]",
			shouldExist: true,
			expected:    "server1",
		},
		{
			name:        "数组第二个元素",
			path:        "servers[1]",
			shouldExist: true,
			expected:    "server2",
		},
		{
			name:        "数组越界访问",
			path:        "servers[2]",
			shouldExist: false,
		},
		{
			name:        "负数索引",
			path:        "servers[-1]",
			shouldExist: false,
		},
		{
			name:        "非数字索引",
			path:        "servers[abc]",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := storage.Sub(tt.path)

			if !tt.shouldExist {
				if result == nil {
					t.Error("Sub should return nil MapStorage, not nil interface")
				}
				if ms, ok := result.(*MapStorage); !ok || ms != nil {
					t.Errorf("Expected nil *MapStorage, got %T: %v", result, ms)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result for valid array index")
			}

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			if err != nil {
				t.Fatalf("ConvertTo failed: %v", err)
			}

			if actualData != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, actualData)
			}
		})
	}
}

// TestMapStorage_Sub_ComplexPath 测试复杂路径访问
func TestMapStorage_Sub_ComplexPath(t *testing.T) {
	storage := NewMapStorage(testData)

	tests := []struct {
		name        string
		path        string
		shouldExist bool
		expected    interface{}
	}{
		{
			name:        "数组元素的字段",
			path:        "database.connections[0].name",
			shouldExist: true,
			expected:    "primary",
		},
		{
			name:        "数组第二个元素的字段",
			path:        "database.connections[1].user",
			shouldExist: true,
			expected:    "readonly",
		},
		{
			name:        "数组越界的字段访问",
			path:        "database.connections[2].name",
			shouldExist: false,
		},
		{
			name:        "数组元素不存在的字段",
			path:        "database.connections[0].nonexistent",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := storage.Sub(tt.path)

			if !tt.shouldExist {
				if result == nil {
					t.Error("Sub should return nil MapStorage, not nil interface")
				}
				if ms, ok := result.(*MapStorage); !ok || ms != nil {
					t.Errorf("Expected nil *MapStorage, got %T: %v", result, ms)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result for valid complex path")
			}

			var actualData interface{}
			err := result.ConvertTo(&actualData)
			if err != nil {
				t.Fatalf("ConvertTo failed: %v", err)
			}

			if actualData != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, actualData)
			}
		})
	}
}

// TestMapStorage_Sub_DefaultsInheritance 测试子Storage的默认值继承
func TestMapStorage_Sub_DefaultsInheritance(t *testing.T) {
	// 测试带默认值的Storage
	storageWithDefaults := NewMapStorage(testData)
	sub1 := storageWithDefaults.Sub("database")
	
	if sub1 == nil {
		t.Fatal("Expected non-nil sub storage")
	}
	
	subMS := sub1.(*MapStorage)
	if !subMS.enableDefaults {
		t.Error("Expected sub storage to inherit enableDefaults=true")
	}

	// 测试不带默认值的Storage
	storageWithoutDefaults := NewMapStorageWithoutDefaults(testData)
	sub2 := storageWithoutDefaults.Sub("database")
	
	if sub2 == nil {
		t.Fatal("Expected non-nil sub storage")
	}
	
	subMS2 := sub2.(*MapStorage)
	if subMS2.enableDefaults {
		t.Error("Expected sub storage to inherit enableDefaults=false")
	}
}

// deepEqual 深度比较两个值是否相等
func deepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

// TestMapStorage_ConvertTo_BasicTypes 测试基本类型转换
func TestMapStorage_ConvertTo_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		target   interface{}
		expected interface{}
	}{
		{
			name:     "字符串转换",
			data:     "hello world",
			target:   new(string),
			expected: "hello world",
		},
		{
			name:     "整数转换",
			data:     42,
			target:   new(int),
			expected: 42,
		},
		{
			name:     "浮点数转换",
			data:     3.14,
			target:   new(float64),
			expected: 3.14,
		},
		{
			name:     "布尔值转换",
			data:     true,
			target:   new(bool),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMapStorage(tt.data)
			err := storage.ConvertTo(tt.target)
			if err != nil {
				t.Fatalf("ConvertTo failed: %v", err)
			}

			// 获取转换后的值
			actual := reflect.ValueOf(tt.target).Elem().Interface()
			if actual != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, actual)
			}
		})
	}
}

// TestMapStorage_ConvertTo_Struct 测试结构体转换
func TestMapStorage_ConvertTo_Struct(t *testing.T) {
	type ServerConfig struct {
		Name    string `json:"name"`
		Port    int    `json:"port"`
		Enabled bool   `json:"enabled"`
	}

	data := map[string]interface{}{
		"name":    "test-server",
		"port":    8080,
		"enabled": true,
	}

	storage := NewMapStorage(data)
	var config ServerConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if config.Name != "test-server" {
		t.Errorf("Expected name 'test-server', got %v", config.Name)
	}
	if config.Port != 8080 {
		t.Errorf("Expected port 8080, got %v", config.Port)
	}
	if !config.Enabled {
		t.Errorf("Expected enabled true, got %v", config.Enabled)
	}
}

// TestMapStorage_ConvertTo_Slice 测试切片转换
func TestMapStorage_ConvertTo_Slice(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		target   interface{}
		expected interface{}
	}{
		{
			name:     "字符串切片",
			data:     []interface{}{"item1", "item2", "item3"},
			target:   &[]string{},
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "整数切片",
			data:     []interface{}{1, 2, 3},
			target:   &[]int{},
			expected: []int{1, 2, 3},
		},
		{
			name:     "空切片",
			data:     []interface{}{},
			target:   &[]string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMapStorage(tt.data)
			err := storage.ConvertTo(tt.target)
			if err != nil {
				t.Fatalf("ConvertTo failed: %v", err)
			}

			actual := reflect.ValueOf(tt.target).Elem().Interface()
			if !deepEqual(actual, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, actual)
			}
		})
	}
}

// TestMapStorage_ConvertTo_Map 测试Map转换
func TestMapStorage_ConvertTo_Map(t *testing.T) {
	data := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": 123,
	}

	storage := NewMapStorage(data)
	
	// 转换为 map[string]interface{}
	var result1 map[string]interface{}
	err := storage.ConvertTo(&result1)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}
	
	if !deepEqual(result1, data) {
		t.Errorf("Expected %v, got %v", data, result1)
	}

	// 转换为 map[string]string
	stringData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2", 
	}
	storage2 := NewMapStorage(stringData)
	var result2 map[string]string
	err = storage2.ConvertTo(&result2)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	if !deepEqual(result2, expected) {
		t.Errorf("Expected %v, got %v", expected, result2)
	}
}

// TestMapStorage_ConvertTo_Time 测试时间类型转换
func TestMapStorage_ConvertTo_Time(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected time.Time
	}{
		{
			name:     "RFC3339 字符串",
			data:     "2023-12-25T15:30:45Z",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		},
		{
			name:     "日期字符串",
			data:     "2023-12-25",
			expected: time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "日期时间字符串",
			data:     "2023-12-25 15:30:45",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		},
		{
			name:     "Unix时间戳整数",
			data:     int64(1703517045),
			expected: time.Unix(1703517045, 0),
		},
		{
			name:     "Unix时间戳浮点数",
			data:     1703517045.5,
			expected: time.Unix(1703517045, 500000000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMapStorage(tt.data)
			var timeValue time.Time
			err := storage.ConvertTo(&timeValue)
			if err != nil {
				t.Fatalf("ConvertTo failed: %v", err)
			}
			if !timeValue.Equal(tt.expected) {
				t.Errorf("Expected time %v, got %v", tt.expected, timeValue)
			}
		})
	}
}

// TestMapStorage_ConvertTo_Duration 测试Duration类型转换
func TestMapStorage_ConvertTo_Duration(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected time.Duration
	}{
		{
			name:     "字符串Duration",
			data:     "5m30s",
			expected: 5*time.Minute + 30*time.Second,
		},
		{
			name:     "小时Duration",
			data:     "2h15m",
			expected: 2*time.Hour + 15*time.Minute,
		},
		{
			name:     "纳秒整数",
			data:     int64(1000000000),
			expected: time.Second,
		},
		{
			name:     "秒浮点数",
			data:     2.5,
			expected: 2*time.Second + 500*time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMapStorage(tt.data)
			var duration time.Duration
			err := storage.ConvertTo(&duration)
			if err != nil {
				t.Fatalf("ConvertTo failed: %v", err)
			}
			if duration != tt.expected {
				t.Errorf("Expected duration %v, got %v", tt.expected, duration)
			}
		})
	}
}

// TestMapStorage_ConvertTo_StructWithTags 测试带标签的结构体转换
func TestMapStorage_ConvertTo_StructWithTags(t *testing.T) {
	type TestConfig struct {
		Field1 string `cfg:"custom_name" json:"json_name"`
		Field2 string `json:"json_field"`
		Field3 string `yaml:"yaml_field"`
		Field4 string `toml:"toml_field"`
		Field5 string `ini:"ini_field"`
		Field6 string // 无标签，使用字段名
	}

	data := map[string]interface{}{
		"custom_name": "value1",
		"json_field":  "value2",
		"yaml_field":  "value3",
		"toml_field":  "value4",
		"ini_field":   "value5",
		"Field6":      "value6",
	}

	storage := NewMapStorage(data)
	var config TestConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if config.Field1 != "value1" {
		t.Errorf("Expected Field1 'value1', got %v", config.Field1)
	}
	if config.Field2 != "value2" {
		t.Errorf("Expected Field2 'value2', got %v", config.Field2)
	}
	if config.Field3 != "value3" {
		t.Errorf("Expected Field3 'value3', got %v", config.Field3)
	}
	if config.Field4 != "value4" {
		t.Errorf("Expected Field4 'value4', got %v", config.Field4)
	}
	if config.Field5 != "value5" {
		t.Errorf("Expected Field5 'value5', got %v", config.Field5)
	}
	if config.Field6 != "value6" {
		t.Errorf("Expected Field6 'value6', got %v", config.Field6)
	}
}

// TestMapStorage_ConvertTo_NestedStruct 测试嵌套结构体转换
func TestMapStorage_ConvertTo_NestedStruct(t *testing.T) {
	type DatabaseConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	type AppConfig struct {
		Name     string         `json:"name"`
		Database DatabaseConfig `json:"database"`
		Servers  []string       `json:"servers"`
	}

	data := map[string]interface{}{
		"name": "test-app",
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		},
		"servers": []interface{}{"server1", "server2"},
	}

	storage := NewMapStorage(data)
	var config AppConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if config.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got %v", config.Name)
	}
	if config.Database.Host != "localhost" {
		t.Errorf("Expected database host 'localhost', got %v", config.Database.Host)
	}
	if config.Database.Port != 3306 {
		t.Errorf("Expected database port 3306, got %v", config.Database.Port)
	}
	if len(config.Servers) != 2 || config.Servers[0] != "server1" || config.Servers[1] != "server2" {
		t.Errorf("Expected servers [server1, server2], got %v", config.Servers)
	}
}

// TestMapStorage_Equals_Basic 测试基本比较功能
func TestMapStorage_Equals_Basic(t *testing.T) {
	data1 := map[string]interface{}{
		"host": "localhost",
		"port": 3306,
	}

	data2 := map[string]interface{}{
		"host": "localhost", 
		"port": 3306,
	}

	data3 := map[string]interface{}{
		"host": "localhost",
		"port": 3307,
	}

	storage1 := NewMapStorage(data1)
	storage2 := NewMapStorage(data2)
	storage3 := NewMapStorage(data3)

	// 测试相同数据的比较
	if !storage1.Equals(storage2) {
		t.Error("Expected storage1 to equal storage2")
	}

	// 测试不同数据的比较
	if storage1.Equals(storage3) {
		t.Error("Expected storage1 to not equal storage3")
	}

	// 测试与自身的比较
	if !storage1.Equals(storage1) {
		t.Error("Expected storage to equal itself")
	}
}

// TestMapStorage_Equals_ComplexData 测试复杂数据比较
func TestMapStorage_Equals_ComplexData(t *testing.T) {
	complexData1 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		},
		"servers": []interface{}{"server1", "server2"},
	}

	complexData2 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		},
		"servers": []interface{}{"server1", "server2"},
	}

	complexData3 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		},
		"servers": []interface{}{"server1", "server3"}, // 不同的server
	}

	storage1 := NewMapStorage(complexData1)
	storage2 := NewMapStorage(complexData2)
	storage3 := NewMapStorage(complexData3)

	if !storage1.Equals(storage2) {
		t.Error("Expected complex storage1 to equal complex storage2")
	}

	if storage1.Equals(storage3) {
		t.Error("Expected complex storage1 to not equal complex storage3")
	}
}

// TestMapStorage_Equals_NilHandling 测试nil处理
func TestMapStorage_Equals_NilHandling(t *testing.T) {
	data := map[string]interface{}{
		"key": "value",
	}

	normalStorage := NewMapStorage(data)
	var nilStorage1 *MapStorage = nil
	var nilStorage2 *MapStorage = nil

	// 获取通过Sub方法返回的nil storage
	nilFromSub := normalStorage.Sub("nonexistent")

	// 测试nil storage与nil storage比较
	if !nilStorage1.Equals(nilStorage2) {
		t.Error("Expected nil MapStorage to equal nil MapStorage")
	}

	// 测试nil storage与从Sub返回的nil storage比较
	if !nilStorage1.Equals(nilFromSub) {
		t.Error("Expected nil MapStorage to equal nil MapStorage from Sub")
	}

	// 测试从Sub返回的nil storage之间比较
	nilFromSub2 := normalStorage.Sub("another_nonexistent")
	if !nilFromSub.Equals(nilFromSub2) {
		t.Error("Expected nil MapStorage from Sub to equal nil MapStorage from Sub")
	}

	// 测试nil storage与正常storage比较
	if nilStorage1.Equals(normalStorage) {
		t.Error("Expected nil MapStorage to not equal normal MapStorage")
	}

	// 测试正常storage与nil storage比较
	if normalStorage.Equals(nilStorage1) {
		t.Error("Expected normal MapStorage to not equal nil MapStorage")
	}

	// 测试nil storage与nil接口比较
	if nilStorage1.Equals(nil) {
		t.Error("Expected nil MapStorage to not equal nil interface")
	}
}

// TestMapStorage_Equals_EmptyData 测试空数据比较
func TestMapStorage_Equals_EmptyData(t *testing.T) {
	empty1 := NewMapStorage(nil)
	empty2 := NewMapStorage(nil)
	emptyMap1 := NewMapStorage(map[string]interface{}{})
	emptyMap2 := NewMapStorage(map[string]interface{}{})

	// nil数据应该相等
	if !empty1.Equals(empty2) {
		t.Error("Expected empty1 to equal empty2")
	}

	// 空map应该相等
	if !emptyMap1.Equals(emptyMap2) {
		t.Error("Expected emptyMap1 to equal emptyMap2")
	}

	// nil和空map在reflect.DeepEqual中不相等，这是预期行为
	if empty1.Equals(emptyMap1) {
		t.Error("Expected empty1 to not equal emptyMap1 (nil vs empty map)")
	}
}

// TestMapStorage_Equals_SubStorage 测试子Storage的比较
func TestMapStorage_Equals_SubStorage(t *testing.T) {
	data1 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		},
	}

	data2 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3306,
		},
	}

	data3 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 3307,
		},
	}

	storage1 := NewMapStorage(data1)
	storage2 := NewMapStorage(data2)
	storage3 := NewMapStorage(data3)

	sub1 := storage1.Sub("database")
	sub2 := storage2.Sub("database")
	sub3 := storage3.Sub("database")

	// 相同数据的子Storage应该相等
	if !sub1.Equals(sub2) {
		t.Error("Expected sub1 to equal sub2")
	}

	// 不同数据的子Storage应该不相等
	if sub1.Equals(sub3) {
		t.Error("Expected sub1 to not equal sub3")
	}
}

// MockStorage 用于测试的模拟Storage实现
type MockStorage struct{}

func (ms *MockStorage) Sub(key string) Storage { return nil }
func (ms *MockStorage) ConvertTo(object interface{}) error { return nil }
func (ms *MockStorage) Equals(other Storage) bool { return false }

// TestMapStorage_Equals_DifferentTypes 测试不同类型的比较
func TestMapStorage_Equals_DifferentTypes(t *testing.T) {
	storage := NewMapStorage(testData)
	mockStorage := &MockStorage{}

	// MapStorage与其他类型的Storage比较应该返回false
	if storage.Equals(mockStorage) {
		t.Error("Expected MapStorage to not equal different storage type")
	}
}

// TestMapStorage_ConvertTo_NilStorage 测试nil storage的ConvertTo行为
func TestMapStorage_ConvertTo_NilStorage(t *testing.T) {
	// 获取一个nil storage
	normalStorage := NewMapStorage(testData)
	nilStorage := normalStorage.Sub("nonexistent")

	// 测试对空指针的处理
	type TestConfig struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}

	var nilConfig *TestConfig = nil
	err := nilStorage.ConvertTo(&nilConfig)
	if err != nil {
		t.Errorf("ConvertTo should not fail for nil storage, got error: %v", err)
	}
	if nilConfig != nil {
		t.Error("Expected nil pointer to remain nil when converting from nil storage")
	}

	// 测试对非空指针的处理
	existingConfig := &TestConfig{Name: "existing", Port: 5432}
	err = nilStorage.ConvertTo(&existingConfig)
	if err != nil {
		t.Errorf("ConvertTo should not fail for nil storage, got error: %v", err)
	}
	// 对于非空指针，应该保持不变
	if existingConfig.Name != "existing" || existingConfig.Port != 5432 {
		t.Error("Expected non-nil pointer values to remain unchanged when converting from nil storage")
	}
}

// TestMapStorage_ConvertTo_PointerFields 测试指针字段的智能处理
func TestMapStorage_ConvertTo_PointerFields(t *testing.T) {
	type InnerConfig struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	type OuterConfig struct {
		Name     string       `json:"name"`
		Inner    *InnerConfig `json:"inner"`
		Optional *InnerConfig `json:"optional"`
	}

	t.Run("配置中没有指针字段", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
		}
		storage := NewMapStorage(data)

		// 目标结构体的指针字段为nil
		var config1 OuterConfig
		config1.Inner = nil
		config1.Optional = nil

		err := storage.ConvertTo(&config1)
		if err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}

		if config1.Name != "test" {
			t.Errorf("Expected name 'test', got %v", config1.Name)
		}
		if config1.Inner != nil {
			t.Error("Expected Inner to remain nil when not in config")
		}
		if config1.Optional != nil {
			t.Error("Expected Optional to remain nil when not in config")
		}

		// 目标结构体的指针字段已有值
		existingInner := &InnerConfig{Value: "existing", Count: 999}
		var config2 OuterConfig
		config2.Inner = existingInner
		config2.Optional = nil

		err = storage.ConvertTo(&config2)
		if err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}

		// 已存在的指针应该保持不变
		if config2.Inner != existingInner {
			t.Error("Expected Inner to remain unchanged when not in config")
		}
		if config2.Inner.Value != "existing" || config2.Inner.Count != 999 {
			t.Error("Expected Inner values to remain unchanged when not in config")
		}
	})

	t.Run("配置中有指针字段", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
			"inner": map[string]interface{}{
				"value": "configured",
				"count": 42,
			},
		}
		storage := NewMapStorage(data)

		// 目标结构体的inner字段为nil
		var config OuterConfig
		config.Inner = nil
		config.Optional = nil

		err := storage.ConvertTo(&config)
		if err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}

		// Inner应该被创建并赋值
		if config.Inner == nil {
			t.Error("Expected Inner to be created when in config")
		} else {
			if config.Inner.Value != "configured" {
				t.Errorf("Expected Inner.Value 'configured', got %v", config.Inner.Value)
			}
			if config.Inner.Count != 42 {
				t.Errorf("Expected Inner.Count 42, got %v", config.Inner.Count)
			}
		}
		if config.Optional != nil {
			t.Error("Expected Optional to remain nil when not in config")
		}
	})
}

// TestMapStorage_ErrorHandling 测试错误处理
func TestMapStorage_ErrorHandling(t *testing.T) {

	t.Run("类型转换错误", func(t *testing.T) {
		// 尝试将非数组数据转换为切片
		stringStorage := NewMapStorage("not a slice")
		var slice []string
		err := stringStorage.ConvertTo(&slice)
		if err == nil {
			t.Error("Expected error when converting string to slice")
		}
	})

	t.Run("时间解析错误", func(t *testing.T) {
		// 无效的时间格式
		invalidTimeStorage := NewMapStorage("invalid-time-format")
		var timeValue time.Time
		err := invalidTimeStorage.ConvertTo(&timeValue)
		if err == nil {
			t.Error("Expected error when parsing invalid time format")
		}
	})

	t.Run("Duration解析错误", func(t *testing.T) {
		// 无效的duration格式
		invalidDurationStorage := NewMapStorage("invalid-duration")
		var duration time.Duration
		err := invalidDurationStorage.ConvertTo(&duration)
		if err == nil {
			t.Error("Expected error when parsing invalid duration format")
		}
	})
}

// TestMapStorage_EdgeCases 测试边界情况
func TestMapStorage_EdgeCases(t *testing.T) {
	t.Run("空字符串路径", func(t *testing.T) {
		storage := NewMapStorage(testData)
		result := storage.Sub("")
		
		// 空路径应该返回自身
		if result == nil {
			t.Error("Expected non-nil result for empty path")
		}
		
		var data interface{}
		err := result.ConvertTo(&data)
		if err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}
		
		if !deepEqual(data, testData) {
			t.Error("Expected empty path to return original data")
		}
	})

	t.Run("路径中包含特殊字符", func(t *testing.T) {
		specialData := map[string]interface{}{
			"key.with.dots":     "value1",
			"key[with]brackets": "value2",
			"normal_key":        "value3",
		}
		
		storage := NewMapStorage(specialData)
		
		// 正常的key应该能访问
		result := storage.Sub("normal_key")
		if result == nil {
			t.Error("Expected to find normal_key")
		}
		
		var value string
		err := result.ConvertTo(&value)
		if err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}
		
		if value != "value3" {
			t.Errorf("Expected 'value3', got %v", value)
		}
	})

	t.Run("深层嵌套null值", func(t *testing.T) {
		nullData := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": nil,
			},
		}
		
		storage := NewMapStorage(nullData)
		result := storage.Sub("level1.level2")
		
		// 应该返回nil storage
		if result == nil {
			t.Error("Sub should return nil MapStorage, not nil interface")
		}
		if ms, ok := result.(*MapStorage); !ok || ms != nil {
			t.Errorf("Expected nil *MapStorage, got %T: %v", result, ms)
		}
	})
}