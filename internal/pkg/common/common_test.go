package common

import (
	"os"
	"path/filepath"
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

func TestWriteTemplate(t *testing.T) {
	// logrus.SetOutput(ioutil.Discard)
	tmpDir := t.TempDir()

	// Write OK template
	fnOk := "ok.tmpl"
	tOk := []byte("Template example\n{{ .Text }}\nAnother line\n")
	if err := os.WriteFile(filepath.Join(tmpDir, fnOk), tOk, 0644); err != nil {
		t.Error(err)
	}

	// Write bad template
	fnBadTmpl := "bad.tmpl"
	tBadTmpl := []byte("This is a {{ bad template {{")
	if err := os.WriteFile(filepath.Join(tmpDir, fnBadTmpl), tBadTmpl, 0644); err != nil {
		t.Error(err)
	}

	fs := os.DirFS(tmpDir)
	data := struct {
		Text string
	}{"this is a test"}

	t.Run("ok", func(t *testing.T) {
		if err := WriteTemplate(filepath.Join(tmpDir, "ok.txt"), fs, fnOk, data); err != nil {
			t.Error(err)
		}
	})
	t.Run("bad template", func(t *testing.T) {
		if err := WriteTemplate(filepath.Join(tmpDir, "badTemplate.txt"), fs, fnBadTmpl, data); err == nil {
			t.Error("expected bad template error")
		}
	})
	//TODO
	// t.Run("bad render", func(t *testing.T) {
	// 	if err := WriteTemplate(filepath.Join(tmpDir, "badRender.txt"), fs, fnOk, nil); err == nil {
	// 		t.Error("expected bad template render")
	// 	}
	// })
}
