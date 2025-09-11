package storage

import (
	"testing"
	"time"
)

func TestFlatStorage_BasicUsage(t *testing.T) {
	// 创建打平的数据
	data := map[string]interface{}{
		"name":               "test-app",
		"version":            "1.0.0",
		"database.host":      "localhost",
		"database.port":      5432,
		"database.enabled":   true,
		"database.timeout":   "30s",
		"servers[0]":         "server1.com",
		"servers[1]":         "server2.com",
		"config.debug":       false,
		"config.max_workers": 10,
	}

	storage := NewFlatStorage(data)

	// 测试简单字段访问
	nameStorage := storage.Sub("name")
	var name string
	err := nameStorage.ConvertTo(&name)
	if err != nil {
		t.Fatalf("Failed to get name: %v", err)
	}
	if name != "test-app" {
		t.Errorf("Expected name 'test-app', got %v", name)
	}

	// 测试嵌套字段访问
	hostStorage := storage.Sub("database.host")
	var host string
	err = hostStorage.ConvertTo(&host)
	if err != nil {
		t.Fatalf("Failed to get database host: %v", err)
	}
	if host != "localhost" {
		t.Errorf("Expected host 'localhost', got %v", host)
	}

	// 测试数组索引访问
	server0Storage := storage.Sub("servers[0]")
	var server0 string
	err = server0Storage.ConvertTo(&server0)
	if err != nil {
		t.Fatalf("Failed to get servers[0]: %v", err)
	}
	if server0 != "server1.com" {
		t.Errorf("Expected servers[0] 'server1.com', got %v", server0)
	}

	// 测试时间类型转换
	timeoutStorage := storage.Sub("database.timeout")
	var timeout time.Duration
	err = timeoutStorage.ConvertTo(&timeout)
	if err != nil {
		t.Fatalf("Failed to get database timeout: %v", err)
	}
	expectedTimeout := 30 * time.Second
	if timeout != expectedTimeout {
		t.Errorf("Expected timeout %v, got %v", expectedTimeout, timeout)
	}
}

func TestFlatStorage_SubStorage(t *testing.T) {
	data := map[string]interface{}{
		"app.name":            "test-service",
		"app.version":         "2.0.0",
		"app.config.port":     8080,
		"app.config.debug":    true,
		"app.config.timeout":  "15s",
		"database.host":       "db.example.com",
		"database.port":       3306,
		"database.pools[0].name": "read",
		"database.pools[0].size": 10,
		"database.pools[1].name": "write",
		"database.pools[1].size": 5,
	}

	storage := NewFlatStorage(data)

	// 测试获取app子存储
	appStorage := storage.Sub("app")
	
	// 从子存储获取name
	appNameStorage := appStorage.Sub("name")
	var appName string
	err := appNameStorage.ConvertTo(&appName)
	if err != nil {
		t.Fatalf("Failed to get app name from sub storage: %v", err)
	}
	if appName != "test-service" {
		t.Errorf("Expected app name 'test-service', got %v", appName)
	}

	// 从子存储获取嵌套配置
	appConfigPortStorage := appStorage.Sub("config.port")
	var appPort int
	err = appConfigPortStorage.ConvertTo(&appPort)
	if err != nil {
		t.Fatalf("Failed to get app config port from sub storage: %v", err)
	}
	if appPort != 8080 {
		t.Errorf("Expected app port 8080, got %v", appPort)
	}

	// 测试获取数据库子存储
	dbStorage := storage.Sub("database")
	
	// 获取数据库池的第一个元素
	pool0NameStorage := dbStorage.Sub("pools[0].name")
	var pool0Name string
	err = pool0NameStorage.ConvertTo(&pool0Name)
	if err != nil {
		t.Fatalf("Failed to get pool0 name from db sub storage: %v", err)
	}
	if pool0Name != "read" {
		t.Errorf("Expected pool0 name 'read', got %v", pool0Name)
	}
}

