package common

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetEnv(t *testing.T) {
	var got string
	var want string

	// Fallback
	os.Unsetenv("TESTING")
	got = GetEnv("TESTING", "empty")
	want = "empty"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Value
	os.Setenv("TESTING", "something")
	got = GetEnv("TESTING", "anotherthing")
	want = "something"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsDir(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(tmpDir+"/file", []byte{}, 0644); err != nil {
		t.Error(err)
	}

	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{
			name: "Everything is fine",
			dir:  tmpDir,
			want: true,
		},
		{
			name: "Empty path",
			dir:  "",
			want: false,
		},
		{
			name: "Path does not exists",
			dir:  tmpDir + "/abc",
			want: false,
		},
		{
			name: "Path is not a directory",
			dir:  tmpDir + "/file",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDir(tt.dir)
			if tt.want != got {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func Test_isCmdlineDebug(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"fileJustDebug":       "debug",
		"fileStart":           "debug after",
		"fileBetween":         "before debug after",
		"fileEnd":             "something debug",
		"fileEmpty":           "",
		"fileInAnotherCastle": "in another castle",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal("can not write test files")
		}
	}

	type args struct {
		filePath string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "ok", args: args{filePath: filepath.Join(dir, "fileJustDebug")}, want: true},
		{name: "start", args: args{filePath: filepath.Join(dir, "fileStart")}, want: true},
		{name: "between", args: args{filePath: filepath.Join(dir, "fileBetween")}, want: true},
		{name: "end", args: args{filePath: filepath.Join(dir, "fileEnd")}, want: true},
		{name: "empty", args: args{filePath: filepath.Join(dir, "fileEmpty")}, want: false},
		{name: "not-here", args: args{filePath: filepath.Join(dir, "fileInAnotherCastle")}, want: false},
		{name: "file-not-found", args: args{filePath: filepath.Join(dir, "doesnotexists")}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCmdlineDebug(tt.args.filePath); got != tt.want {
				t.Errorf("isCmdlineDebug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckFileExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte("I am a file"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "empty-file"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "directory"), 0644); err != nil {
		t.Fatal(err)
	}

	type args struct {
		filePath string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "ok-file", args: args{filePath: filepath.Join(dir, "file")}, want: true},
		{name: "ok-empty-file", args: args{filePath: filepath.Join(dir, "empty-file")}, want: true},
		{name: "ok-directory", args: args{filePath: filepath.Join(dir, "directory")}, want: true},
		{name: "ko", args: args{filePath: filepath.Join(dir, "nothing")}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckFileExists(tt.args.filePath); got != tt.want {
				t.Errorf("CheckFileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateOrAppend(t *testing.T) {
	// Example struct for testing.
	type Person struct {
		Name string
		Age  int
	}

	// Simple equals function for Person.
	personEquals := func(a, b Person) bool {
		return a.Name == b.Name
	}

	tests := []struct {
		name     string
		items    []Person
		newItem  Person
		expected []Person
	}{
		{
			name:     "empty slice - append new item",
			items:    []Person{},
			newItem:  Person{Name: "Alice", Age: 30},
			expected: []Person{{Name: "Alice", Age: 30}},
		},
		{
			name: "no match - append new item",
			items: []Person{
				{Name: "Bob", Age: 25},
			},
			newItem: Person{Name: "Charlie", Age: 40},
			expected: []Person{
				{Name: "Bob", Age: 25},
				{Name: "Charlie", Age: 40},
			},
		},
		{
			name: "match found - update existing item",
			items: []Person{
				{Name: "Alice", Age: 30},
				{Name: "Bob", Age: 25},
			},
			newItem: Person{Name: "Bob", Age: 26}, // updated age.
			expected: []Person{
				{Name: "Alice", Age: 30},
				{Name: "Bob", Age: 26},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateOrAppend(tt.items, tt.newItem, personEquals)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestGetOrDefault(t *testing.T) {
	t.Run("bool pointer is nil", func(t *testing.T) {
		var ptr *bool
		result := GetOrDefault(ptr, true)
		if result != true {
			t.Errorf("expected true, got %v", result)
		}
	})

	t.Run("bool pointer is not nil", func(t *testing.T) {
		val := false
		result := GetOrDefault(&val, true)
		if result != false {
			t.Errorf("expected false, got %v", result)
		}
	})

	t.Run("string pointer is nil", func(t *testing.T) {
		var name *string
		result := GetOrDefault(name, "default")
		if result != "default" {
			t.Errorf("expected 'default', got %q", result)
		}
	})

	t.Run("string pointer is not nil", func(t *testing.T) {
		value := "hello"
		result := GetOrDefault(&value, "default")
		if result != "hello" {
			t.Errorf("expected 'hello', got %q", result)
		}
	})

	t.Run("int pointer is nil", func(t *testing.T) {
		var number *int
		result := GetOrDefault(number, 42)
		if result != 42 {
			t.Errorf("expected 42, got %v", result)
		}
	})

	t.Run("int pointer is not nil", func(t *testing.T) {
		val := 99
		result := GetOrDefault(&val, 0)
		if result != 99 {
			t.Errorf("expected 99, got %v", result)
		}
	})
}
