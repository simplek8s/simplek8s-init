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

package sysfs

import (
	"os"
	"path/filepath"

	"github.com/simplek8s/simplek8s-init/pkg/common"
	"github.com/simplek8s/simplek8s-init/pkg/linux/mount"
)

// Injected functions for easier testing.
var (
	isPathExists = common.IsPathExists
	osReadDir    = os.ReadDir
	doMount      = mount.Mount
	doUnmount    = mount.Unmount
)

// GetBlockDevices returns a slice containing the paths of all block devices
// currently present on the host system (e.g., /dev/sda, /dev/vdb1, ...).
//
// It reads `/sys/class/block` which lists the block device names, then
// prefixes each name with `/dev/` to produce full device paths.
//
// In order to reads `/sys/class/block`, the pseudofs `sysfs` will be mounted
// and unmounted automatically if it is necessary.
func GetBlockDevices() ([]string, error) {
	// Only mount & unmount "/sys" is there is not mounted already.
	if !isPathExists("/sys/class/block") {
		doMount(mount.Mountpoints.Sys)
		defer doUnmount(mount.Mountpoints.Sys.Target, 0)
	}

	entries, err := osReadDir("/sys/class/block")
	if err != nil {
		return nil, err
	}

	devices := []string{}
	for _, entry := range entries {
		devPath := filepath.Join("/dev/", entry.Name())
		devices = append(devices, devPath)
	}

	return devices, nil
}
