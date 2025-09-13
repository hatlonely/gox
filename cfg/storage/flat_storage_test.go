package storage

import (
	"strings"
	"testing"
	"time"
)

func TestFlatStorage_Equals(t *testing.T) {
	// 测试基本的 Equals 功能
	data1 := map[string]interface{}{
		"database.host": "localhost",
		"database.port": 3306,
		"servers[0]":    "server1",
		"servers[1]":    "server2",
	}

	data2 := map[string]interface{}{
		"database.host": "localhost",
		"database.port": 3306,
		"servers[0]":    "server1",
		"servers[1]":    "server2",
	}

	data3 := map[string]interface{}{
		"database.host": "localhost",
		"database.port": 3307, // 不同的端口
		"servers[0]":    "server1",
		"servers[1]":    "server2",
	}

	storage1 := NewFlatStorage(data1)
	storage2 := NewFlatStorage(data2)
	storage3 := NewFlatStorage(data3)

	// 测试相同数据的比较
	if !storage1.Equals(storage2) {
		t.Error("Expected storage1 to equal storage2")
	}

	// 测试不同数据的比较
	if storage1.Equals(storage3) {
		t.Error("Expected storage1 to not equal storage3")
	}

	// 测试与 nil 的比较
	if storage1.Equals(nil) {
		t.Error("Expected storage1 to not equal nil")
	}

	// 测试 Sub 后的比较
	sub1 := storage1.Sub("database")
	sub2 := storage2.Sub("database")
	sub3 := storage3.Sub("database")

	if !sub1.Equals(sub2) {
		t.Error("Expected sub1 to equal sub2")
	}

	if sub1.Equals(sub3) {
		t.Error("Expected sub1 to not equal sub3")
	}

	// 测试空数据比较
	empty1 := NewFlatStorage(map[string]interface{}{})
	empty2 := NewFlatStorage(map[string]interface{}{})

	if !empty1.Equals(empty2) {
		t.Error("Expected empty1 to equal empty2")
	}

	// 测试不同长度的数据
	shortData := map[string]interface{}{
		"database.host": "localhost",
	}
	shortStorage := NewFlatStorage(shortData)

	if storage1.Equals(shortStorage) {
		t.Error("Expected storage1 to not equal shortStorage")
	}

	// 测试相同长度但键不同的数据
	differentKeys := map[string]interface{}{
		"database.host": "localhost",
		"database.port": 3306,
		"servers[0]":    "server1",
		"cache.timeout": 30, // 不同的键
	}
	differentKeysStorage := NewFlatStorage(differentKeys)

	if storage1.Equals(differentKeysStorage) {
		t.Error("Expected storage1 to not equal differentKeysStorage")
	}

	// 测试相同键但值不同的数据
	differentValues := map[string]interface{}{
		"database.host": "localhost",
		"database.port": 3306,
		"servers[0]":    "server1",
		"servers[1]":    "server3", // 不同的值
	}
	differentValuesStorage := NewFlatStorage(differentValues)

	if storage1.Equals(differentValuesStorage) {
		t.Error("Expected storage1 to not equal differentValuesStorage")
	}
}

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

