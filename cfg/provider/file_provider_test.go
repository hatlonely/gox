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

	provider, err := NewFileProviderWithOptions(&FileProviderOptions{
		FilePath: testFile,
	})
	if err != nil {
		t.Fatalf("Failed to create FileProvider: %v", err)
	}
	defer provider.Close()

	data, err := provider.Load()
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != testContent {
		t.Errorf("Expected %s, got %s", testContent, string(data))
	}
}

func TestFileProvider_ReadNonexistentFile(t *testing.T) {
	provider, err := NewFileProviderWithOptions(&FileProviderOptions{
		FilePath: "/nonexistent/file.json",
	})
	if err != nil {
		t.Fatalf("Failed to create FileProvider: %v", err)
	}
	defer provider.Close()

	_, err = provider.Load()
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

	provider, err := NewFileProviderWithOptions(&FileProviderOptions{
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

func TestFileProvider_Save(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	testContent := `{"key": "new_value"}`

	provider, err := NewFileProviderWithOptions(&FileProviderOptions{
		FilePath: testFile,
	})
	if err != nil {
		t.Fatalf("Failed to create FileProvider: %v", err)
	}
	defer provider.Close()

	err = provider.Save([]byte(testContent))
	if err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	data, err := provider.Load()
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if string(data) != testContent {
		t.Errorf("Expected %s, got %s", testContent, string(data))
	}
}

func TestFileProvider_SaveInvalidPath(t *testing.T) {
	provider, err := NewFileProviderWithOptions(&FileProviderOptions{
		FilePath: "/invalid/path/test.json",
	})
	if err != nil {
		t.Fatalf("Failed to create FileProvider: %v", err)
	}
	defer provider.Close()

	err = provider.Save([]byte("test"))
	if err == nil {
		t.Error("Expected error when saving to invalid path")
	}
}

func TestFileProvider_MultipleOnChange(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	initialContent := `{"key": "value1"}`
	updatedContent := `{"key": "value2"}`

	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	provider, err := NewFileProviderWithOptions(&FileProviderOptions{
		FilePath: testFile,
	})
	if err != nil {
		t.Fatalf("Failed to create FileProvider: %v", err)
	}
	defer provider.Close()

	// 注册多个回调函数
	changeChan1 := make(chan []byte, 1)
	changeChan2 := make(chan []byte, 1)
	changeChan3 := make(chan []byte, 1)

	provider.OnChange(func(data []byte) error {
		changeChan1 <- data
		return nil
	})

	provider.OnChange(func(data []byte) error {
		changeChan2 <- data
		return nil
	})

	provider.OnChange(func(data []byte) error {
		changeChan3 <- data
		return nil
	})

	time.Sleep(100 * time.Millisecond)

	// 更新文件
	err = os.WriteFile(testFile, []byte(updatedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// 验证所有回调都被调用
	select {
	case data := <-changeChan1:
		if string(data) != updatedContent {
			t.Errorf("Callback 1: Expected %s, got %s", updatedContent, string(data))
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for callback 1")
	}

	select {
	case data := <-changeChan2:
		if string(data) != updatedContent {
			t.Errorf("Callback 2: Expected %s, got %s", updatedContent, string(data))
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for callback 2")
	}

	select {
	case data := <-changeChan3:
		if string(data) != updatedContent {
			t.Errorf("Callback 3: Expected %s, got %s", updatedContent, string(data))
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for callback 3")
	}
}

func TestFileProvider_InvalidOptions(t *testing.T) {
	_, err := NewFileProviderWithOptions(nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	_, err = NewFileProviderWithOptions(&FileProviderOptions{})
	if err == nil {
		t.Error("Expected error for empty file path")
	}
}
