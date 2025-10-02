package uid

import (
	"strings"
	"testing"
)

func TestNewIntGenerator(t *testing.T) {
	generator := NewIntGenerator()
	if generator == nil {
		t.Fatal("NewIntGenerator() returned nil")
	}

	// 测试生成多个ID
	ids := make(map[int64]bool)
	for i := 0; i < 1000; i++ {
		id := generator.Generate()
		if id <= 0 {
			t.Errorf("Generated ID should be positive, got %d", id)
		}
		if ids[id] {
			t.Errorf("Generated duplicate ID: %d", id)
		}
		ids[id] = true
	}
}

func TestNewStrGenerator(t *testing.T) {
	generator := NewStrGenerator()
	if generator == nil {
		t.Fatal("NewStrGenerator() returned nil")
	}

	// 测试生成多个UUID
	uuids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		uuid := generator.Generate()
		if uuid == "" {
			t.Error("Generated UUID should not be empty")
		}
		
		// UUID v7 格式检查：8-4-4-4-12 字符
		if len(uuid) != 36 {
			t.Errorf("UUID length should be 36, got %d", len(uuid))
		}
		
		if strings.Count(uuid, "-") != 4 {
			t.Errorf("UUID should have 4 hyphens, got %d", strings.Count(uuid, "-"))
		}
		
		// 检查版本号（第15个字符应该是7）
		if uuid[14] != '7' {
			t.Errorf("Expected UUID v7, but version character is %c", uuid[14])
		}
		
		if uuids[uuid] {
			t.Errorf("Generated duplicate UUID: %s", uuid)
		}
		uuids[uuid] = true
	}
}

func BenchmarkNewIntGenerator(b *testing.B) {
	generator := NewIntGenerator()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		generator.Generate()
	}
}

func BenchmarkNewStrGenerator(b *testing.B) {
	generator := NewStrGenerator()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		generator.Generate()
	}
}