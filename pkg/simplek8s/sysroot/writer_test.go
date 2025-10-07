package sysroot

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"
)

func TestEnsureWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	filename := filepath.Join(tmpDir, "subdir", "testfile.txt")
	content := []byte("hello world")
	mode := os.FileMode(0644)

	// Skip testing chown by setting it as current user.
	var uid, gid int
	if current, err := user.Current(); err != nil {
		t.Error(err)
	} else {
		var err error
		if uid, err = strconv.Atoi(current.Uid); err != nil {
			t.Error(err)
		}
		if gid, err = strconv.Atoi(current.Gid); err != nil {
			t.Error(err)
		}
	}

	err := ensureWriteFile(filename, content, mode, uid, gid)
	if err != nil {
		t.Fatalf("EnsureWriteFile returned error: %v", err)
	}

	// Verify file was created.
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("file was not created: %v", err)
	}

	// Verify permissions.
	if info.Mode().Perm() != mode {
		t.Errorf("expected mode %v, got %v", mode, info.Mode().Perm())
	}

	// Verify content.
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("expected content %q, got %q", content, data)
	}

	// Verify MkdirAll fails with a invalid name.
	invalidDir := string([]byte{0}) // nombre de directorio inválido
	invalidFilename := filepath.Join(invalidDir, "file.txt")
	if err := ensureWriteFile(invalidFilename, content, mode, 0, 0); err == nil {
		t.Error("expected error from MkdirAll, got nil")
	}

	// Verify WriteFile fails with RO directory.
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0444); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}
	readOnlyfilename := filepath.Join(readOnlyDir, "file.txt")
	if err := ensureWriteFile(readOnlyfilename, content, mode, 0, 0); err == nil {
		t.Error("expected error from WriteFile due to readonly dir, got nil")
	}
}