func TestFlatStorage_ComplexNestedStructure(t *testing.T) {
	// 复杂嵌套结构：同时包含 struct、map、slice
	data := map[string]interface{}{
		// 应用基础配置
		"APP_NAME":    "complex-service",
		"APP_VERSION": "2.0.0",
		"APP_DEBUG":   true,
		
		// 服务器配置 (struct)
		"SERVER_HOST":    "0.0.0.0",
		"SERVER_PORT":    8080,
		"SERVER_TIMEOUT": "60s",
		
		// 数据库连接池配置 (slice of struct)
		"DATABASE_POOLS_0_NAME":        "primary",
		"DATABASE_POOLS_0_HOST":        "db1.example.com",
		"DATABASE_POOLS_0_PORT":        5432,
		"DATABASE_POOLS_0_MAX_CONNS":   20,
		"DATABASE_POOLS_0_TIMEOUT":     "30s",
		"DATABASE_POOLS_1_NAME":        "replica",
		"DATABASE_POOLS_1_HOST":        "db2.example.com",
		"DATABASE_POOLS_1_PORT":        5433,
		"DATABASE_POOLS_1_MAX_CONNS":   10,
		"DATABASE_POOLS_1_TIMEOUT":     "15s",
		
		// 缓存配置 (map)
		"CACHE_REDIS_URL":      "redis://localhost:6379",
		"CACHE_REDIS_PASSWORD": "secret",
		"CACHE_MEMCACHED_URLS": "mc1.example.com:11211,mc2.example.com:11211",
		
		// 功能开关 (map)
		"FEATURES_USER_REGISTRATION": true,
		"FEATURES_PAYMENT_GATEWAY":   false,
		"FEATURES_EMAIL_VERIFICATION": true,
		
		// 外部服务配置 (slice of map)
		"SERVICES_0_NAME": "auth-service",
		"SERVICES_0_URL":  "https://auth.example.com",
		"SERVICES_0_TIMEOUT": "10s",
		"SERVICES_0_RETRIES": 3,
		"SERVICES_1_NAME": "payment-service", 
		"SERVICES_1_URL":  "https://payment.example.com",
		"SERVICES_1_TIMEOUT": "20s",
		"SERVICES_1_RETRIES": 5,
		
		// 监控配置 (nested struct with slice)
		"MONITORING_METRICS_ENABLED":     true,
		"MONITORING_METRICS_INTERVAL":    "30s",
		"MONITORING_ALERTS_0_TYPE":       "email",
		"MONITORING_ALERTS_0_TARGET":     "admin@example.com",
		"MONITORING_ALERTS_0_THRESHOLD":  90.0,
		"MONITORING_ALERTS_1_TYPE":       "slack",
		"MONITORING_ALERTS_1_TARGET":     "#alerts",
		"MONITORING_ALERTS_1_THRESHOLD":  95.0,
	}

	storage := NewFlatStorageWithOptions(data, "_", "_%d")

	// 定义复杂嵌套结构
	type DatabasePool struct {
		Name     string        `cfg:"name"`
		Host     string        `cfg:"host"`
		Port     int           `cfg:"port"`
		MaxConns int           `cfg:"max_conns"`
		Timeout  time.Duration `cfg:"timeout"`
	}

	type ServiceConfig map[string]interface{}

	type Alert struct {
		Type      string  `cfg:"type"`
		Target    string  `cfg:"target"`
		Threshold float64 `cfg:"threshold"`
	}

	type MonitoringConfig struct {
		Metrics struct {
			Enabled  bool          `cfg:"enabled"`
			Interval time.Duration `cfg:"interval"`
		} `cfg:"metrics"`
		Alerts []Alert `cfg:"alerts"`
	}

	type ComplexConfig struct {
		// 基本字段
		Name    string `cfg:"name"`
		Version string `cfg:"version"`
		Debug   bool   `cfg:"debug"`
		
		// 嵌套结构体
		Server struct {
			Host    string        `cfg:"host"`
			Port    int           `cfg:"port"`
			Timeout time.Duration `cfg:"timeout"`
		} `cfg:"server"`
		
		// 结构体切片
		Database struct {
			Pools []DatabasePool `cfg:"pools"`
		} `cfg:"database"`
		
		// Map类型
		Cache    map[string]string `cfg:"cache"`
		Features map[string]bool   `cfg:"features"`
		
		// Map切片
		Services []ServiceConfig `cfg:"services"`
		
		// 复杂嵌套：包含结构体和切片的结构体
		Monitoring MonitoringConfig `cfg:"monitoring"`
	}

	var config ComplexConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert complex nested structure: %v", err)
	}

	// 验证基本字段
	if config.Name != "complex-service" {
		t.Errorf("Expected name 'complex-service', got %v", config.Name)
	}
	if config.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got %v", config.Version)
	}
	if !config.Debug {
		t.Errorf("Expected debug true, got %v", config.Debug)
	}

	// 验证嵌套结构体
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected server host '0.0.0.0', got %v", config.Server.Host)
	}
	if config.Server.Port != 8080 {
		t.Errorf("Expected server port 8080, got %v", config.Server.Port)
	}
	if config.Server.Timeout != 60*time.Second {
		t.Errorf("Expected server timeout 60s, got %v", config.Server.Timeout)
	}

	// 验证结构体切片
	if len(config.Database.Pools) != 2 {
		t.Errorf("Expected 2 database pools, got %d", len(config.Database.Pools))
	}
	if len(config.Database.Pools) > 0 {
		pool := config.Database.Pools[0]
		if pool.Name != "primary" {
			t.Errorf("Expected pool 0 name 'primary', got %v", pool.Name)
		}
		if pool.Host != "db1.example.com" {
			t.Errorf("Expected pool 0 host 'db1.example.com', got %v", pool.Host)
		}
		if pool.Port != 5432 {
			t.Errorf("Expected pool 0 port 5432, got %v", pool.Port)
		}
		if pool.MaxConns != 20 {
			t.Errorf("Expected pool 0 max conns 20, got %v", pool.MaxConns)
		}
		if pool.Timeout != 30*time.Second {
			t.Errorf("Expected pool 0 timeout 30s, got %v", pool.Timeout)
		}
	}

	// 验证Map类型
	if config.Cache == nil {
		t.Error("Expected cache map to be initialized")
	} else {
		// 检查不同的键名格式
		redisURL := ""
		redisPassword := ""
		for k, v := range config.Cache {
			switch strings.ToLower(k) {
			case "redis_url", "url":
				redisURL = v
			case "redis_password", "password":
				redisPassword = v
			}
		}
		
		if redisURL != "redis://localhost:6379" {
			t.Errorf("Expected cache redis url 'redis://localhost:6379', got %v", redisURL)
		}
		if redisPassword != "secret" {
			t.Errorf("Expected cache redis password 'secret', got %v", redisPassword)
		}
	}

	if config.Features == nil {
		t.Error("Expected features map to be initialized")
	} else {
		// 检查不同的键名格式 
		userReg := false
		paymentGW := true // 默认为true，这样如果找不到就会显示错误
		for k, v := range config.Features {
			switch strings.ToLower(strings.ReplaceAll(k, "_", "")) {
			case "userregistration", "registration":
				userReg = v
			case "paymentgateway", "gateway":
				paymentGW = v
			}
		}
		
		if !userReg {
			t.Errorf("Expected features user_registration true, got %v", userReg)
		}
		if paymentGW {
			t.Errorf("Expected features payment_gateway false, got %v", paymentGW)
		}
	}

	// 验证Map切片
	if len(config.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(config.Services))
	}
	if len(config.Services) > 0 {
		service := config.Services[0]
		
		// 查找name和url，考虑不同的键名格式
		var serviceName, serviceURL interface{}
		for k, v := range service {
			switch strings.ToUpper(k) {
			case "NAME":
				serviceName = v
			case "URL":
				serviceURL = v
			}
		}
		
		if serviceName != "auth-service" {
			t.Errorf("Expected service 0 name 'auth-service', got %v", serviceName)
		}
		if serviceURL != "https://auth.example.com" {
			t.Errorf("Expected service 0 url 'https://auth.example.com', got %v", serviceURL)
		}
	}

	// 验证复杂嵌套结构
	if !config.Monitoring.Metrics.Enabled {
		t.Errorf("Expected monitoring metrics enabled true, got %v", config.Monitoring.Metrics.Enabled)
	}
	if config.Monitoring.Metrics.Interval != 30*time.Second {
		t.Errorf("Expected monitoring metrics interval 30s, got %v", config.Monitoring.Metrics.Interval)
	}
	
	if len(config.Monitoring.Alerts) != 2 {
		t.Errorf("Expected 2 monitoring alerts, got %d", len(config.Monitoring.Alerts))
	}
	if len(config.Monitoring.Alerts) > 0 {
		alert := config.Monitoring.Alerts[0]
		if alert.Type != "email" {
			t.Errorf("Expected alert 0 type 'email', got %v", alert.Type)
		}
		if alert.Target != "admin@example.com" {
			t.Errorf("Expected alert 0 target 'admin@example.com', got %v", alert.Target)
		}
		if alert.Threshold != 90.0 {
			t.Errorf("Expected alert 0 threshold 90.0, got %v", alert.Threshold)
		}
	}
}

