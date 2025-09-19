package ref_test

import (
	"testing"

	"github.com/hatlonely/gox/cfg/storage"
	"github.com/hatlonely/gox/ref"
)

// TestStorageIntegration tests the integration between ref and storage packages
// This test is in ref_test package to demonstrate the dependency direction
func TestStorageIntegration(t *testing.T) {
	namespace := "test-storage-integration"

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

	err := ref.Register(namespace, "Database", newDatabase)
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

	// Test using Storage as options - storage.Storage implements ref.Convertable
	result, err := ref.New(namespace, "Database", mapStorage)
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

// TestFlatStorageIntegration tests Storage support with FlatStorage
func TestFlatStorageIntegration(t *testing.T) {
	namespace := "test-flat-storage-integration"

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

	err := ref.Register(namespace, "Server", newServer)
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
	result, err := ref.New(namespace, "Server", flatStorage)
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

// TestStorageIntegrationWithNewT tests Storage support with NewT method
func TestStorageIntegrationWithNewT(t *testing.T) {
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
	err := ref.RegisterT[*APIClient](newAPIClient)
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
	client, err := ref.NewT[*APIClient](mapStorage)
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
