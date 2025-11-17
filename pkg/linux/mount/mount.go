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
	"os/exec"
	"strings"

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
	MountFlagReadOnly    = unix.MS_RDONLY                   // Cannot write anything on that filesystem.
	MountFlagReadWrite   = 0                                // Read-write (default, no flag).
	MountFlagSUID        = 0                                // SUID enabled (default, no flag).
	MountFlagNoSUID      = unix.MS_NOSUID                   // Disables execution with the same permissions as the owner.
	MountFlagDev         = 0                                // Interpret device files (default, not flag).
	MountFlagNoDev       = unix.MS_NODEV                    // Device nodes on that filesystem won't work.
	MountFlagExec        = 0                                // Allow execution of binaries (default, no flag).
	MountFlagNoExec      = unix.MS_NOEXEC                   // Disallow execution.
	MountFlagSync        = unix.MS_SYNCHRONOUS              // Synchronous writes.
	MountFlagAsync       = 0                                // Asynchronous directory updates (default, no flag).
	MountFlagDirSync     = unix.MS_DIRSYNC                  // Synchronous directory updates.
	MountFlagRemount     = unix.MS_REMOUNT                  // Reconfigure flags for an already mounted mountpoint.
	MountFlagMandLock    = unix.MS_MANDLOCK                 // Allow mandatory locking.
	MountFlagNoMandLock  = 0                                // Disallow mandatory locking (default, no flag).
	MountFlagATime       = 0                                // Always updates access time (default, no flag).
	MountFlagNoATime     = unix.MS_NOATIME                  // Do not update access time.
	MountFlagRelATime    = unix.MS_RELATIME                 // Sometimes updates access time.
	MountFlagStrictATime = unix.MS_STRICTATIME              // Always updates access time.
	MountFlagNoDirATime  = unix.MS_NODIRATIME               // Do not update directory access time.
	MountFlagBind        = unix.MS_BIND                     // Mount a folder somewhere else.
	MountFlagRBind       = unix.MS_BIND | unix.MS_REC       // Recursive bind.
	MountFlagRec         = unix.MS_REC                      // Recursive mount (used with bind).
	MountFlagPrivate     = unix.MS_PRIVATE                  // Mount propagation private.
	MountFlagRPrivate    = unix.MS_PRIVATE | unix.MS_REC    // Recursive mount propagation private.
	MountFlagShared      = unix.MS_SHARED                   // Mount propagation shared.
	MountFlagRShared     = unix.MS_SHARED | unix.MS_REC     // Recursive mount propagation shared.
	MountFlagSlave       = unix.MS_SLAVE                    // Mount propagation slave.
	MountFlagRSlave      = unix.MS_SLAVE | unix.MS_REC      // Recursive mount propagation slave.
	MountFlagUnbindable  = unix.MS_UNBINDABLE               // Mount propagation unbindable.
	MountFlagRUnbindable = unix.MS_UNBINDABLE | unix.MS_REC // Recursive mount propagation unbindable.
	MountFlagLazyTime    = unix.MS_LAZYTIME                 // Defer updates to atime, mtime, ctime.
	MountFlagMove        = unix.MS_MOVE                     // Move a subtree to some other place.
	MountFlagSubmount    = unix.MS_SUBMOUNT                 // Allow a mount to be moved.
	MountFlagPosixAcl    = unix.MS_POSIXACL                 // Enable POSIX Access Control Lists.
	MountFlagNoAcl       = 0                                // Disable POSIX Access Control Lists (default, no flag).
)

var MountFlags = map[string]MountFlag{
	"ro":           MountFlagReadOnly,
	"rw":           MountFlagReadWrite,
	"suid":         MountFlagSUID,
	"nosuid":       MountFlagNoSUID,
	"dev":          MountFlagDev,
	"nodev":        MountFlagNoDev,
	"exec":         MountFlagExec,
	"noexec":       MountFlagNoExec,
	"sync":         MountFlagSync,
	"async":        MountFlagAsync,
	"dirsync":      MountFlagDirSync,
	"remount":      MountFlagRemount,
	"mand":         MountFlagMandLock,
	"nomand":       MountFlagNoMandLock,
	"atime":        MountFlagATime,
	"noatime":      MountFlagNoATime,
	"relatime":     MountFlagRelATime,
	"strictatime":  MountFlagStrictATime,
	"nodiratime":   MountFlagNoDirATime,
	"bind":         MountFlagBind,
	"rbind":        MountFlagRBind,
	"rec":          MountFlagRec,
	"private":      MountFlagPrivate,
	"shared":       MountFlagShared,
	"rshared":      MountFlagRShared,
	"slave":        MountFlagSlave,
	"rslave":       MountFlagRSlave,
	"unbindable":   MountFlagUnbindable,
	"runbindable":  MountFlagRUnbindable,
	"r-unbindable": MountFlagRUnbindable,
	"lazytime":     MountFlagLazyTime,
	"move":         MountFlagMove,
	"submount":     MountFlagSubmount,
	"posixacl":     MountFlagPosixAcl,
	"noacl":        MountFlagNoAcl,
}

// Can be concat by using |.
//
//	flags := UnmountFlagDetach | UnmountFlagForce
type UnmountFlag int

const (
	UnmountFlagDetach = unix.MNT_DETACH // Async detach.
	UnmountFlagExpire = unix.MNT_EXPIRE // Async expire.
	UnmountFlagForce  = unix.MNT_FORCE  // Force unmount even with files in use.
)

type MountPoint struct {
	Target string      // Target directory where to mount.
	Chmod  os.FileMode // When target directory doesnot exists, create it with this permissions.
	Source string      // Linux source directory, device or pseudo filesystem.
	Fstype string      // Empty for auto.
	Flags  MountFlag   // Can be concat by |.
	Data   string      // Normally empty string.
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
	Sysroot: MountPoint{"/", 0o755, "none", "tmpfs", MountFlagNoSUID | MountFlagNoDev, ""},
	Dev:     MountPoint{"/dev", 0o755, "none", "devtmpfs", MountFlagNoSUID | MountFlagStrictATime, ""},
	Sys:     MountPoint{"/sys", 0o555, "none", "sysfs", 0, ""},
	Proc:    MountPoint{"/proc", 0o555, "none", "proc", 0, ""},
	Run:     MountPoint{"/run", 0o755, "none", "tmpfs", MountFlagNoSUID | MountFlagNoDev, ""},
	Usr:     MountPoint{"/usr", 0o755, "none", "tmpfs", MountFlagNoSUID | MountFlagNoDev, ""},
}

func blkidType(dev string) (string, error) {
	cmd := "/usr/sbin/blkid"
	args := []string{"-o", "value", "-s", "TYPE", dev}
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return "", fmt.Errorf("error executing: %s %s: %w", cmd, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Mount creates destination directory and mounts the mountpoints there.
func Mount(mount MountPoint) error {
	// Ensure that destination exists.
	if err := os.MkdirAll(mount.Target, mount.Chmod); err != nil {
		return fmt.Errorf("mkdir %s failed: %w", mount.Target, err)
	}

	// unix.Mount requires FSType, it can not be empty or auto.
	if mount.Source != "" && mount.Source != "none" && (mount.Fstype == "" || mount.Fstype == "auto") {
		fstype, err := blkidType(mount.Source)
		if err != nil {
			return fmt.Errorf("cannot fetch filesystem type from %s: %w", mount.Source, err)
		}
		mount.Fstype = fstype
	}

	// Mount on destination.
	if err := sysMount(mount.Source, mount.Target, mount.Fstype, uintptr(mount.Flags), mount.Data); err != nil {
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
