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

package mount

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/simplek8s/simplek8s-init/pkg/linux/blkid"

	"golang.org/x/sys/unix"
)

// Mock functions for testing
var (
	mockSysMount              func(source string, target string, fstype string, flags uintptr, data string) error
	mockSysUnmount            func(target string, flags int) error
	mockBlkidDetectFilesystem func(device string) (string, error)

	origSysMount              func(string, string, string, uintptr, string) error
	origSysUnmount            func(string, int) error
	origBlkidDetectFileSystem func(string) (string, error)
)

func TestMain(m *testing.M) {
	// Save original functions
	origSysMount = sysMount
	origSysUnmount = sysUnmount
	origBlkidDetectFileSystem = blkidDetectFileSystem

	// Run tests
	code := m.Run()

	// Restore original functions
	sysMount = origSysMount
	sysUnmount = origSysUnmount
	blkidDetectFileSystem = origBlkidDetectFileSystem

	os.Exit(code)
}

func setupMocks() {
	sysMount = func(source string, target string, fstype string, flags uintptr, data string) error {
		if mockSysMount != nil {
			return mockSysMount(source, target, fstype, flags, data)
		}
		return nil
	}
	sysUnmount = func(target string, flags int) error {
		if mockSysUnmount != nil {
			return mockSysUnmount(target, flags)
		}
		return nil
	}
	blkidDetectFileSystem = func(device string) (string, error) {
		if mockBlkidDetectFilesystem != nil {
			return mockBlkidDetectFilesystem(device)
		}
		return "", nil
	}
}

func resetMocks() {
	mockSysMount = nil
	mockSysUnmount = nil
	mockBlkidDetectFilesystem = nil
	sysMount = origSysMount
	sysUnmount = origSysUnmount
	blkidDetectFileSystem = origBlkidDetectFileSystem
}

func TestMount_Success(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	mockSysMount = func(source, target, fstype string, flags uintptr, data string) error {
		return nil
	}

	mp := MountPoint{
		Target: target,
		Chmod:  0755,
		Source: "tmpfs",
		Fstype: "tmpfs",
		Flags:  MountFlagNoSUID,
		Data:   "",
	}

	err := Mount(mp)
	if err != nil {
		t.Errorf("Mount() error = %v, want nil", err)
	}

	// Verify directory was created
	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error("target directory was not created")
	}
}

func TestMount_WithBlockIDType_Success(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	mockBlkidDetectFilesystem = func(device string) (string, error) {
		return "xfs", nil
	}

	var capturedFstype string
	mockSysMount = func(source, target, fstype string, flags uintptr, data string) error {
		capturedFstype = fstype
		return nil
	}

	mp := MountPoint{
		Target: target,
		Chmod:  0755,
		Source: "/dev/sda1",
		Fstype: "auto", // This will trigger BlockIDType
		Flags:  0,
		Data:   "",
	}

	err := Mount(mp)
	if err != nil {
		t.Errorf("Mount() error = %v, want nil", err)
	}

	// Verify that fstype was correctly detected
	if capturedFstype != "xfs" {
		t.Errorf("Mount() fstype = %q, want %q", capturedFstype, "xfs")
	}
}

func TestMount_WithBlockIDType_EmptyFstype_Success(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	mockBlkidDetectFilesystem = func(device string) (string, error) {
		return "btrfs", nil
	}

	var capturedFstype string
	mockSysMount = func(source, target, fstype string, flags uintptr, data string) error {
		capturedFstype = fstype
		return nil
	}

	mp := MountPoint{
		Target: target,
		Chmod:  0755,
		Source: "/dev/sdb1",
		Fstype: "", // Empty fstype will trigger BlockIDType
		Flags:  0,
		Data:   "",
	}

	err := Mount(mp)
	if err != nil {
		t.Errorf("Mount() error = %v, want nil", err)
	}

	// Verify that fstype was correctly detected
	if capturedFstype != "btrfs" {
		t.Errorf("Mount() fstype = %q, want %q", capturedFstype, "btrfs")
	}
}

