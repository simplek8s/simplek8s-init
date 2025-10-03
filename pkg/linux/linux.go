package linux

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jlsalvador/simplek8s/pkg/common"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func IsStageInitrd() bool {
	log.Debug("start")
	defer log.Debug("end")

	return common.CheckFileExists("/etc/initrd-release")
}

type MountPoint struct {
	Target string
	Chmod  os.FileMode
	Source string
	Fstype string
	Flags  uintptr
	Data   string
}

func Mount(mount MountPoint) error {
	log.Debug("start")
	defer log.Debug("end")

	// Ensure that destination exists.
	if _, err := os.Stat(mount.Target); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(mount.Target, mount.Chmod); err != nil {
			return fmt.Errorf("mkdir %s failed: %w", mount.Target, err)
		}
	} else if err != nil {
		return fmt.Errorf("can not stat %s: %w", mount.Target, err)
	}

	// Mount on destination.
	if err := unix.Mount(mount.Source, mount.Target, mount.Fstype, mount.Flags, ""); err != nil {
		return fmt.Errorf("mount %s at %s failed: %w", mount.Source, mount.Target, err)
	}

	return nil
}

var Mountpoints = struct {
	Dev  MountPoint
	Sys  MountPoint
	Proc MountPoint
}{
	Dev:  MountPoint{"/dev", 0755, "devtmpfs", "devtmpfs", unix.MS_NOSUID | unix.MS_STRICTATIME, ""},
	Sys:  MountPoint{"/sys", 0555, "sysfs", "sysfs", 0, ""},
	Proc: MountPoint{"/proc", 0555, "proc", "proc", 0, ""},
}

func MountPseudoFS(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	for _, m := range []MountPoint{
		Mountpoints.Dev,
		Mountpoints.Sys,
		Mountpoints.Proc,
	} {
		m.Target = filepath.Join(where, m.Target)
		if err := Mount(m); err != nil {
			return err
		}
	}

	return nil
}

func UnmountPseudoFS(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	for _, target := range []string{Mountpoints.Dev.Target, Mountpoints.Sys.Target, Mountpoints.Proc.Target} {
		target = filepath.Join(where, target)
		if err := unix.Unmount(target, unix.MNT_DETACH); err != nil {
			return fmt.Errorf("unmount %s failed: %w", target, err)
		}
	}
	return nil
}

func CreateDeprecatedSymlinks(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	for _, sl := range []struct {
		old string
		new string
	}{
		{"usr/bin", filepath.Join(where, "bin")},
		{"usr/lib", filepath.Join(where, "lib")},
		{"lib", filepath.Join(where, "lib64")},
		{"usr/sbin", filepath.Join(where, "sbin")},
	} {
		if err := os.Symlink(sl.old, sl.new); err != nil {
			return fmt.Errorf("can not create symlink %s as %s", sl.old, sl.new)
		}
	}
	return nil
}
