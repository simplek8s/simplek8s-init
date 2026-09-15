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
	"context"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/simplek8s/simplek8s-init/pkg/common"
	"github.com/simplek8s/simplek8s-init/pkg/linux/mount"
)

const timeout = 5 * time.Second

var (
	mu        sync.Mutex
	populated bool
)

// PopulateDev populates the /dev directory with necessary devices.
// For example, this function will populates /dev/disk/by-label.
//
// It is idempotent: concurrent calls are serialized and a previous success
// is cached. A previous failure is NOT cached, so a later call retries.
func PopulateDev() error {
	mu.Lock()
	defer mu.Unlock()

	if populated {
		return nil
	}

	if err := populateDev(); err != nil {
		return err
	}

	populated = true
	return nil
}

func populateDev() error {
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
	cmd := exec.Command("/usr/lib/systemd/systemd-udevd")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start systemd-udevd: %w", err)
	}
	// Ensure the daemon does not leak on failure.
	daemonRunning := true
	defer func() {
		if daemonRunning {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	// Trigger events (udevadm trigger/settle) with timeouts so a
	// hung udevadm cannot block PID 1 forever.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := exec.CommandContext(ctx, "/usr/bin/udevadm", "trigger", "--action=add").Run(); err != nil {
		return fmt.Errorf("cannot trigger udev events: %w", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := exec.CommandContext(ctx, "/usr/bin/udevadm", "settle").Run(); err != nil {
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
		_ = cmd.Process.Kill()
		<-done
		return fmt.Errorf("timeout waiting for systemd-udevd to exit")
	case err := <-done:
		daemonRunning = false
		if err != nil {
			return fmt.Errorf("systemd_udevd exited with error: %w", err)
		}
	}

	return nil
}