func TestMount_MkdirFails(t *testing.T) {
	setupMocks()
	defer resetMocks()

	// Use an invalid path that will fail mkdir
	mp := MountPoint{
		Target: "/proc/invalid/path/that/cannot/be/created",
		Chmod:  0755,
		Source: "tmpfs",
		Fstype: "tmpfs",
		Flags:  0,
		Data:   "",
	}

	err := Mount(mp)
	if err == nil {
		t.Error("Mount() expected error for invalid path, got nil")
	}
}

func TestMount_BlockIDTypeFails(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	mockBlkidDetectFilesystem = func(device string) (string, error) {
		return "", blkid.ErrUnknownFilesystem
	}

	mockSysMount = func(source, target, fstype string, flags uintptr, data string) error {
		return nil
	}

	mp := MountPoint{
		Target: target,
		Chmod:  0755,
		Source: "/dev/nonexistent_device",
		Fstype: "auto",
		Flags:  0,
		Data:   "",
	}

	err := Mount(mp)
	if err == nil {
		t.Error("Mount() expected error when BlockIDType fails, got nil")
	}

	// Verify error message
	if err != nil && !contains(err.Error(), "cannot fetch filesystem type") {
		t.Errorf("Mount() error should mention 'cannot fetch filesystem type': %v", err)
	}
}

func TestMount_NoneSource(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	mockSysMount = func(source, target, fstype string, flags uintptr, data string) error {
		return nil
	}

	mp := MountPoint{
		Target: target,
		Chmod:  0755,
		Source: "none",
		Fstype: "",
		Flags:  0,
		Data:   "",
	}

	err := Mount(mp)
	if err != nil {
		t.Errorf("Mount() error = %v, want nil for 'none' source", err)
	}
}

func TestMount_EmptySource(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	mockSysMount = func(source, target, fstype string, flags uintptr, data string) error {
		return nil
	}

	mp := MountPoint{
		Target: target,
		Chmod:  0755,
		Source: "",
		Fstype: "tmpfs",
		Flags:  0,
		Data:   "",
	}

	err := Mount(mp)
	if err != nil {
		t.Errorf("Mount() error = %v, want nil for empty source", err)
	}
}

func TestMount_SysMountFails(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	mockSysMount = func(source, target, fstype string, flags uintptr, data string) error {
		return errors.New("mount failed")
	}

	mp := MountPoint{
		Target: target,
		Chmod:  0755,
		Source: "tmpfs",
		Fstype: "tmpfs",
		Flags:  0,
		Data:   "",
	}

	err := Mount(mp)
	if err == nil {
		t.Error("Mount() expected error when sysMount fails, got nil")
	}
}

func TestUnmount_Success(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	// Create the directory first
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target directory: %v", err)
	}

	mockSysUnmount = func(target string, flags int) error {
		return nil
	}

	err := Unmount(target, 0)
	if err != nil {
		t.Errorf("Unmount() error = %v, want nil", err)
	}

	// Verify directory was removed
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("target directory was not removed")
	}
}

func TestUnmount_SysUnmountFails(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target directory: %v", err)
	}

	mockSysUnmount = func(target string, flags int) error {
		return errors.New("unmount failed")
	}

	err := Unmount(target, 0)
	if err == nil {
		t.Error("Unmount() expected error when sysUnmount fails, got nil")
	}
}

func TestUnmount_RemoveFails(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	// Create the directory with a file inside to make removal fail
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target directory: %v", err)
	}
	fileInside := filepath.Join(target, "file.txt")
	if err := os.WriteFile(fileInside, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file inside target: %v", err)
	}

	mockSysUnmount = func(target string, flags int) error {
		return nil
	}

	// Unmount succeeded, so a non-empty (pre-existing) directory must not
	// turn into an error; it is simply kept in place.
	err := Unmount(target, 0)
	if err != nil {
		t.Errorf("Unmount() expected nil when target is not empty, got %v", err)
	}

	// Verify directory was kept
	if _, err := os.Stat(target); err != nil {
		t.Error("non-empty target directory should have been kept")
	}
}

