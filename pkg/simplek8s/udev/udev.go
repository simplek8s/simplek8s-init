package udev

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/mount"
)

const timeout = 5 * time.Second

func PopulateDev() error {
	// systemd-udevd requires /dev, /sys, and /proc.
	if !common.IsPathExists("/dev/kmsg") {
		if err := mount.Mount(mount.Mountpoints.Dev); err != nil {
			return fmt.Errorf("cannot mount /dev: %w", err)
		}
		//defer mount.Unmount(mount.Mountpoints.Dev.Target, 0)
	}
	if !common.IsPathExists("/proc/cmdline") {
		if err := mount.Mount(mount.Mountpoints.Proc); err != nil {
			return fmt.Errorf("cannot mount /proc: %w", err)
		}
		//defer mount.Unmount(mount.Mountpoints.Proc.Target, 0)
	}
	if !common.IsPathExists("/sys/class") {
		if err := mount.Mount(mount.Mountpoints.Sys); err != nil {
			return fmt.Errorf("cannot mount /sys: %w", err)
		}
		//defer mount.Unmount(mount.Mountpoints.Sys.Target, 0)
	}

	// Execute systemd-udevd as daemon on background.
	cmd := exec.Command("/usr/lib/systemd/systemd-udevd", "--daemon")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start systemd-udevd: %w", err)
	}

	// Trigger events (udevadm trigger/settle).
	if err := exec.Command("/usr/bin/udevadm", "trigger", "--action=add").Run(); err != nil {
		return fmt.Errorf("cannot trigger udev events: %w", err)
	}
	if err := exec.Command("/usr/bin/udevadm", "settle").Run(); err != nil {
		return fmt.Errorf("cannot settle udev events: %w", err)
	}

	// Stop systemd-udevd daemon.
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("cannot send SIGTERM to systemd_udevd: %w", err)
	}

	// Wait for systemd-udevd to finish or kill it on timeout.
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(timeout):
		cmd.Process.Kill()
		cmd.Wait()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("systemd_udevd exited with error: %w", err)
		}
	}

	return nil
}
