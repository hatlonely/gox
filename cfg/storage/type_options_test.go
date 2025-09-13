package storage_test

import (
	"testing"

	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/refx"
)

// TestTypeOptionsSupport tests TypeOptions support in both MapStorage and FlatStorage
func TestMapStorageTypeOptionsSupport(t *testing.T) {
	// Create a MapStorage with TypeOptions-like configuration
	configData := map[string]interface{}{
		"namespace": "myapp",
		"type":      "Database",
		"options": map[string]interface{}{
			"host":     "localhost",
			"port":     3306,
			"username": "root",
			"password": "secret",
		},
	}
	mapStorage := storage.NewMapStorage(configData)

	// Test converting to TypeOptions
	var typeOpts refx.TypeOptions
	err := mapStorage.ConvertTo(&typeOpts)
	if err != nil {
		t.Fatalf("MapStorage.ConvertTo(TypeOptions) error = %v", err)
	}

	// Verify basic fields
	if typeOpts.Namespace != "myapp" {
		t.Errorf("Expected namespace 'myapp', got '%s'", typeOpts.Namespace)
	}
	if typeOpts.Type != "Database" {
		t.Errorf("Expected type 'Database', got '%s'", typeOpts.Type)
	}

	// Verify Options field is a Storage
	if typeOpts.Options == nil {
		t.Fatal("Expected Options to be non-nil")
	}

	optionsStorage, ok := typeOpts.Options.(storage.Storage)
	if !ok {
		t.Fatalf("Expected Options to be a Storage, got %T", typeOpts.Options)
	}

	// Test that we can use the Options storage to convert to specific config
	type DatabaseConfig struct {
		Host     string `cfg:"host"`
		Port     int    `cfg:"port"`
		Username string `cfg:"username"`
		Password string `cfg:"password"`
	}

	var dbConfig DatabaseConfig
	err = optionsStorage.ConvertTo(&dbConfig)
	if err != nil {
		t.Fatalf("Options storage ConvertTo error = %v", err)
	}

	// Verify the converted config
	if dbConfig.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", dbConfig.Host)
	}
	if dbConfig.Port != 3306 {
		t.Errorf("Expected port 3306, got %d", dbConfig.Port)
	}
	if dbConfig.Username != "root" {
		t.Errorf("Expected username 'root', got '%s'", dbConfig.Username)
	}
	if dbConfig.Password != "secret" {
		t.Errorf("Expected password 'secret', got '%s'", dbConfig.Password)
	}
}

// TestFlatStorageTypeOptionsSupport tests TypeOptions support in FlatStorage
func TestFlatStorageTypeOptionsSupport(t *testing.T) {
	// Create a FlatStorage with TypeOptions-like configuration
	flatData := map[string]interface{}{
		"namespace":             "myapp",
		"type":                  "Server",
		"options.database.host": "db.example.com",
		"options.database.port": 5432,
		"options.redis.host":    "redis.example.com",
		"options.redis.port":    6379,
	}
	flatStorage := storage.NewFlatStorage(flatData)

	// Test converting to TypeOptions
	var typeOpts refx.TypeOptions
	err := flatStorage.ConvertTo(&typeOpts)
	if err != nil {
		t.Fatalf("FlatStorage.ConvertTo(TypeOptions) error = %v", err)
	}

	// Verify basic fields
	if typeOpts.Namespace != "myapp" {
		t.Errorf("Expected namespace 'myapp', got '%s'", typeOpts.Namespace)
	}
	if typeOpts.Type != "Server" {
		t.Errorf("Expected type 'Server', got '%s'", typeOpts.Type)
	}

	// Verify Options field is a Storage
	if typeOpts.Options == nil {
		t.Fatal("Expected Options to be non-nil")
	}

	optionsStorage, ok := typeOpts.Options.(storage.Storage)
	if !ok {
		t.Fatalf("Expected Options to be a Storage, got %T", typeOpts.Options)
	}

	// Test that we can use the Options storage to convert to specific config
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

	var serverConfig ServerConfig
	err = optionsStorage.ConvertTo(&serverConfig)
	if err != nil {
		t.Fatalf("Options storage ConvertTo error = %v", err)
	}

	// Verify the converted config
	if serverConfig.Database.Host != "db.example.com" {
		t.Errorf("Expected database host 'db.example.com', got '%s'", serverConfig.Database.Host)
	}
	if serverConfig.Database.Port != 5432 {
		t.Errorf("Expected database port 5432, got %d", serverConfig.Database.Port)
	}
	if serverConfig.Redis.Host != "redis.example.com" {
		t.Errorf("Expected redis host 'redis.example.com', got '%s'", serverConfig.Redis.Host)
	}
	if serverConfig.Redis.Port != 6379 {
		t.Errorf("Expected redis port 6379, got %d", serverConfig.Redis.Port)
	}
}

// TestTypeOptionsWithRefxIntegration tests the complete flow with refx
func TestTypeOptionsWithRefxIntegration(t *testing.T) {
	// Define a simple struct for testing
	type SimpleConfig struct {
		Name string `cfg:"name"`
		Port int    `cfg:"port"`
	}

	type SimpleService struct {
		Config *SimpleConfig
	}

	// Constructor function
	newSimpleService := func(config *SimpleConfig) *SimpleService {
		return &SimpleService{Config: config}
	}

	// Register the constructor
	err := refx.Register("testapp", "SimpleService", newSimpleService)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Create MapStorage with TypeOptions configuration
	configData := map[string]interface{}{
		"namespace": "testapp",
		"type":      "SimpleService",
		"options": map[string]interface{}{
			"name": "test-service",
			"port": 8080,
		},
	}
	mapStorage := storage.NewMapStorage(configData)

	// Convert to TypeOptions first
	var typeOpts refx.TypeOptions
	err = mapStorage.ConvertTo(&typeOpts)
	if err != nil {
		t.Fatalf("ConvertTo(TypeOptions) error = %v", err)
	}

	// Use refx.New with the TypeOptions.Options as the options parameter
	result, err := refx.New(typeOpts.Namespace, typeOpts.Type, typeOpts.Options)
	if err != nil {
		t.Fatalf("refx.New() error = %v", err)
	}

	service, ok := result.(*SimpleService)
	if !ok {
		t.Fatalf("Expected *SimpleService, got %T", result)
	}

	// Verify the service was created correctly
	if service.Config.Name != "test-service" {
		t.Errorf("Expected name 'test-service', got '%s'", service.Config.Name)
	}
	if service.Config.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", service.Config.Port)
	}
}
