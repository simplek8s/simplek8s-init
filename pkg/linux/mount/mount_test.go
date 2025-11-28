// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package mount

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

var (
	sysMountReal   = sysMount
	sysUnmountReal = sysUnmount
)

func restoreSysCalls() {
	sysMount = sysMountReal
	sysUnmount = sysUnmountReal
}

func TestMount_TableDriven(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func(target string) // optional setup (example: chmod path to break it).
		mockMount func(source, target, fstype string, flags uintptr, data string) error
		expectErr bool
	}{
		{
			name: "success",
			mockMount: func(source, target, fstype string, flags uintptr, data string) error {
				return nil
			},
		},
		{
			name: "mkdir fails",
			setup: func(target string) {
				// Create a file where the directory should be.
				file := filepath.Dir(target)
				os.WriteFile(file, []byte("not a dir"), 0644)
			},
			mockMount: func(source, target, fstype string, flags uintptr, data string) error {
				return nil
			},
			expectErr: true,
		},
		{
			name: "mount syscall fails",
			mockMount: func(source, target, fstype string, flags uintptr, data string) error {
				return errors.New("simulated mount failure")
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := filepath.Join(tmpDir, tt.name, "mnt")
			if tt.setup != nil {
				tt.setup(target)
			}

			sysMount = tt.mockMount
			defer restoreSysCalls()

			m := MountPoint{
				Target: target,
				Chmod:  0755,
				Source: "tmpfs",
				Fstype: "tmpfs",
				Flags:  MountFlagNoSUID,
				Data:   "",
			}

			err := Mount(m)
			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUnmount_TableDriven(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		mockUnmount func(target string, flags int) error
		removeFails bool
		expectErr   bool
	}{
		{
			name: "success",
			mockUnmount: func(target string, flags int) error {
				return nil
			},
		},
		{
			name: "unmount syscall fails",
			mockUnmount: func(target string, flags int) error {
				return errors.New("simulated unmount failure")
			},
			expectErr: true,
		},
		{
			name: "remove fails",
			mockUnmount: func(target string, flags int) error {
				return nil
			},
			removeFails: true,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := filepath.Join(tmpDir, tt.name, "mnt")
			if err := os.MkdirAll(target, 0755); err != nil {
				t.Fatal(err)
			}

			if tt.removeFails {
				// Create a file inside so that os.Remove fails.
				file := filepath.Join(target, "dummy")
				if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			sysUnmount = tt.mockUnmount
			defer restoreSysCalls()

			err := Unmount(target, UnmountFlagDetach)
			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
