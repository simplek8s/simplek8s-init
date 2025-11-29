// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package procfs_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"simplek8s/pkg/linux/procfs"
)

type testCase struct {
	name         string
	cmdline      string
	key          string
	defaultValue any
	want         any
	wantErr      bool
}

func TestGetCmdlineValue(t *testing.T) {
	tests := []testCase{
		// string tests:
		{
			name:         "string value found",
			cmdline:      "root=/dev/sda1 console=tty0",
			key:          "root",
			defaultValue: "",
			want:         "/dev/sda1",
			wantErr:      false,
		},
		{
			name:         "string value not found returns default",
			cmdline:      "root=/dev/sda1",
			key:          "missing",
			defaultValue: "default",
			want:         "default",
			wantErr:      false,
		},
		{
			name:         "string key without value returns default",
			cmdline:      "quiet root=/dev/sda1",
			key:          "quiet",
			defaultValue: "defaultval",
			want:         "defaultval",
			wantErr:      false,
		},

		// bool tests:
		{
			name:         "bool key without value returns true",
			cmdline:      "quiet debug",
			key:          "quiet",
			defaultValue: false,
			want:         true,
			wantErr:      false,
		},
		{
			name:         "bool value true",
			cmdline:      "debug=true",
			key:          "debug",
			defaultValue: false,
			want:         true,
			wantErr:      false,
		},
		{
			name:         "bool value false",
			cmdline:      "debug=false",
			key:          "debug",
			defaultValue: true,
			want:         false,
			wantErr:      false,
		},
		{
			name:         "bool invalid value returns error",
			cmdline:      "debug=invalid",
			key:          "debug",
			defaultValue: false,
			want:         false,
			wantErr:      true,
		},

		// int tests:
		{
			name:         "int value found",
			cmdline:      "loglevel=7",
			key:          "loglevel",
			defaultValue: 0,
			want:         7,
			wantErr:      false,
		},
		{
			name:         "int invalid value returns error",
			cmdline:      "loglevel=abc",
			key:          "loglevel",
			defaultValue: 0,
			want:         0,
			wantErr:      true,
		},
		{
			name:         "int key without value returns default",
			cmdline:      "loglevel",
			key:          "loglevel",
			defaultValue: 5,
			want:         5,
			wantErr:      false,
		},

		// float32 tests:
		{
			name:         "float32 value found",
			cmdline:      "scale=1.5",
			key:          "scale",
			defaultValue: float32(0),
			want:         float32(1.5),
			wantErr:      false,
		},
		{
			name:         "float32 invalid value returns error",
			cmdline:      "scale=invalid",
			key:          "scale",
			defaultValue: float32(0),
			want:         float32(0),
			wantErr:      true,
		},

		// float64 tests:
		{
			name:         "float64 value found",
			cmdline:      "ratio=3.14159",
			key:          "ratio",
			defaultValue: float64(0),
			want:         3.14159,
			wantErr:      false,
		},
		{
			name:         "float64 invalid value returns error",
			cmdline:      "ratio=invalid",
			key:          "ratio",
			defaultValue: float64(0),
			want:         float64(0),
			wantErr:      true,
		},

		// []string tests:
		{
			name:         "string slice value found",
			cmdline:      "modules=mod1,mod2,mod3",
			key:          "modules",
			defaultValue: []string{},
			want:         []string{"mod1", "mod2", "mod3"},
			wantErr:      false,
		},
		{
			name:         "string slice with spaces trimmed",
			cmdline:      "modules=mod1,mod2,mod3",
			key:          "modules",
			defaultValue: []string{},
			want:         []string{"mod1", "mod2", "mod3"},
			wantErr:      false,
		},
		{
			name:         "string slice single value",
			cmdline:      "module=single",
			key:          "module",
			defaultValue: []string{},
			want:         []string{"single"},
			wantErr:      false,
		},

		// []int tests:
		{
			name:         "int slice value found",
			cmdline:      "cpus=0,1,2,3",
			key:          "cpus",
			defaultValue: []int{},
			want:         []int{0, 1, 2, 3},
			wantErr:      false,
		},
		{
			name:         "int slice invalid value returns error",
			cmdline:      "cpus=0,1,abc,3",
			key:          "cpus",
			defaultValue: []int{},
			want:         []int{},
			wantErr:      true,
		},

		// []float32 tests:
		{
			name:         "float32 slice value found",
			cmdline:      "ratios=1.1,2.2,3.3",
			key:          "ratios",
			defaultValue: []float32{},
			want:         []float32{1.1, 2.2, 3.3},
			wantErr:      false,
		},
		{
			name:         "float32 slice invalid value returns error",
			cmdline:      "ratios=1.1,invalid,3.3",
			key:          "ratios",
			defaultValue: []float32{},
			want:         []float32{},
			wantErr:      true,
		},

		// []float64 tests:
		{
			name:         "float64 slice value found",
			cmdline:      "values=1.1,2.2,3.3",
			key:          "values",
			defaultValue: []float64{},
			want:         []float64{1.1, 2.2, 3.3},
			wantErr:      false,
		},
		{
			name:         "float64 slice invalid value returns error",
			cmdline:      "values=1.1,invalid,3.3",
			key:          "values",
			defaultValue: []float64{},
			want:         []float64{},
			wantErr:      true,
		},

		// []bool tests:
		{
			name:         "bool slice value found",
			cmdline:      "flags=true,false,true",
			key:          "flags",
			defaultValue: []bool{},
			want:         []bool{true, false, true},
			wantErr:      false,
		},
		{
			name:         "bool slice invalid value returns error",
			cmdline:      "flags=true,invalid,false",
			key:          "flags",
			defaultValue: []bool{},
			want:         []bool{},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute test and verify.
			runTestCase(t, tt.cmdline, tt)
		})
	}
}

