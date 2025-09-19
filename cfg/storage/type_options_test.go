package storage_test

import (
	"testing"

	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/ref"
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
	var typeOpts ref.TypeOptions
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

// TestTypeOptionsWithrefIntegration tests the complete flow with ref
func TestTypeOptionsWithrefIntegration(t *testing.T) {
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
	err := ref.Register("testapp", "SimpleService", newSimpleService)
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
	var typeOpts ref.TypeOptions
	err = mapStorage.ConvertTo(&typeOpts)
	if err != nil {
		t.Fatalf("ConvertTo(TypeOptions) error = %v", err)
	}

	// Use ref.New with the TypeOptions.Options as the options parameter
	result, err := ref.New(typeOpts.Namespace, typeOpts.Type, typeOpts.Options)
	if err != nil {
		t.Fatalf("ref.New() error = %v", err)
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

// TestFlatStorageTypeOptionsSupport tests TypeOptions support in FlatStorage
func TestFlatStorageTypeOptionsSupport(t *testing.T) {
	// Create a FlatStorage with TypeOptions-like configuration
	configData := map[string]interface{}{
		"namespace":        "myapp",
		"type":             "Database",
		"options.host":     "localhost",
		"options.port":     3306,
		"options.username": "root",
		"options.password": "secret",
	}
	flatStorage := storage.NewFlatStorage(configData)

	// Test converting to TypeOptions
	var typeOpts ref.TypeOptions
	err := flatStorage.ConvertTo(&typeOpts)
	if err != nil {
		t.Fatalf("FlatStorage.ConvertTo(TypeOptions) error = %v", err)
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

// TestFlatStorageTypeOptionsWithrefIntegration tests the complete flow with ref using FlatStorage
func TestFlatStorageTypeOptionsWithrefIntegration(t *testing.T) {
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
	err := ref.Register("testapp", "FlatSimpleService", newSimpleService)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Create FlatStorage with TypeOptions configuration
	configData := map[string]interface{}{
		"namespace":    "testapp",
		"type":         "FlatSimpleService",
		"options.name": "test-service",
		"options.port": 8080,
	}
	flatStorage := storage.NewFlatStorage(configData)

	// Convert to TypeOptions first
	var typeOpts ref.TypeOptions
	err = flatStorage.ConvertTo(&typeOpts)
	if err != nil {
		t.Fatalf("ConvertTo(TypeOptions) error = %v", err)
	}

	// Use ref.New with the TypeOptions.Options as the options parameter
	result, err := ref.New(typeOpts.Namespace, typeOpts.Type, typeOpts.Options)
	if err != nil {
		t.Fatalf("ref.New() error = %v", err)
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

// TestFlatStorageTypeOptionsWithSeparator tests FlatStorage TypeOptions with custom separator
func TestFlatStorageTypeOptionsWithSeparator(t *testing.T) {
	// Create a FlatStorage with custom separator and TypeOptions-like configuration
	configData := map[string]interface{}{
		"namespace":        "myapp",
		"type":             "Database",
		"options-host":     "localhost",
		"options-port":     3306,
		"options-username": "root",
		"options-password": "secret",
	}
	flatStorage := storage.NewFlatStorage(configData).WithSeparator("-")

	// Test converting to TypeOptions
	var typeOpts ref.TypeOptions
	err := flatStorage.ConvertTo(&typeOpts)
	if err != nil {
		t.Fatalf("FlatStorage.ConvertTo(TypeOptions) error = %v", err)
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
