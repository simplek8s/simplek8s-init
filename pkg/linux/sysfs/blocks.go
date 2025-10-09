// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package sysfs

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/jlsalvador/simplek8s/pkg/linux/mount"
)

// Injected functions for easier testing.
var (
	osStat    = os.Stat
	osReadDir = os.ReadDir
	doMount   = mount.Mount
	doUnmount = mount.Unmount
)

// GetBlockDevices returns a slice containing the paths of all block devices
// currently present on the host system (e.g., /dev/sda, /dev/vdb1, ...).
//
// It reads `/sys/class/block` which lists the block device names, then
// prefixes each name with `/dev/` to produce full device paths.
//
// In order to reads `/sys/class/block`, the pseudofs `sysfs` must` will be
// mounted and unmounted automatically if it is necessary.
func GetBlockDevices() ([]string, error) {
	devices := []string{}

	// Only mount & unmount "/sys" is there is not mounted already.
	if _, err := osStat("/sys/class/block"); errors.Is(err, os.ErrNotExist) {
		doMount(mount.Mountpoints.Sys)
		defer doUnmount(mount.Mountpoints.Sys.Target, 0)
	}

	entries, err := osReadDir("/sys/class/block")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		devPath := filepath.Join("/dev/", entry.Name())
		devices = append(devices, devPath)
	}

	return devices, nil
}