// runTestCase runs a test case based on the type of defaultValue.
func runTestCase(t *testing.T, cmdline string, tt testCase) {
	t.Helper()

	switch tt.defaultValue.(type) {
	case string:
		testTypedValue(t, tt, func(key string, def string) (string, error) {
			return procfs.GetCmdlineValue(cmdline, key, def)
		})
	case bool:
		testTypedValue(t, tt, func(key string, def bool) (bool, error) {
			return procfs.GetCmdlineValue(cmdline, key, def)
		})
	case int:
		testTypedValue(t, tt, func(key string, def int) (int, error) {
			return procfs.GetCmdlineValue(cmdline, key, def)
		})
	case float32:
		testTypedValue(t, tt, func(key string, def float32) (float32, error) {
			return procfs.GetCmdlineValue(cmdline, key, def)
		})
	case float64:
		testTypedValue(t, tt, func(key string, def float64) (float64, error) {
			return procfs.GetCmdlineValue(cmdline, key, def)
		})
	case []string:
		testTypedValue(t, tt, func(key string, def []string) ([]string, error) {
			return procfs.GetCmdlineValue(cmdline, key, def)
		})
	case []int:
		testTypedValue(t, tt, func(key string, def []int) ([]int, error) {
			return procfs.GetCmdlineValue(cmdline, key, def)
		})
	case []float32:
		testTypedValue(t, tt, func(key string, def []float32) ([]float32, error) {
			return procfs.GetCmdlineValue(cmdline, key, def)
		})
	case []float64:
		testTypedValue(t, tt, func(key string, def []float64) ([]float64, error) {
			return procfs.GetCmdlineValue(cmdline, key, def)
		})
	case []bool:
		testTypedValue(t, tt, func(key string, def []bool) ([]bool, error) {
			return procfs.GetCmdlineValue(cmdline, key, def)
		})
	}
}

// testTypedValue is a generic function to test values of any type.
func testTypedValue[T any](t *testing.T, tt testCase, fn func(string, T) (T, error)) {
	t.Helper()

	got, err := fn(tt.key, tt.defaultValue.(T))

	if (err != nil) != tt.wantErr {
		t.Errorf("GetCmdlineValue() error = %v, wantErr %v", err, tt.wantErr)
		return
	}

	if !reflect.DeepEqual(got, tt.want.(T)) {
		t.Errorf("GetCmdlineValue() = %v, want %v", got, tt.want)
	}
}

func TestParseScalarUnsupportedType(t *testing.T) {
	type unsupported struct{}
	_, err := procfs.GetCmdlineValue("key=unsupported", "key", unsupported{})
	if err == nil {
		t.Error("Expected error for unsupported type, got nil")
	}
	if err.Error() != "unsupported scalar type" {
		t.Errorf("Expected 'unsupported scalar type', got %v", err.Error())
	}
}

func TestGetCmdline_Success(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "cmdline")
	expectedContent := "BOOT_IMAGE=/vmlinuz root=/dev/sda1\n"

	if err := os.WriteFile(tmpFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Override filepath
	originalPath := procfs.CmdlineFilepath
	t.Cleanup(func() { procfs.CmdlineFilepath = originalPath })
	procfs.CmdlineFilepath = tmpFile

	// Test
	result, err := procfs.GetCmdline()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result != expectedContent {
		t.Errorf("Expected %q, got %q", expectedContent, result)
	}
}

// Point to non-existent file
func TestGetCmdline_Error(t *testing.T) {
	// Override filepath
	originalPath := procfs.CmdlineFilepath
	t.Cleanup(func() { procfs.CmdlineFilepath = originalPath })
	procfs.CmdlineFilepath = "/nonexistent/path/cmdline"

	// Test
	result, err := procfs.GetCmdline()

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty string on error, got %q", result)
	}
}
