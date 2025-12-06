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

package simpleinit

import (
	"fmt"
	"os"

	"github.com/simplek8s/simplek8s-init/pkg/common"
	"github.com/simplek8s/simplek8s-init/pkg/linux/mount"
	"github.com/simplek8s/simplek8s-init/pkg/log"

	l "github.com/sirupsen/logrus"
)

// disableKmsgRateLimit deactivate the /dev/kmsg rate limiting rate by setting
// the rate limit and burst to zero.
func disableKmsgRateLimit() error {
	if !common.IsPathExists("/proc/sys") {
		mount.Mount(mount.Mountpoints.Proc)
		defer mount.Unmount(mount.Mountpoints.Proc.Target, 0)
	}

	params := map[string]string{
		"/proc/sys/kernel/printk_ratelimit":       "0", // Deactivate rate limit.
		"/proc/sys/kernel/printk_ratelimit_burst": "0", // Disable burst.
		// "/proc/sys/kernel/printk_devkmsg":         "on", // Enable devkmsg.
	}

	for path, value := range params {
		if err := os.WriteFile(path, []byte(value), 0644); err != nil {
			fmt.Printf("cannot write to %s: %v\n", path, err)
		}
	}

	return nil
}

func logToDevKmsg() {
	disableKmsgRateLimit()

	if !common.IsPathExists("/dev/kmsg") {
		mount.Mount(mount.Mountpoints.Dev)
		defer mount.Unmount(mount.Mountpoints.Dev.Target, 0)
	}

	w, err := os.OpenFile("/dev/kmsg", os.O_WRONLY, 0o644)
	if err == nil {
		log.Output = w
		l.SetOutput(w)
	}
}