func TestFlatStorage_TimeConversion(t *testing.T) {
	// 测试 time.Time 类型转换
	data := map[string]interface{}{
		"start_time":   "2023-01-01T00:00:00Z",
		"end_time":     "2023-12-31T23:59:59Z",
		"timeout":      "30s",
		"timestamp":    int64(1672531200), // 2023-01-01 00:00:00 UTC
		"float_time":   1672531200.5,      // 2023-01-01 00:00:00.5 UTC
	}

	storage := NewFlatStorage(data)

	type TimeConfig struct {
		StartTime   time.Time     `cfg:"start_time"`
		EndTime     time.Time     `cfg:"end_time"`
		Timeout     time.Duration `cfg:"timeout"`
		Timestamp   time.Time     `cfg:"timestamp"`
		FloatTime   time.Time     `cfg:"float_time"`
	}

	var config TimeConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert to time config: %v", err)
	}

	// 验证字符串到 time.Time 的转换
	expectedStartTime, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	if !config.StartTime.Equal(expectedStartTime) {
		t.Errorf("Expected start time %v, got %v", expectedStartTime, config.StartTime)
	}

	expectedEndTime, _ := time.Parse(time.RFC3339, "2023-12-31T23:59:59Z")
	if !config.EndTime.Equal(expectedEndTime) {
		t.Errorf("Expected end time %v, got %v", expectedEndTime, config.EndTime)
	}

	// 验证字符串到 time.Duration 的转换
	expectedTimeout := 30 * time.Second
	if config.Timeout != expectedTimeout {
		t.Errorf("Expected timeout %v, got %v", expectedTimeout, config.Timeout)
	}

	// 验证 int64 时间戳到 time.Time 的转换
	expectedTimestamp := time.Unix(1672531200, 0)
	if !config.Timestamp.Equal(expectedTimestamp) {
		t.Errorf("Expected timestamp %v, got %v", expectedTimestamp, config.Timestamp)
	}

	// 验证 float64 时间戳到 time.Time 的转换
	expectedFloatTime := time.Unix(1672531200, 500000000)
	if !config.FloatTime.Equal(expectedFloatTime) {
		t.Errorf("Expected float time %v, got %v", expectedFloatTime, config.FloatTime)
	}
}

