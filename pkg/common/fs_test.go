// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

package common

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestIsPathExists(t *testing.T) {
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
			if got := IsPathExists(tt.args.filePath); got != tt.want {
				t.Errorf("CheckFileExists() = %v, want %v", got, tt.want)
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

// --- Tests for ForEachLineOfReader ---

func TestForEachLineOfReader(t *testing.T) {
	input := "line1\nline2\nline3"
	reader := strings.NewReader(input)

	var lines []string
	err := ForEachLineOfReader(reader, func(line string) error {
		lines = append(lines, line)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"line1", "line2", "line3"}
	if !slices.Equal(lines, expected) {
		t.Errorf("expected %v, got %v", expected, lines)
	}
}

func TestForEachLineOfReader_ErrorInCallback(t *testing.T) {
	reader := strings.NewReader("a\nb\nc")
	count := 0

	err := ForEachLineOfReader(reader, func(line string) error {
		count++
		if line == "b" {
			return errors.New("forced error")
		}
		return nil
	})

	if err == nil || err.Error() != "forced error" {
		t.Errorf("expected forced error, got %v", err)
	}

	if count != 2 {
		t.Errorf("expected to process 2 lines, got %d", count)
	}
}

// errReader is a dummy reader that always fails
type errReader struct{}

func (e *errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestForEachLineOfReader_InvalidInput(t *testing.T) {
	// Reader that always errors
	r := io.Reader(&errReader{})
	err := ForEachLineOfReader(r, func(line string) error { return nil })
	if err == nil {
		t.Errorf("expected error for invalid reader, got nil")
	}
}

// --- Tests for ForEachLineOfFilename ---

func TestForEachLineOfFilename(t *testing.T) {
	content := "x\ny\nz"
	tmpfile, err := os.CreateTemp("", "foreachline_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	var lines []string
	err = ForEachLineOfFilepath(tmpfile.Name(), func(line string) error {
		lines = append(lines, line)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"x", "y", "z"}
	if !slices.Equal(lines, expected) {
		t.Errorf("expected %v, got %v", expected, lines)
	}
}

func TestForEachLineOfFilename_FileNotExist(t *testing.T) {
	err := ForEachLineOfFilepath("nonexistent_file.txt", func(line string) error { return nil })
	if err == nil {
		t.Errorf("expected error for nonexistent file, got nil")
	}
}

// --- Tests for CreateSymlink ---

func TestCreateSymlink_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "target.txt")
	linkFile := filepath.Join(tmpDir, "link.txt")

	// Create target file
	if err := os.WriteFile(targetFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := CreateSymlink(linkFile, targetFile, false, os.Getuid(), os.Getgid(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify symlink
	info, err := os.Lstat(linkFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink, got mode %v", info.Mode())
	}

	resolved, err := os.Readlink(linkFile)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != targetFile {
		t.Errorf("expected target %q, got %q", targetFile, resolved)
	}
}

func TestCreateSymlink_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "target.txt")
	linkFile := filepath.Join(tmpDir, "link.txt")

	os.WriteFile(targetFile, []byte("a"), 0644)
	os.WriteFile(linkFile, []byte("old"), 0644)

	err := CreateSymlink(linkFile, targetFile, true, os.Getuid(), os.Getgid(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Lstat(linkFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink after overwrite, got mode %v", info.Mode())
	}
}

func TestCreateSymlink_NoOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "target.txt")
	linkFile := filepath.Join(tmpDir, "link.txt")

	os.WriteFile(targetFile, []byte("a"), 0644)
	os.WriteFile(linkFile, []byte("b"), 0644)

	err := CreateSymlink(linkFile, targetFile, false, os.Getuid(), os.Getgid(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not overwrite the existing file
	info, err := os.Lstat(linkFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("did not expect symlink when overwrite=false")
	}
}

func TestCreateSymlink_Hardlink(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "target.txt")
	hardlinkFile := filepath.Join(tmpDir, "hardlink.txt")

	os.WriteFile(targetFile, []byte("content"), 0644)

	err := CreateSymlink(hardlinkFile, targetFile, false, os.Getuid(), os.Getgid(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify hardlink: both files should have the same inode
	t1, _ := os.Stat(targetFile)
	t2, _ := os.Stat(hardlinkFile)

	if !os.SameFile(t1, t2) {
		t.Errorf("expected hardlink files to refer to same inode")
	}
}

func TestCreateSymlink_InvalidTarget(t *testing.T) {
	tmpDir := t.TempDir()
	linkFile := filepath.Join(tmpDir, "link.txt")

	err := CreateSymlink(linkFile, "/nonexistent/target", false, os.Getuid(), os.Getgid(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSymlink_RemoveFileError(t *testing.T) {
	tmpDir := t.TempDir()

	target := filepath.Join(tmpDir, "target.txt")
	link := filepath.Join(tmpDir, "link.txt")

	os.WriteFile(target, []byte("ok"), 0644)
	os.WriteFile(link, []byte("will_fail_remove"), 0444) // read-only

	// Set directory as read-only.
	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(tmpDir, 0755)

	// Remove should fail because file is read-only
	err := CreateSymlink(link, target, true, os.Getuid(), os.Getgid(), false)
	if err == nil {
		t.Fatalf("expected error when removing read-only file, got nil")
	}
}

func TestCreateSymlink_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()

	// Set directory as read-only.
	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(tmpDir, 0755)

	target := filepath.Join(tmpDir, "target.txt")

	path := filepath.Join(tmpDir, "subdir", "link.txt")
	// The subdir can't be created because parent has 0555 permissions
	err := CreateSymlink(path, target, false, os.Getuid(), os.Getgid(), false)
	if err == nil {
		t.Fatalf("expected mkdir error, got nil")
	}
}

func TestCreateSymlink_HardlinkError(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "missing_target.txt")
	link := filepath.Join(tmpDir, "hardlink.txt")

	// Hardlink should fail because target doesn't exist
	err := CreateSymlink(link, target, false, os.Getuid(), os.Getgid(), true)
	if err == nil {
		t.Fatalf("expected error creating hardlink to nonexistent target")
	}
}

func TestCreateSymlink_LchownError(t *testing.T) {
	// Mock Lchown.
	old := osLchown
	defer func() { osLchown = old }()
	osLchown = func(path string, uid, gid int) error {
		return errors.New("forced Lchown error")
	}

	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target.txt")
	link := filepath.Join(tmpDir, "link.txt")

	os.WriteFile(target, []byte("ok"), 0644)

	// Create symlink normally, but then force Lchown to fail with invalid uid/gid
	err := CreateSymlink(link, target, false, -1, -1, false)
	if err == nil || err.Error() != "forced Lchown error" {
		t.Fatalf("expected forced Lchown error, got %v:", err)
	}
}

func TestReadFileAsString(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	tmpContent := "content of test.txt"
	if err := os.WriteFile(tmpFile, []byte(tmpContent), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		filepath string
		want     string
		wantErr  error
	}{
		{"success", tmpFile, tmpContent, nil},
		{"file_notfound", filepath.Join(tmpDir, "nonexistent.txt"), "", os.ErrNotExist},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ReadFileAsString(tt.filepath)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("want: %q, got %q", tt.wantErr, err)
			}

			if data != tt.want {
				t.Fatalf("want: %q, got %q", tt.want, data)
			}
		})
	}
}
