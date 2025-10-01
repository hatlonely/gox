package strgen

import (
	"regexp"
	"testing"

	"github.com/google/uuid"
)

func TestNewUUIDGeneratorWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		options  *UUIDOptions
		expected string
	}{
		{
			name:     "nil options should use default v4",
			options:  nil,
			expected: "v4",
		},
		{
			name:     "empty version should use default v4",
			options:  &UUIDOptions{Version: ""},
			expected: "v4",
		},
		{
			name:     "v1 version",
			options:  &UUIDOptions{Version: "v1"},
			expected: "v1",
		},
		{
			name:     "v4 version",
			options:  &UUIDOptions{Version: "v4"},
			expected: "v4",
		},
		{
			name:     "v6 version",
			options:  &UUIDOptions{Version: "v6"},
			expected: "v6",
		},
		{
			name:     "v7 version",
			options:  &UUIDOptions{Version: "v7"},
			expected: "v7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewUUIDGeneratorWithOptions(tt.options)
			if gen.version != tt.expected {
				t.Errorf("expected version %s, got %s", tt.expected, gen.version)
			}
		})
	}
}

func TestUUIDGenerator_Generate(t *testing.T) {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	tests := []struct {
		name    string
		version string
	}{
		{"v1", "v1"},
		{"v4", "v4"},
		{"v6", "v6"},
		{"v7", "v7"},
		{"invalid version fallback to v4", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewUUIDGeneratorWithOptions(&UUIDOptions{Version: tt.version})
			
			result := gen.Generate()
			
			if !uuidRegex.MatchString(result) {
				t.Errorf("generated UUID %s does not match expected format", result)
			}
			
			_, err := uuid.Parse(result)
			if err != nil {
				t.Errorf("generated UUID %s is not valid: %v", result, err)
			}
		})
	}
}

func TestUUIDGenerator_GenerateUniqueness(t *testing.T) {
	gen := NewUUIDGeneratorWithOptions(&UUIDOptions{Version: "v4"})
	
	generated := make(map[string]bool)
	iterations := 1000
	
	for i := 0; i < iterations; i++ {
		result := gen.Generate()
		if generated[result] {
			t.Errorf("duplicate UUID generated: %s", result)
		}
		generated[result] = true
	}
	
	if len(generated) != iterations {
		t.Errorf("expected %d unique UUIDs, got %d", iterations, len(generated))
	}
}

func TestUUIDGenerator_VersionSpecificFormat(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		expectedPrefix string
	}{
		{"v1 has version 1", "v1", "1"},
		{"v4 has version 4", "v4", "4"},
		{"v6 has version 6", "v6", "6"},
		{"v7 has version 7", "v7", "7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewUUIDGeneratorWithOptions(&UUIDOptions{Version: tt.version})
			result := gen.Generate()
			
			parsed, err := uuid.Parse(result)
			if err != nil {
				t.Fatalf("failed to parse UUID: %v", err)
			}
			
			versionByte := parsed[6] >> 4
			expectedVersion := tt.expectedPrefix[0] - '0'
			
			if versionByte != byte(expectedVersion) {
				t.Errorf("expected UUID version %s, got version %d", tt.expectedPrefix, versionByte)
			}
		})
	}
}

func BenchmarkUUIDGenerator_Generate(b *testing.B) {
	gen := NewUUIDGeneratorWithOptions(&UUIDOptions{Version: "v4"})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkUUIDGenerator_GenerateV7(b *testing.B) {
	gen := NewUUIDGeneratorWithOptions(&UUIDOptions{Version: "v7"})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}