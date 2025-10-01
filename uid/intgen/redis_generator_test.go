package intgen

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestRedisGenerator_Generate(t *testing.T) {
	// 启动 miniredis 模拟 Redis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// 创建 RedisGenerator
	generator := NewRedisGeneratorWithOptions(&RedisOptions{
		Addr:    mr.Addr(),
		KeyName: "test:uid",
		Timeout: time.Second,
	})

	// 测试生成ID
	id1 := generator.Generate()
	id2 := generator.Generate()

	// 验证ID不为0
	if id1 == 0 {
		t.Error("Generated ID should not be 0")
	}
	if id2 == 0 {
		t.Error("Generated ID should not be 0")
	}

	// 验证ID唯一性
	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	// 验证ID的时间戳部分
	timestamp1 := id1 >> 12
	timestamp2 := id2 >> 12
	now := time.Now().UnixMilli()

	if timestamp1 > now || timestamp1 < now-1000 {
		t.Errorf("Timestamp1 %d should be close to current time %d", timestamp1, now)
	}
	if timestamp2 > now || timestamp2 < now-1000 {
		t.Errorf("Timestamp2 %d should be close to current time %d", timestamp2, now)
	}
}

func TestRedisGenerator_GenerateWithDefaultOptions(t *testing.T) {
	// 启动 miniredis 模拟 Redis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// 创建 RedisGenerator 使用默认选项
	generator := NewRedisGeneratorWithOptions(&RedisOptions{
		Addr: mr.Addr(),
	})

	// 测试生成ID
	id := generator.Generate()
	if id == 0 {
		t.Error("Generated ID should not be 0")
	}
}

func TestRedisGenerator_GenerateNilOptions(t *testing.T) {
	// 创建 RedisGenerator 使用 nil 选项（会使用默认值）
	generator := NewRedisGeneratorWithOptions(nil)

	// 由于没有真实的Redis服务器，这里应该触发降级方案
	id := generator.Generate()
	if id == 0 {
		t.Error("Generated ID should not be 0 even with fallback")
	}

	// 验证ID的时间戳部分
	timestamp := id >> 12
	now := time.Now().UnixMilli()
	if timestamp > now || timestamp < now-1000 {
		t.Errorf("Timestamp %d should be close to current time %d", timestamp, now)
	}

	// 验证序列号部分为0（降级方案）
	sequence := id & 0xFFF
	if sequence != 0 {
		t.Errorf("Sequence should be 0 in fallback mode, got %d", sequence)
	}
}

func TestRedisGenerator_Concurrency(t *testing.T) {
	// 启动 miniredis 模拟 Redis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	generator := NewRedisGeneratorWithOptions(&RedisOptions{
		Addr:    mr.Addr(),
		KeyName: "test:concurrent",
		Timeout: time.Second,
	})

	// 并发生成ID
	const numGoroutines = 100
	const numIDsPerGoroutine = 10
	
	idChan := make(chan int64, numGoroutines*numIDsPerGoroutine)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numIDsPerGoroutine; j++ {
				idChan <- generator.Generate()
			}
		}()
	}

	// 收集所有ID
	ids := make(map[int64]bool)
	for i := 0; i < numGoroutines*numIDsPerGoroutine; i++ {
		id := <-idChan
		if ids[id] {
			t.Errorf("Duplicate ID found: %d", id)
		}
		ids[id] = true
	}

	// 验证生成了预期数量的唯一ID
	if len(ids) != numGoroutines*numIDsPerGoroutine {
		t.Errorf("Expected %d unique IDs, got %d", numGoroutines*numIDsPerGoroutine, len(ids))
	}
}

func TestFormatTimestamp(t *testing.T) {
	timestamp := int64(1609459200000) // 2021-01-01 00:00:00 UTC
	expected := "20210101080000.000"   // 转换为本地时间
	
	result := formatTimestamp(timestamp)
	if len(result) != len(expected) {
		t.Errorf("Expected format length %d, got %d. Result: %s", len(expected), len(result), result)
	}
}

func BenchmarkRedisGenerator_Generate(b *testing.B) {
	// 启动 miniredis 模拟 Redis
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	generator := NewRedisGeneratorWithOptions(&RedisOptions{
		Addr:    mr.Addr(),
		KeyName: "bench:uid",
		Timeout: time.Second,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			generator.Generate()
		}
	})
}