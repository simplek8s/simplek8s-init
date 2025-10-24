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
	MountFlagReadOnly    = unix.MS_RDONLY // Cannot write anything on that filesystem.
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

// Mountpoints defines common linux mountpoints.
var Mountpoints = struct {
	Sysroot MountPoint
	Dev     MountPoint
	Sys     MountPoint
	Proc    MountPoint
	Run     MountPoint
	Usr     MountPoint
}{
	Sysroot: MountPoint{"/", 0o755, "tmpfs", "tmpfs", MountFlagNoSUID | MountFlagNoDev, ""},
	Dev:     MountPoint{"/dev", 0o755, "devtmpfs", "devtmpfs", MountFlagNoSUID | MountFlagStrictATime, ""},
	Sys:     MountPoint{"/sys", 0o555, "sysfs", "sysfs", 0, ""},
	Proc:    MountPoint{"/proc", 0o555, "proc", "proc", 0, ""},
	Run:     MountPoint{"/run", 0o755, "tmpfs", "tmpfs", MountFlagNoSUID | MountFlagNoDev, ""},
	Usr:     MountPoint{"/usr", 0o755, "tmpfs", "tmpfs", MountFlagNoSUID | MountFlagNoDev, ""},
}

// Mount creates destination directory and mounts the mountpoints there.
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

// Unmount unmounts target and removes the target (empty) directory.
func Unmount(target string, flags UnmountFlag) error {
	if err := sysUnmount(target, int(flags)); err != nil {
		return fmt.Errorf("cannot unmount %s: %w", target, err)
	}

	if err := os.Remove(target); err != nil {
		return fmt.Errorf("cannot remove (must be empty) directory %s: %w", target, err)
	}

	return nil
}
