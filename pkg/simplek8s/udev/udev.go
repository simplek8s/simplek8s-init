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

// Package udev package provides functions to interact with the udev system.
package udev

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/mount"
)

const timeout = 5 * time.Second

var once sync.Once

// PopulateDev populates the /dev directory with necessary devices.
// For example, this function will populates /dev/disk/by-label.
func PopulateDev() error {
	var e error

	once.Do(func() {
		// systemd-udevd requires /dev, /sys, and /proc.
		if !common.IsPathExists("/dev/kmsg") {
			if err := mount.Mount(mount.Mountpoints.Dev); err != nil {
				e = fmt.Errorf("cannot mount /dev: %w", err)
				return
			}
			//defer mount.Unmount(mount.Mountpoints.Dev.Target, 0)
		}
		if !common.IsPathExists("/proc/cmdline") {
			if err := mount.Mount(mount.Mountpoints.Proc); err != nil {
				e = fmt.Errorf("cannot mount /proc: %w", err)
				return
			}
			//defer mount.Unmount(mount.Mountpoints.Proc.Target, 0)
		}
		if !common.IsPathExists("/sys/class") {
			if err := mount.Mount(mount.Mountpoints.Sys); err != nil {
				e = fmt.Errorf("cannot mount /sys: %w", err)
				return
			}
			//defer mount.Unmount(mount.Mountpoints.Sys.Target, 0)
		}

		// Execute systemd-udevd as daemon on background.
		cmd := exec.Command("/usr/lib/systemd/systemd-udevd")
		if err := cmd.Start(); err != nil {
			e = fmt.Errorf("cannot start systemd-udevd: %w", err)
			return
		}

		// Trigger events (udevadm trigger/settle).
		if err := exec.Command("/usr/bin/udevadm", "trigger", "--action=add").Run(); err != nil {
			e = fmt.Errorf("cannot trigger udev events: %w", err)
			return
		}
		if err := exec.Command("/usr/bin/udevadm", "settle").Run(); err != nil {
			e = fmt.Errorf("cannot settle udev events: %w", err)
			return
		}

		// Stop systemd-udevd daemon.
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			e = fmt.Errorf("cannot send SIGTERM to systemd_udevd: %w", err)
			return
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
				e = fmt.Errorf("systemd_udevd exited with error: %w", err)
				return
			}
		}
	})

	return e
}
