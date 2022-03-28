package common

import (
	"os"
	"testing"
)

/*
 * unused
 *
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
*/

func TestIsDir(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(tmpDir+"/file", []byte{}, 0644); err != nil {
		t.Error(err)
	}

	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{
			name:    "Everything is fine",
			dir:     tmpDir,
			wantErr: false,
		},
		{
			name:    "Empty path",
			dir:     "",
			wantErr: true,
		},
		{
			name:    "Path does not exists",
			dir:     tmpDir + "/abc",
			wantErr: true,
		},
		{
			name:    "Path is not a directory",
			dir:     tmpDir + "/file",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDir(tt.dir)
			if tt.wantErr && got == nil {
				t.Errorf("wantErr %v, got %v", tt.wantErr, got)
			} else if !tt.wantErr && got != nil {
				t.Errorf("wantErr %v, got %v", tt.wantErr, got)
			}
		})
	}
}
