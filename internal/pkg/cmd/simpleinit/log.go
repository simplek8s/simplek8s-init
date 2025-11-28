package simpleinit

import (
	"fmt"
	"os"

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/mount"
	"simplek8s/pkg/log"

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