func TestFlatStorage_TimeInNestedStructure(t *testing.T) {
	// 测试嵌套结构中的 time.Time
	data := map[string]interface{}{
		"server_config_start_time":  "2023-06-15T10:30:00Z",
		"server_config_timeout":     "45s",
		"database_created_at":       "2023-01-15T08:00:00Z",
		"database_backup_interval":  "24h",
	}

	storage := NewFlatStorageWithOptions(data, "_", "_%d")

	type ServerConfig struct {
		StartTime time.Time     `cfg:"start_time"`
		Timeout   time.Duration `cfg:"timeout"`
	}

	type DatabaseConfig struct {
		CreatedAt      time.Time     `cfg:"created_at"`
		BackupInterval time.Duration `cfg:"backup_interval"`
	}

	type AppConfig struct {
		Server   ServerConfig    `cfg:"server_config"`
		Database DatabaseConfig `cfg:"database"`
	}

	var config AppConfig
	err := storage.ConvertTo(&config)
	if err != nil {
		t.Fatalf("Failed to convert nested time config: %v", err)
	}

	// 验证嵌套结构中的时间字段
	expectedServerStartTime, _ := time.Parse(time.RFC3339, "2023-06-15T10:30:00Z")
	if !config.Server.StartTime.Equal(expectedServerStartTime) {
		t.Errorf("Expected server start time %v, got %v", expectedServerStartTime, config.Server.StartTime)
	}

	if config.Server.Timeout != 45*time.Second {
		t.Errorf("Expected server timeout 45s, got %v", config.Server.Timeout)
	}

	expectedDBCreatedAt, _ := time.Parse(time.RFC3339, "2023-01-15T08:00:00Z")
	if !config.Database.CreatedAt.Equal(expectedDBCreatedAt) {
		t.Errorf("Expected database created at %v, got %v", expectedDBCreatedAt, config.Database.CreatedAt)
	}

	if config.Database.BackupInterval != 24*time.Hour {
		t.Errorf("Expected database backup interval 24h, got %v", config.Database.BackupInterval)
	}
}