func TestFlatStorage_ConvertToStruct(t *testing.T) {
	data := map[string]interface{}{
		"service_name":  "user-service",
		"listen_port":   9000,
		"debug_enabled": true,
		"request_timeout": "25s",
	}

	storage := NewFlatStorage(data)

	type ServiceConfig struct {
		ServiceName    string        `cfg:"service_name"`
		ListenPort     int           `cfg:"listen_port"`
		DebugEnabled   bool          `cfg:"debug_enabled"`
		RequestTimeout time.Duration `cfg:"request_timeout"`
	}

	var config ServiceConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to struct: %v", err)
	}

	if config.ServiceName != "user-service" {
		t.Errorf("Expected service name 'user-service', got %v", config.ServiceName)
	}
	if config.ListenPort != 9000 {
		t.Errorf("Expected listen port 9000, got %v", config.ListenPort)
	}
	if !config.DebugEnabled {
		t.Errorf("Expected debug enabled true, got %v", config.DebugEnabled)
	}
	if config.RequestTimeout != 25*time.Second {
		t.Errorf("Expected request timeout 25s, got %v", config.RequestTimeout)
	}
}

func TestFlatStorage_ConvertToNestedStruct(t *testing.T) {
	// 使用打平的数据表示嵌套结构
	data := map[string]interface{}{
		"name":                    "complex-app",
		"version":                 "3.0.0",
		"database.driver":         "postgres",
		"database.host":           "pg.example.com",
		"database.port":           5432,
		"database.pool.min_size":  5,
		"database.pool.max_size":  20,
		"database.pool.timeout":   "60s",
		"cache.type":              "redis",
		"cache.host":              "redis.example.com",
		"cache.port":              6379,
		"cache.ttl":               "300s",
	}

	storage := NewFlatStorage(data)

	// 定义嵌套结构体
	type PoolConfig struct {
		MinSize int           `cfg:"min_size"`
		MaxSize int           `cfg:"max_size"`
		Timeout time.Duration `cfg:"timeout"`
	}

	type DatabaseConfig struct {
		Driver string     `cfg:"driver"`
		Host   string     `cfg:"host"`
		Port   int        `cfg:"port"`
		Pool   PoolConfig `cfg:"pool"`
	}

	type CacheConfig struct {
		Type string        `cfg:"type"`
		Host string        `cfg:"host"`
		Port int           `cfg:"port"`
		TTL  time.Duration `cfg:"ttl"`
	}

	type AppConfig struct {
		Name     string         `cfg:"name"`
		Version  string         `cfg:"version"`
		Database DatabaseConfig `cfg:"database"`
		Cache    CacheConfig    `cfg:"cache"`
	}

	var config AppConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to nested struct: %v", err)
	}

	// 验证基本字段
	if config.Name != "complex-app" {
		t.Errorf("Expected name 'complex-app', got %v", config.Name)
	}
	if config.Version != "3.0.0" {
		t.Errorf("Expected version '3.0.0', got %v", config.Version)
	}

	// 验证数据库配置
	if config.Database.Driver != "postgres" {
		t.Errorf("Expected database driver 'postgres', got %v", config.Database.Driver)
	}
	if config.Database.Host != "pg.example.com" {
		t.Errorf("Expected database host 'pg.example.com', got %v", config.Database.Host)
	}
	if config.Database.Port != 5432 {
		t.Errorf("Expected database port 5432, got %v", config.Database.Port)
	}

	// 验证连接池配置
	if config.Database.Pool.MinSize != 5 {
		t.Errorf("Expected pool min size 5, got %v", config.Database.Pool.MinSize)
	}
	if config.Database.Pool.MaxSize != 20 {
		t.Errorf("Expected pool max size 20, got %v", config.Database.Pool.MaxSize)
	}
	if config.Database.Pool.Timeout != 60*time.Second {
		t.Errorf("Expected pool timeout 60s, got %v", config.Database.Pool.Timeout)
	}

	// 验证缓存配置
	if config.Cache.Type != "redis" {
		t.Errorf("Expected cache type 'redis', got %v", config.Cache.Type)
	}
	if config.Cache.TTL != 300*time.Second {
		t.Errorf("Expected cache TTL 300s, got %v", config.Cache.TTL)
	}
}

