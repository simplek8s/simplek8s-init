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

package sysroot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateLegacySymlinks_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create referred symlinks.
	targets := []string{
		"usr/bin",
		"usr/sbin",
		"usr/lib",
	}
	for _, d := range targets {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0o755); err != nil {
			t.Fatalf("cannot create directory %s: %v", d, err)
		}
	}

	// Run the function under test.
	if err := CreateLegacySymlinks(tmpDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that each symlink was created and points to the expected target.
	for _, l := range []struct {
		name   string
		target string
	}{
		{"bin", "usr/bin"},
		{"sbin", "usr/sbin"},
		{"lib", "usr/lib"},
		{"lib64", "lib"},
	} {
		symlinkPath := filepath.Join(tmpDir, l.name)
		info, err := os.Lstat(symlinkPath)
		if err != nil {
			t.Fatalf("symlink %s does not exist: %v", symlinkPath, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("file %s is not a symlink", symlinkPath)
		}
		gotTarget, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatalf("cannot readlink %s: %v", symlinkPath, err)
		}
		if gotTarget != l.target {
			t.Fatalf("symlink %s points to %q, want %q", symlinkPath, gotTarget, l.target)
		}
	}
}

func TestCreateLegacySymlinks_SymlinkExistsError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file at the location where the first symlink should be created.
	// This forces os.Symlink to fail with EEXIST.
	binPath := filepath.Join(tmpDir, "bin")
	if err := os.WriteFile(binPath, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("cannot create file %s: %v", binPath, err)
	}

	// It should return an error because the symlink cannot be created.
	err := CreateLegacySymlinks(tmpDir)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	// The error returned by the function is the formatted message.
	expectedPrefix := "cannot create symlink"
	if !strings.HasPrefix(err.Error(), expectedPrefix) {
		t.Fatalf("unexpected error message: %q, want prefix %q", err.Error(), expectedPrefix)
	}

	// Verify that the file we created remains and that no further symlinks
	// were created.
	if _, errStat := os.Stat(filepath.Join(tmpDir, "sbin")); !os.IsNotExist(errStat) {
		t.Fatalf("unexpected file %s exists", filepath.Join(tmpDir, "sbin"))
	}
}
