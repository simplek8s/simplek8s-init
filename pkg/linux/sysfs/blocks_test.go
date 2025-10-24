// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package sysfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/mount"
)

func TestGetBlockDevices_Success(t *testing.T) {
	t.Cleanup(func() {
		isPathExists = common.IsPathExists
		osReadDir = os.ReadDir
		doMount = mount.Mount
		doUnmount = mount.Unmount
	})

	tmpDir := t.TempDir()
	blockDir := filepath.Join(tmpDir, "class", "block")

	if err := os.MkdirAll(blockDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sda", "vdb1"} {
		if err := os.WriteFile(filepath.Join(blockDir, name), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}

	isPathExists = func(name string) bool {
		if name == "/sys/class/block" {
			return common.IsPathExists(blockDir)
		}
		return common.IsPathExists(name)
	}

	osReadDir = func(name string) ([]os.DirEntry, error) {
		if name == "/sys/class/block" {
			return os.ReadDir(blockDir)
		}
		return os.ReadDir(name)
	}

	devices, err := GetBlockDevices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"/dev/sda", "/dev/vdb1"}
	if len(devices) != len(expected) {
		t.Fatalf("expected %d devices, got %d", len(expected), len(devices))
	}
	for i, d := range expected {
		if devices[i] != d {
			t.Errorf("expected %q, got %q", d, devices[i])
		}
	}
}

func TestGetBlockDevices_SysNotMounted(t *testing.T) {
	t.Cleanup(func() {
		isPathExists = common.IsPathExists
		osReadDir = os.ReadDir
		doMount = mount.Mount
		doUnmount = mount.Unmount
	})

	isPathExists = func(name string) bool {
		if name == "/sys/class/block" {
			return false
		}
		return common.IsPathExists(name)
	}

	osReadDir = func(name string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{"sda"}}, nil
	}

	calledMount := false
	calledUnmount := false

	doMount = func(_ mount.MountPoint) error {
		calledMount = true
		return nil
	}
	doUnmount = func(_ string, _ mount.UnmountFlag) error {
		calledUnmount = true
		return nil
	}

	devices, err := GetBlockDevices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(devices) != 1 || devices[0] != "/dev/sda" {
		t.Errorf("expected [/dev/sda], got %v", devices)
	}

	if !calledMount {
		t.Error("expected doMount to be called")
	}
	if !calledUnmount {
		t.Error("expected doUnmount to be called")
	}
}

func TestGetBlockDevices_ReadDirError(t *testing.T) {
	t.Cleanup(func() {
		isPathExists = common.IsPathExists
		osReadDir = os.ReadDir
	})

	isPathExists = func(name string) bool {
		return common.IsPathExists(".")
	}
	osReadDir = func(name string) ([]os.DirEntry, error) {
		return nil, errors.New("fake readdir error")
	}

	_, err := GetBlockDevices()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

type fakeDirEntry struct{ name string }

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return false }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }
