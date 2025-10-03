package sysfs

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/jlsalvador/simplek8s/pkg/linux"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// GetBlockDevices returns a slice containing the paths of all block devices
// currently present on the host system (e.g., /dev/sda, /dev/vdb1, ...).
//
// It reads `/sys/class/block` which lists the block device names, then
// prefixes each name with `/dev/` to produce full device paths.
func GetBlockDevices() ([]string, error) {
	log.Debug("start")
	defer log.Debug("end")

	devices := []string{}

	// Only mount & unmount "/sys" is there is not mounted already.
	if _, err := os.Stat("/sys/class/block"); errors.Is(err, os.ErrNotExist) {
		linux.Mount(linux.Mountpoints.Sys)
		defer unix.Unmount(linux.Mountpoints.Sys.Target, 0)
	} else {
		log.Warn(err)
	}

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
