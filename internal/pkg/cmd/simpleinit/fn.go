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
	"path/filepath"

	"golang.org/x/sys/unix"

	"simplek8s/pkg/linux/mount"
	"simplek8s/pkg/simplek8s"
)

func dropToShell(where string, reason string) error {
	// Because shell requires pseudo filesystems, binds them from sysroot.
	for _, s := range []string{
		mount.Mountpoints.Dev.Target,
		mount.Mountpoints.Proc.Target,
		mount.Mountpoints.Sys.Target,
	} {
		mp := mount.MountPoint{
			Target: s,
			Chmod:  0o755,
			Source: filepath.Join(where, s),
			Fstype: "none",
			Flags:  mount.MountFlagRBind,
		}
		if err := mount.Mount(mp); err != nil {
			return fmt.Errorf("cannot mount %s at %s: %w", mp.Source, mp.Target, err)
		}
	}

	fmt.Printf("%s\n\n", reason)
	return unix.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ())
}

func bootNextRoot(where string) error {
	if err := createSysroot(where); err != nil {
		return fmt.Errorf("cannot populate next root %s: %w", where, err)
	}

	if simplek8s.IsDebug() {
		return dropToShell(where, "Debug is true, dropping to a shell ...\nExecute `\033[1mexec /usr/lib/simplek8s/switchroot\033[0m` to continue the boot sequence.")
	}

	//TODO: Cleanup initrd rootfs.

	if err := SwitchRoot(where); err != nil {
		return fmt.Errorf("cannot chroot to %s: %w", where, err)
	}

	return nil
}
