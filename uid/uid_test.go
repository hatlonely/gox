package uid

import (
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
		
		// UUID v7 格式检查：默认无连字符，32个十六进制字符
		if len(uuid) != 32 {
			t.Errorf("UUID length should be 32 (without hyphens), got %d", len(uuid))
		}
		
		// 验证只包含十六进制字符
		for _, char := range uuid {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
				t.Errorf("UUID should only contain hex characters, got: %s", uuid)
				break
			}
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