// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package mount

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// Wrappers that can be mocked in tests.
var (
	sysMount   = unix.Mount
	sysUnmount = unix.Unmount
)

// Can be concat by using |.
//
//	flags := MountFlagReadOnly | MountFlagNoSUID
type MountFlag uintptr

const (
	MountFlagReadOnly    = unix.MS_RDONLY // Can not write anything on that filesystem.
	MountFlagNoSUID      = unix.MS_NOSUID // Disables execution with the same permissions as the owner.
	MountFlagNoDev       = unix.MS_NODEV  // Device nodes on that filesystem won't work.
	MountFlagBind        = unix.MS_BIND
	MountFlagRemount     = unix.MS_REMOUNT
	MountFlagStrictATime = unix.MS_STRICTATIME // Always updates access time.
	MountFlagRelATime    = unix.MS_RELATIME    // Sometimes updates access time.
)

// Can be concat by using |.
//
//	flags := UnmountFlagDetach | UnmountFlagForce
type UnmountFlag int

const (
	UnmountFlagDetach = unix.MNT_DETACH // Async detach.
	UnmountFlagForce  = unix.MNT_FORCE  // Force unmount even with files in use.
)

type MountPoint struct {
	Target string
	Chmod  os.FileMode
	Source string
	Fstype string    // Empty for auto.
	Flags  MountFlag // Can be concat by |.
	Data   string
}

// Common Linux mountpoints.
var Mountpoints = struct {
	Dev  MountPoint
	Sys  MountPoint
	Proc MountPoint
	Run  MountPoint
}{
	Dev:  MountPoint{"/dev", 0755, "devtmpfs", "devtmpfs", MountFlagNoSUID | MountFlagStrictATime, ""},
	Sys:  MountPoint{"/sys", 0555, "sysfs", "sysfs", 0, ""},
	Proc: MountPoint{"/proc", 0555, "proc", "proc", 0, ""},
	Run:  MountPoint{"/run", 0755, "tmpfs", "tmpfs", MountFlagNoSUID | MountFlagNoDev, ""},
}

// Creates destination directory and mounts the mountpoints there.
func Mount(mount MountPoint) error {
	// Ensure that destination exists.
	if err := os.MkdirAll(mount.Target, mount.Chmod); err != nil {
		return fmt.Errorf("mkdir %s failed: %w", mount.Target, err)
	}

	// Mount on destination.
	if err := sysMount(mount.Source, mount.Target, mount.Fstype, uintptr(mount.Flags), ""); err != nil {
		return fmt.Errorf("mount %s at %s failed: %w", mount.Source, mount.Target, err)
	}

	return nil
}

// Unmounts target and removes the target (empty) directory.
func Unmount(target string, flags UnmountFlag) error {
	if err := sysUnmount(target, int(flags)); err != nil {
		return fmt.Errorf("can not unmount %s: %w", target, err)
	}

	if err := os.Remove(target); err != nil {
		return fmt.Errorf("can not remove (must be empty) directory %s: %w", target, err)
	}

	return nil
}
