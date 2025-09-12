package provider

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileProvider_Read(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	testContent := `{"key": "value"}`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	provider, err := NewFileProvider(&FileProviderOptions{
		FilePath: testFile,
	})
	if err != nil {
		t.Fatalf("Failed to create FileProvider: %v", err)
	}
	defer provider.Close()

	data, err := provider.Read()
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != testContent {
		t.Errorf("Expected %s, got %s", testContent, string(data))
	}
}

func TestFileProvider_ReadNonexistentFile(t *testing.T) {
	provider, err := NewFileProvider(&FileProviderOptions{
		FilePath: "/nonexistent/file.json",
	})
	if err != nil {
		t.Fatalf("Failed to create FileProvider: %v", err)
	}
	defer provider.Close()

	_, err = provider.Read()
	if err == nil {
		t.Error("Expected error when reading nonexistent file")
	}
}

func TestFileProvider_OnChange(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	initialContent := `{"key": "value1"}`
	updatedContent := `{"key": "value2"}`

	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	provider, err := NewFileProvider(&FileProviderOptions{
		FilePath: testFile,
	})
	if err != nil {
		t.Fatalf("Failed to create FileProvider: %v", err)
	}
	defer provider.Close()

	changeChan := make(chan []byte, 1)
	provider.OnChange(func(data []byte) error {
		changeChan <- data
		return nil
	})

	time.Sleep(100 * time.Millisecond)

	err = os.WriteFile(testFile, []byte(updatedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	select {
	case data := <-changeChan:
		if string(data) != updatedContent {
			t.Errorf("Expected %s, got %s", updatedContent, string(data))
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for file change notification")
	}
}

func TestFileProvider_InvalidOptions(t *testing.T) {
	_, err := NewFileProvider(nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	_, err = NewFileProvider(&FileProviderOptions{})
	if err == nil {
		t.Error("Expected error for empty file path")
	}
}