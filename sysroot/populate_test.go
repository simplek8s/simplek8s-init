package sysroot

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openlyinc/pointy"
)

type fsType int

const (
	fsType_DIRECTORY fsType = iota
	fsType_FILE
	fsType_SYMLINK
)

var testFS = []struct {
	fsType fsType
	name   string
	data   []byte
	perm   *fs.FileMode
	source string
}{
	{
		fsType: fsType_FILE,
		name:   "plainFileA",
		data:   []byte("This is a plain file called A\n"),
	},
	{
		fsType: fsType_SYMLINK,
		name:   "symlinkB",
		source: "plainFileA",
	},
	{
		fsType: fsType_DIRECTORY,
		name:   "directoryC",
	},
	{
		fsType: fsType_SYMLINK,
		name:   "SymlinkD",
		source: "directoryC",
	},
	{
		fsType: fsType_FILE,
		name:   "executableFileE",
		data:   []byte("#!/bin/sh\necho \"I'm a executable file\"\n"),
		perm:   (*fs.FileMode)(pointy.Uint32(0755)),
	},
	{
		fsType: fsType_DIRECTORY,
		name:   "directoryC/directoryF",
	},
	{
		fsType: fsType_SYMLINK,
		name:   "directoryC/directoryF/symlinkG",
		source: "../../executableFileE",
	},
	{
		fsType: fsType_FILE,
		name:   "directoryC/directoryF/binaryFileH",
		data:   []byte{1, 2, 3, 4, 5, 6, 7},
	},
	{
		fsType: fsType_SYMLINK,
		name:   "directoryC/symlinkI",
		source: "./directoryF",
	},
}

func populateDir(t *testing.T, dir string) error {
	t.Helper()
	for _, entry := range testFS {
		name := filepath.Join(dir, entry.name)
		switch entry.fsType {
		case fsType_DIRECTORY:
			var perm fs.FileMode
			if entry.perm != nil {
				perm = *entry.perm
			} else {
				perm = 0755
			}
			if err := os.Mkdir(name, perm); err != nil {
				return err
			}
		case fsType_FILE:
			var perm fs.FileMode
			if entry.perm != nil {
				perm = *entry.perm
			} else {
				perm = 0644
			}
			if err := os.WriteFile(name, entry.data, perm); err != nil {
				return err
			}
		case fsType_SYMLINK:
			var source string
			if strings.HasPrefix(entry.source, ".") {
				source = entry.source
			} else {
				source = filepath.Join(dir, entry.source)
			}
			if err := os.Symlink(source, name); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDirTestFS(t *testing.T, dir string) error {
	t.Helper()
	for _, entry := range testFS {
		name := filepath.Join(dir, entry.name)
		switch entry.fsType {
		case fsType_DIRECTORY:
			if info, err := os.Stat(name); err != nil {
				t.Error(err)
			} else if !info.IsDir() {
				t.Errorf("%q is not a directory", name)
			} else if entry.perm != nil && info.Mode() != *entry.perm {
				t.Errorf("%q perm have %q want %q", name, info.Mode(), entry.perm)
			}
		case fsType_FILE:
			if info, err := os.Stat(name); err != nil {
				t.Error(err)
			} else if !info.Mode().IsRegular() {
				t.Errorf("%q is not a regular file", name)
			} else if entry.perm != nil && info.Mode() != *entry.perm {
				t.Errorf("%q perm have %q want %q", name, info.Mode(), entry.perm)
			} else if int64(len(entry.data)) != info.Size() {
				t.Errorf("%q size %d, want %d", name, info.Size(), int64(len(entry.data)))
			}
		case fsType_SYMLINK:
			if info, err := os.Lstat(name); err != nil {
				t.Error(err)
			} else if info.Mode()&os.ModeSymlink == 0 {
				t.Errorf("%q mode %q is not a symlink", name, info.Mode())
			}
		}
	}
	return nil
}

func TestCopyAll(t *testing.T) {
	source := t.TempDir()
	destination := t.TempDir()

	t.Run("populate source for testing", func(t *testing.T) {
		if err := populateDir(t, source); err != nil {
			t.Error(err)
		}
	})

	t.Run("recursive copy from source to destination", func(t *testing.T) {
		for _, entry := range testFS {
			name := filepath.Join(source, entry.name)
			if err := copyAll(name, destination); err != nil {
				t.Error(err)
			}
		}
	})

	t.Run("validate destination", func(t *testing.T) {
		if err := validateDirTestFS(t, destination); err != nil {
			t.Error(err)
		}
	})
}
