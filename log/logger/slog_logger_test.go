package logger

import (
	"os"
	"strings"
	"testing"

	"github.com/hatlonely/gox/log/writer"
	"github.com/hatlonely/gox/ref"
)

func TestNewLogWithOptions(t *testing.T) {
	tests := []struct {
		name    string
		options *SLogOptions
		wantErr bool
	}{
		{
			name:    "nil options",
			options: nil,
			wantErr: true,
		},
		{
			name: "default console output",
			options: &SLogOptions{
				Level: "info",
			},
			wantErr: false,
		},
		{
			name: "console output with options",
			options: &SLogOptions{
				Level:  "debug",
				Format: "json",
				Output: &ref.TypeOptions{
					Namespace: "github.com/hatlonely/gox/log/writer",
					Type:      "ConsoleWriter",
					Options: &writer.ConsoleWriterOptions{
						Color:  false,
						Target: "stdout",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			options: &SLogOptions{
				Level: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			options: &SLogOptions{
				Level:  "info",
				Format: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewSLogWithOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("NewLogWithOptions() returned nil logger without error")
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		level   string
		wantErr bool
	}{
		{"debug", false},
		{"info", false},
		{"warn", false},
		{"warning", false},
		{"error", false},
		{"DEBUG", false}, // 测试大小写不敏感
		{"INFO", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			_, err := parseLevel(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLevel(%q) error = %v, wantErr %v", tt.level, err, tt.wantErr)
			}
		})
	}
}

func TestConsoleWriter(t *testing.T) {
	w, err := writer.NewConsoleWriterWithOptions(&writer.ConsoleWriterOptions{
		Color:  true,
		Target: "stdout",
	})
	if err != nil {
		t.Fatalf("NewConsoleWriter() error = %v", err)
	}

	testData := []byte("test log message\n")
	n, err := w.Write(testData)
	if err != nil {
		t.Errorf("ConsoleWriter.Write() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("ConsoleWriter.Write() wrote %d bytes, want %d", n, len(testData))
	}

	err = w.Close()
	if err != nil {
		t.Errorf("ConsoleWriter.Close() error = %v", err)
	}
}

func TestFileWriter(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	logFile := tempDir + "/test.log"

	w, err := writer.NewFileWriterWithOptions(&writer.FileWriterOptions{
		Path: logFile,
	})
	if err != nil {
		t.Fatalf("NewFileWriter() error = %v", err)
	}
	defer w.Close()

	testData := []byte("test log message\n")
	n, err := w.Write(testData)
	if err != nil {
		t.Errorf("FileWriter.Write() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("FileWriter.Write() wrote %d bytes, want %d", n, len(testData))
	}

	// 检查文件内容
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}
	if !strings.Contains(string(content), "test log message") {
		t.Errorf("Log file doesn't contain expected message")
	}
}

func TestMultiWriter(t *testing.T) {
	tempDir := t.TempDir()
	logFile := tempDir + "/multi_test.log"

	w, err := writer.NewMultiWriterWithOptions(&writer.MultiWriterOptions{
		Writers: []ref.TypeOptions{
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "ConsoleWriter",
				Options: &writer.ConsoleWriterOptions{
					Color:  false,
					Target: "stdout",
				},
			},
			{
				Namespace: "github.com/hatlonely/gox/log/writer",
				Type:      "FileWriter",
				Options: &writer.FileWriterOptions{
					Path: logFile,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewMultiWriter() error = %v", err)
	}
	defer w.Close()

	testData := []byte("multi writer test\n")
	n, err := w.Write(testData)
	if err != nil {
		t.Errorf("MultiWriter.Write() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("MultiWriter.Write() wrote %d bytes, want %d", n, len(testData))
	}

	// 检查文件是否被写入
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}
	if !strings.Contains(string(content), "multi writer test") {
		t.Errorf("Log file doesn't contain expected message")
	}
}