func TestFlatStorage_NilHandling(t *testing.T) {
	// 测试 nil 处理特性：
	// 1. Sub 方法在没有相关 key 时返回 nil
	// 2. 对于 nil Storage，ConvertTo 不修改空指针

	data := map[string]interface{}{
		"database.host": "localhost",
		"database.port": 3306,
	}

	storage := NewFlatStorage(data)

	// 测试 Sub 方法返回 nil FlatStorage 当 key 不存在时
	nonExistentSub := storage.Sub("nonexistent")
	if nonExistentSub == nil {
		t.Error("Sub should return a nil FlatStorage, not nil interface")
	}
	if fs, ok := nonExistentSub.(*FlatStorage); !ok || fs != nil {
		t.Errorf("Expected Sub to return nil *FlatStorage for non-existent key, got %T: %v", nonExistentSub, fs)
	}

	// 测试嵌套路径不存在的情况
	nonExistentNestedSub := storage.Sub("database.nonexistent")
	if nonExistentNestedSub == nil {
		t.Error("Sub should return a nil FlatStorage, not nil interface")
	}
	if fs, ok := nonExistentNestedSub.(*FlatStorage); !ok || fs != nil {
		t.Errorf("Expected Sub to return nil *FlatStorage for non-existent nested key, got %T: %v", nonExistentNestedSub, fs)
	}

	// 测试数组索引不存在的情况
	nonExistentArraySub := storage.Sub("servers[0]")
	if nonExistentArraySub == nil {
		t.Error("Sub should return a nil FlatStorage, not nil interface")
	}
	if fs, ok := nonExistentArraySub.(*FlatStorage); !ok || fs != nil {
		t.Errorf("Expected Sub to return nil *FlatStorage for non-existent array index, got %T: %v", nonExistentArraySub, fs)
	}

	// 测试对 nil Storage 调用 ConvertTo 的行为
	type DatabaseConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	// 测试 1: 空指针应该保持为 nil
	var nilConfig *DatabaseConfig = nil
	err := nonExistentSub.ConvertTo(&nilConfig)
	if err != nil {
		t.Errorf("ConvertTo should not fail for nil storage, got error: %v", err)
	}
	if nilConfig != nil {
		t.Error("Expected nil pointer to remain nil when converting from nil storage")
	}

	// 测试 2: 非空指针结构的情况
	existingConfig := &DatabaseConfig{Host: "existing", Port: 5432}
	err = nonExistentSub.ConvertTo(&existingConfig)
	if err != nil {
		t.Errorf("ConvertTo should not fail for nil storage, got error: %v", err)
	}
	// 对于非空指针，行为应该是保持不变
	if existingConfig.Host != "existing" || existingConfig.Port != 5432 {
		t.Error("Expected non-nil pointer values to remain unchanged when converting from nil storage")
	}

	// 测试 3: 验证正常情况仍然工作
	validSub := storage.Sub("database")
	if validSub == nil {
		t.Error("Expected Sub to return non-nil for existing key")
	}

	var validConfig DatabaseConfig
	err = validSub.ConvertTo(&validConfig)
	if err != nil {
		t.Fatalf("ConvertTo failed for valid storage: %v", err)
	}

	if validConfig.Host != "localhost" || validConfig.Port != 3306 {
		t.Errorf("Expected valid config conversion, got host=%v port=%v", validConfig.Host, validConfig.Port)
	}
}