func TestFlatStorage_FromNestedData(t *testing.T) {
	// 创建嵌套数据
	nestedData := map[string]interface{}{
		"application": map[string]interface{}{
			"name":    "nested-app",
			"version": "1.0.0",
		},
		"servers": []interface{}{
			"server1.example.com",
			"server2.example.com",
		},
		"database": map[string]interface{}{
			"connections": []interface{}{
				map[string]interface{}{
					"name": "primary",
					"host": "db1.example.com",
				},
				map[string]interface{}{
					"name": "secondary",
					"host": "db2.example.com",
				},
			},
		},
	}

	// 从嵌套数据创建FlatStorage
	storage := NewFlatStorageFromNested(nestedData)

	// 验证打平的数据
	data := storage.Data()
	
	// 检查应用名称
	if name, exists := data["application.name"]; !exists || name != "nested-app" {
		t.Errorf("Expected application.name 'nested-app', got %v", name)
	}
	
	// 检查应用版本
	if version, exists := data["application.version"]; !exists || version != "1.0.0" {
		t.Errorf("Expected application.version '1.0.0', got %v", version)
	}
	
	// 检查服务器数组
	if server0, exists := data["servers[0]"]; !exists || server0 != "server1.example.com" {
		t.Errorf("Expected servers[0] 'server1.example.com', got %v", server0)
	}
	
	if server1, exists := data["servers[1]"]; !exists || server1 != "server2.example.com" {
		t.Errorf("Expected servers[1] 'server2.example.com', got %v", server1)
	}
	
	// 检查嵌套数组的对象
	if connName, exists := data["database.connections[0].name"]; !exists || connName != "primary" {
		t.Errorf("Expected database.connections[0].name 'primary', got %v", connName)
	}
	
	if connHost, exists := data["database.connections[1].host"]; !exists || connHost != "db2.example.com" {
		t.Errorf("Expected database.connections[1].host 'db2.example.com', got %v", connHost)
	}

	// 测试从打平的数据获取值
	appNameStorage := storage.Sub("application.name")
	var appName string
	err := appNameStorage.ConvertTo(&appName)
	if err != nil {
		t.Fatalf("Failed to get application name: %v", err)
	}
	if appName != "nested-app" {
		t.Errorf("Expected application name 'nested-app', got %v", appName)
	}

	// 测试数组访问
	server0Storage := storage.Sub("servers[0]")
	var server0 string
	err = server0Storage.ConvertTo(&server0)
	if err != nil {
		t.Fatalf("Failed to get server 0: %v", err)
	}
	if server0 != "server1.example.com" {
		t.Errorf("Expected server 0 'server1.example.com', got %v", server0)
	}
}

