package common

import (
	"os"
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

func TestIsStringInList(t *testing.T) {
	tests := []struct {
		name  string
		value string
		list  []string
		want  bool
	}{
		{
			name:  "ok first",
			value: "this",
			list:  []string{"this", "is", "a", "test"},
			want:  true,
		},
		{
			name:  "ok middle",
			value: "a",
			list:  []string{"this", "is", "a", "test"},
			want:  true,
		},
		{
			name:  "ok end",
			value: "test",
			list:  []string{"this", "is", "a", "test"},
			want:  true,
		},
		{
			name:  "must fail",
			value: "est",
			list:  []string{"this", "is", "a", "test"},
			want:  false,
		},
		{
			name:  "must fail empty",
			value: "",
			list:  []string{"this", "is", "a", "test"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsStringInList(tt.value, tt.list)
			if tt.want != got {
				t.Errorf("got %t, want %t", got, tt.want)
			}
		})
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
