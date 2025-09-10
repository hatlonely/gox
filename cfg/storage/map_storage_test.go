package storage

import (
	"testing"
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