func TestUnmount_WithFlags(t *testing.T) {
	setupMocks()
	defer resetMocks()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "mnt")

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target directory: %v", err)
	}

	var receivedFlags int
	mockSysUnmount = func(target string, flags int) error {
		receivedFlags = flags
		return nil
	}

	expectedFlags := int(UnmountFlagDetach | UnmountFlagForce)
	err := Unmount(target, UnmountFlag(expectedFlags))
	if err != nil {
		t.Errorf("Unmount() error = %v, want nil", err)
	}

	if receivedFlags != expectedFlags {
		t.Errorf("Unmount() flags = %v, want %v", receivedFlags, expectedFlags)
	}
}

func TestMountFlags_Mapping(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"ro", "ro"},
		{"rw", "rw"},
		{"suid", "suid"},
		{"nosuid", "nosuid"},
		{"dev", "dev"},
		{"nodev", "nodev"},
		{"exec", "exec"},
		{"noexec", "noexec"},
		{"sync", "sync"},
		{"async", "async"},
		{"dirsync", "dirsync"},
		{"remount", "remount"},
		{"mand", "mand"},
		{"nomand", "nomand"},
		{"atime", "atime"},
		{"noatime", "noatime"},
		{"relatime", "relatime"},
		{"strictatime", "strictatime"},
		{"nodiratime", "nodiratime"},
		{"bind", "bind"},
		{"rbind", "rbind"},
		{"rec", "rec"},
		{"private", "private"},
		{"shared", "shared"},
		{"rshared", "rshared"},
		{"slave", "slave"},
		{"rslave", "rslave"},
		{"unbindable", "unbindable"},
		{"runbindable", "runbindable"},
		{"r-unbindable", "r-unbindable"},
		{"lazytime", "lazytime"},
		{"move", "move"},
		{"submount", "submount"},
		{"posixacl", "posixacl"},
		{"noacl", "noacl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := MountFlags[tt.key]; !ok {
				t.Errorf("MountFlags[%q] not found", tt.key)
			}
		})
	}
}

func TestMountpoints_Struct(t *testing.T) {
	// Test that all mountpoints are properly initialized
	mountpoints := []struct {
		name string
		mp   MountPoint
	}{
		{"Sysroot", Mountpoints.Sysroot},
		{"Dev", Mountpoints.Dev},
		{"Sys", Mountpoints.Sys},
		{"Proc", Mountpoints.Proc},
		{"Run", Mountpoints.Run},
		{"Usr", Mountpoints.Usr},
	}

	for _, tt := range mountpoints {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mp.Target == "" {
				t.Errorf("%s.Target is empty", tt.name)
			}
			if tt.mp.Chmod == 0 {
				t.Errorf("%s.Chmod is zero", tt.name)
			}
			if tt.mp.Source == "" {
				t.Errorf("%s.Source is empty", tt.name)
			}
			if tt.mp.Fstype == "" {
				t.Errorf("%s.Fstype is empty", tt.name)
			}
		})
	}
}

func TestMountFlag_Combinations(t *testing.T) {
	// Test flag combinations
	combined := MountFlagReadOnly | MountFlagNoSUID
	if combined == 0 {
		t.Error("Combined flags should not be zero")
	}

	combined2 := MountFlagBind | MountFlagRec
	if combined2 != MountFlagRBind {
		t.Error("MountFlagBind | MountFlagRec should equal MountFlagRBind")
	}
}

func TestUnmountFlag_Constants(t *testing.T) {
	// Test that unmount flags are properly defined
	if UnmountFlagDetach != unix.MNT_DETACH {
		t.Error("UnmountFlagDetach value mismatch")
	}
	if UnmountFlagExpire != unix.MNT_EXPIRE {
		t.Error("UnmountFlagExpire value mismatch")
	}
	if UnmountFlagForce != unix.MNT_FORCE {
		t.Error("UnmountFlagForce value mismatch")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
