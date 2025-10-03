package procfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCustomCmdlineValue(t *testing.T) {
	tdir := t.TempDir()
	testFile := filepath.Join(tdir, "test_cmdline.txt")

	// Simulated /proc/cmdline content
	cmdlineContent := "int_key=42 str_key=hello float_key=3.14 bool_key=true empty_key flag"
	if err := os.WriteFile(testFile, []byte(cmdlineContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		cmdlineFile  string
		key          string
		defaultValue any
		expected     any
		shouldError  bool
	}{
		{testFile, "int_key", 0, 42, false},                    // Valid int
		{testFile, "str_key", "", "hello", false},              // Valid string
		{testFile, "float_key", 0.0, 3.14, false},              // Valid float64
		{testFile, "bool_key", false, true, false},             // Valid bool
		{testFile, "empty_key", "default", "default", false},   // Empty value should return default
		{testFile, "flag", false, false, false},                // Key without value should return default
		{testFile, "missing_key", "default", "default", false}, // Nonexistent key should return default
		{testFile, "int_key", []string{}, "wrong_type", true},  // Incorrect type should error
		{testFile + "_not_found", "int_key", 0, 0, true},       // Cmdline file not found should error
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			switch tt.defaultValue.(type) {
			case int:
				value, err := getCustomCmdlineValue(tt.cmdlineFile, tt.key, tt.defaultValue.(int))
				if (err != nil) != tt.shouldError || value != tt.expected.(int) {
					t.Errorf("Expected %v, got %v, err: %v", tt.expected, value, err)
				}
			case string:
				value, err := getCustomCmdlineValue(tt.cmdlineFile, tt.key, tt.defaultValue.(string))
				if (err != nil) != tt.shouldError || value != tt.expected.(string) {
					t.Errorf("Expected %v, got %v, err: %v", tt.expected, value, err)
				}
			case float64:
				value, err := getCustomCmdlineValue(tt.cmdlineFile, tt.key, tt.defaultValue.(float64))
				if (err != nil) != tt.shouldError || value != tt.expected.(float64) {
					t.Errorf("Expected %v, got %v, err: %v", tt.expected, value, err)
				}
			case bool:
				value, err := getCustomCmdlineValue(tt.cmdlineFile, tt.key, tt.defaultValue.(bool))
				if (err != nil) != tt.shouldError || value != tt.expected.(bool) {
					t.Errorf("Expected %v, got %v, err: %v", tt.expected, value, err)
				}
			}
		})
	}
}

func TestGetCmdlineValue(t *testing.T) {
	// Basic test to check if /proc/cmdline can be read
	_, err := GetCmdlineValue("nonexistent_key", "default")
	if err != nil {
		t.Errorf("GetCmdlineValue failed: %v", err)
	}
}
