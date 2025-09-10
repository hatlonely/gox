package storage

import (
	"testing"
	"time"
)

func TestMapStorage_Sub(t *testing.T) {
	data := map[string]interface{}{
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
		"servers": []interface{}{
			"server1",
			"server2",
		},
	}

	storage := NewMapStorage(data)

	// 测试简单字段访问
	dbStorage := storage.Sub("database")
	var dbConfig map[string]interface{}
	err := dbStorage.ConvertTo(&dbConfig)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if dbConfig["host"] != "localhost" {
		t.Errorf("Expected host to be localhost, got %v", dbConfig["host"])
	}

	// 测试嵌套字段访问
	hostStorage := storage.Sub("database.host")
	var host string
	err = hostStorage.ConvertTo(&host)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if host != "localhost" {
		t.Errorf("Expected host to be localhost, got %v", host)
	}

	// 测试数组索引访问
	firstConnStorage := storage.Sub("database.connections[0]")
	var firstConn map[string]interface{}
	err = firstConnStorage.ConvertTo(&firstConn)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if firstConn["name"] != "primary" {
		t.Errorf("Expected first connection name to be primary, got %v", firstConn["name"])
	}

	// 测试嵌套数组字段访问
	firstConnNameStorage := storage.Sub("database.connections[0].name")
	var connName string
	err = firstConnNameStorage.ConvertTo(&connName)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if connName != "primary" {
		t.Errorf("Expected connection name to be primary, got %v", connName)
	}
}

func TestMapStorage_ConvertTo_Struct(t *testing.T) {
	data := map[string]interface{}{
		"name": "test-server",
		"port": 8080,
		"enabled": true,
	}

	storage := NewMapStorage(data)

	type ServerConfig struct {
		Name    string `json:"name"`
		Port    int    `json:"port"`
		Enabled bool   `json:"enabled"`
	}

	var config ServerConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if config.Name != "test-server" {
		t.Errorf("Expected name to be test-server, got %v", config.Name)
	}
	if config.Port != 8080 {
		t.Errorf("Expected port to be 8080, got %v", config.Port)
	}
	if !config.Enabled {
		t.Errorf("Expected enabled to be true, got %v", config.Enabled)
	}
}

func TestMapStorage_ConvertTo_Slice(t *testing.T) {
	data := []interface{}{"item1", "item2", "item3"}

	storage := NewMapStorage(data)

	var items []string
	err := storage.ConvertTo(&items)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	expected := []string{"item1", "item2", "item3"}
	if len(items) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(items))
	}

	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected item %d to be %s, got %s", i, expected[i], item)
		}
	}
}

func TestMapStorage_ConvertTo_Duration(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected time.Duration
	}{
		{
			name:     "string duration",
			data:     "5m30s",
			expected: 5*time.Minute + 30*time.Second,
		},
		{
			name:     "string hour duration",
			data:     "2h15m",
			expected: 2*time.Hour + 15*time.Minute,
		},
		{
			name:     "nanoseconds as int64",
			data:     int64(1000000000), // 1 second
			expected: time.Second,
		},
		{
			name:     "seconds as float64",
			data:     2.5, // 2.5 seconds
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

func TestMapStorage_ConvertTo_Time(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected time.Time
	}{
		{
			name:     "RFC3339 string",
			data:     "2023-12-25T15:30:45Z",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		},
		{
			name:     "date only string",
			data:     "2023-12-25",
			expected: time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "datetime string",
			data:     "2023-12-25 15:30:45",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		},
		{
			name:     "unix timestamp int64",
			data:     int64(1703517045), // 2023-12-25 15:30:45 UTC
			expected: time.Unix(1703517045, 0),
		},
		{
			name:     "unix timestamp float64",
			data:     1703517045.5, // 2023-12-25 15:30:45.5 UTC
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

func TestMapStorage_ConvertTo_TimeInStruct(t *testing.T) {
	data := map[string]interface{}{
		"timeout":    "30s",
		"created_at": "2023-12-25T15:30:45Z",
		"expires_at": int64(1703517045),
	}

	storage := NewMapStorage(data)

	type Config struct {
		Timeout   time.Duration `json:"timeout"`
		CreatedAt time.Time     `json:"created_at"`
		ExpiresAt time.Time     `json:"expires_at"`
	}

	var config Config
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	expectedTimeout := 30 * time.Second
	if config.Timeout != expectedTimeout {
		t.Errorf("Expected timeout %v, got %v", expectedTimeout, config.Timeout)
	}

	expectedCreatedAt := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
	if !config.CreatedAt.Equal(expectedCreatedAt) {
		t.Errorf("Expected created_at %v, got %v", expectedCreatedAt, config.CreatedAt)
	}

	expectedExpiresAt := time.Unix(1703517045, 0)
	if !config.ExpiresAt.Equal(expectedExpiresAt) {
		t.Errorf("Expected expires_at %v, got %v", expectedExpiresAt, config.ExpiresAt)
	}
}