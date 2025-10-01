package linux

import (
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

func MountPseudoFS(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	mountpoints := []struct {
		source string
		target string
		fstype string
		flags  uintptr
		chmod  os.FileMode
	}{
		{
			source: "proc",
			target: "/proc",
			fstype: "proc",
			flags:  unix.MS_NOSUID | unix.MS_NOEXEC | unix.MS_NODEV,
			chmod:  0555,
		},
		{
			source: "sysfs",
			target: "/sys",
			fstype: "sysfs",
			flags:  unix.MS_NOSUID | unix.MS_NOEXEC | unix.MS_NODEV,
			chmod:  0555,
		},
		{
			source: "devtmpfs",
			target: "/dev",
			fstype: "devtmpfs",
			flags:  unix.MS_NOSUID,
			chmod:  0755,
		},
	}

	for _, mp := range mountpoints {
		target := filepath.Join(where, mp.target)
		if err := os.MkdirAll(target, mp.chmod); err != nil {
			return fmt.Errorf("mkdir %s failed: %w", target, err)
		}
		if err := os.Chmod(target, mp.chmod); err != nil {
			return fmt.Errorf("chmod %s failed: %w", target, err)
		}
		if err := unix.Mount(mp.source, target, mp.fstype, mp.flags, ""); err != nil {
			return fmt.Errorf("mount %s at %s failed: %w", mp.source, target, err)
		}
	}

	return nil
}

func UnmountPseudoFS(where string) error {
	for _, mp := range []string{"/proc", "/sys", "/dev"} {
		dst := filepath.Join(where, mp)
		if err := unix.Unmount(dst, unix.MNT_DETACH); err != nil {
			return fmt.Errorf("unmount %s failed: %w", dst, err)
		}
		if err := unix.Rmdir(dst); err != nil {
			return fmt.Errorf("rmdir %s failed: %w", dst, err)
		}
	}
	return nil
}

func CreateDeprecatedSymlinks(where string) error {
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