func TestFlatStorage_ArrayHandling(t *testing.T) {
	data := map[string]interface{}{
		"tags[0]":           "web",
		"tags[1]":           "api",
		"tags[2]":           "backend",
		"users[0].name":     "alice",
		"users[0].email":    "alice@example.com",
		"users[0].active":   true,
		"users[1].name":     "bob",
		"users[1].email":    "bob@example.com",
		"users[1].active":   false,
	}

	storage := NewFlatStorage(data)

	// 测试简单数组转换
	tagsStorage := storage.Sub("tags")
	var tags []string
	err := tagsStorage.ConvertTo(&tags)
	if err != nil {
		t.Fatalf("Failed to convert tags: %v", err)
	}

	expectedTags := []string{"web", "api", "backend"}
	if len(tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(tags))
	}
	for i, expected := range expectedTags {
		if i < len(tags) && tags[i] != expected {
			t.Errorf("Expected tag %d to be %s, got %s", i, expected, tags[i])
		}
	}

	// 测试对象数组转换
	type User struct {
		Name   string `cfg:"name"`
		Email  string `cfg:"email"`
		Active bool   `cfg:"active"`
	}

	usersStorage := storage.Sub("users")
	var users []User
	err = usersStorage.ConvertTo(&users)
	if err != nil {
		t.Fatalf("Failed to convert users: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
	
	if len(users) > 0 {
		if users[0].Name != "alice" {
			t.Errorf("Expected first user name 'alice', got %v", users[0].Name)
		}
		if users[0].Email != "alice@example.com" {
			t.Errorf("Expected first user email 'alice@example.com', got %v", users[0].Email)
		}
		if !users[0].Active {
			t.Errorf("Expected first user to be active, got %v", users[0].Active)
		}
	}

	if len(users) > 1 {
		if users[1].Name != "bob" {
			t.Errorf("Expected second user name 'bob', got %v", users[1].Name)
		}
		if users[1].Active {
			t.Errorf("Expected second user to be inactive, got %v", users[1].Active)
		}
	}
}

func TestFlatStorage_CustomSeparator(t *testing.T) {
	// 使用自定义分隔符
	data := map[string]interface{}{
		"app_name":         "custom-app",
		"db_host":          "localhost",
		"db_config_pool":   10,
		"servers_0":        "srv1",
		"servers_1":        "srv2",
	}

	storage := NewFlatStorageWithOptions(data, "_", "_%d")

	// 测试基本访问
	nameStorage := storage.Sub("app")
	var name string
	err := nameStorage.ConvertTo(&name)
	if err != nil {
		t.Fatalf("Failed to get app name: %v", err)
	}
	if name != "custom-app" {
		t.Errorf("Expected app name 'custom-app', got %v", name)
	}

	// 测试嵌套访问
	poolStorage := storage.Sub("db_config")
	var pool int
	err = poolStorage.ConvertTo(&pool)
	if err != nil {
		t.Fatalf("Failed to get db config pool: %v", err)
	}
	if pool != 10 {
		t.Errorf("Expected db config pool 10, got %v", pool)
	}

	// 测试数组访问
	server0Storage := storage.Sub("servers_0")
	var server0 string
	err = server0Storage.ConvertTo(&server0)
	if err != nil {
		t.Fatalf("Failed to get server 0: %v", err)
	}
	if server0 != "srv1" {
		t.Errorf("Expected server 0 'srv1', got %v", server0)
	}
}

func TestFlatStorage_SmartFieldMatching(t *testing.T) {
	// 模拟.env文件的键名模式
	data := map[string]interface{}{
		"DATABASE_PRIMARY_HOST":     "db1.example.com",
		"DATABASE_PRIMARY_PORT":     5432,
		"DATABASE_SECONDARY_HOST":   "db2.example.com", 
		"DATABASE_SECONDARY_PORT":   5433,
		"CACHE_REDIS_HOST":          "redis.example.com",
		"CACHE_REDIS_PORT":          6379,
		"APP_NAME":                  "test-service",
		"APP_VERSION":               "1.0.0",
		"SERVER_TIMEOUT":            "30s",
	}

	storage := NewFlatStorageWithOptions(data, "_", "_%d")

	// 定义嵌套结构体，字段路径应该能智能匹配到对应的环境变量
	type DatabaseConfig struct {
		Primary struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
		} `cfg:"primary"`
		Secondary struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
		} `cfg:"secondary"`
	}

	type CacheConfig struct {
		Redis struct {
			Host string `cfg:"host"`
			Port int    `cfg:"port"`
		} `cfg:"redis"`
	}

	type AppConfig struct {
		Name     string        `cfg:"name"`
		Version  string        `cfg:"version"`
		Database DatabaseConfig `cfg:"database"`
		Cache    CacheConfig    `cfg:"cache"`
		Server   struct {
			Timeout time.Duration `cfg:"timeout"`
		} `cfg:"server"`
	}

	var config AppConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to nested struct: %v", err)
	}

	// 验证智能匹配的结果
	if config.Name != "test-service" {
		t.Errorf("Expected app name 'test-service', got %v", config.Name)
	}
	if config.Version != "1.0.0" {
		t.Errorf("Expected app version '1.0.0', got %v", config.Version)
	}
	if config.Database.Primary.Host != "db1.example.com" {
		t.Errorf("Expected database primary host 'db1.example.com', got %v", config.Database.Primary.Host)
	}
	if config.Database.Primary.Port != 5432 {
		t.Errorf("Expected database primary port 5432, got %v", config.Database.Primary.Port)
	}
	if config.Database.Secondary.Host != "db2.example.com" {
		t.Errorf("Expected database secondary host 'db2.example.com', got %v", config.Database.Secondary.Host)
	}
	if config.Cache.Redis.Host != "redis.example.com" {
		t.Errorf("Expected cache redis host 'redis.example.com', got %v", config.Cache.Redis.Host)
	}
	if config.Cache.Redis.Port != 6379 {
		t.Errorf("Expected cache redis port 6379, got %v", config.Cache.Redis.Port)
	}
	if config.Server.Timeout != 30*time.Second {
		t.Errorf("Expected server timeout 30s, got %v", config.Server.Timeout)
	}
}