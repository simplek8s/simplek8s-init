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

	mounts := []struct {
		target string
		chmod  os.FileMode
		source string
		fstype string
		flags  uintptr
	}{
		{"/proc", 0555, "proc", "proc", 0},
		{"/sys", 0555, "sysfs", "sysfs", 0},
		{"/dev", 0755, "devtmpfs", "devtmpfs",
			unix.MS_NOSUID | unix.MS_STRICTATIME},
	}

	for _, m := range mounts {
		t := filepath.Join(where, m.target)

		// Create destination.
		if _, err := os.Stat(t); err != nil {
			return fmt.Errorf("can not stat %s: %w", t, err)
		} else if err == os.ErrNotExist {
			if err := os.Mkdir(t, m.chmod); err != nil {
				return fmt.Errorf("mkdir %s failed: %w", t, err)
			}
		}

		// Mount on destination.
		if err := unix.Mount(m.source, t, m.fstype, m.flags, ""); err != nil {
			return fmt.Errorf("mount %s at %s failed: %w", m.source, t, err)
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

// GetBlockDevices returns a slice containing the paths of all block devices
// currently present on the host system (e.g., /dev/sda, /dev/vdb1, ...).
//
// It reads `/sys/class/block` which lists the block device names, then
// prefixes each name with `/dev/` to produce full device paths.
func GetBlockDevices() ([]string, error) {
	log.Debug("start")
	defer log.Debug("end")

	devices := []string{}

	entries, err := os.ReadDir("/sys/class/block")
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, entry := range entries {
		devPath := filepath.Join("/dev/", entry.Name())
		devices = append(devices, devPath)
	}

	return devices, nil
}
