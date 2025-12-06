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

// Package simpleinit bootstrap from (initrd) root to the next (systemd) root.
package simpleinit

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/simplek8s/simplek8s-init/pkg/common"
	"github.com/simplek8s/simplek8s-init/pkg/cp"
	"github.com/simplek8s/simplek8s-init/pkg/linux/mount"
	"github.com/simplek8s/simplek8s-init/pkg/linux/procfs"
	"github.com/simplek8s/simplek8s-init/pkg/log"
	"github.com/simplek8s/simplek8s-init/pkg/simplek8s"
	"github.com/simplek8s/simplek8s-init/pkg/simplek8s/bootstrap"
	"github.com/simplek8s/simplek8s-init/pkg/simplek8s/sysroot"
	"github.com/simplek8s/simplek8s-init/pkg/simplek8s/sysroot/writer/systemd"

	"golang.org/x/sys/unix"
)

// createUsr mounts tmpfs into `where` (normally `/sysroot/usr`), populates it
// from `/usr`, and remounts it as read-only.
func createUsr(where string) error {
	log.Trace("start")
	defer log.Trace("end")

	// Mount `${where}/usr` mountpoint.
	mUsr := mount.Mountpoints.Usr
	mUsr.Target = where
	if err := mount.Mount(mUsr); err != nil {
		return fmt.Errorf("cannot mount %s: %w", mUsr.Target, err)
	}

	// Populate `${where}/usr` from initrd.
	if err := cp.Copy("/usr", where, &cp.CopyOptions{
		PreserveAll: true,
		Overwrite:   true,
	}); err != nil {
		return fmt.Errorf("cannot copy from /usr to %s: %w", mUsr.Target, err)
	}

	// Create `${where}/usr/libexec/kubernetes` and `${where}/usr/local` for future bind mounts from `${where}/var`.
	for _, d := range []string{"/libexec/kubernetes", "/local"} {
		dst := filepath.Join(where, d)
		if err := os.MkdirAll(dst, 0o755); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", dst, err)
		}
	}

	// Remount `${where}/usr` as RO.
	if err := mount.Mount(mount.MountPoint{
		Target: where,
		Chmod:  mount.Mountpoints.Usr.Chmod,
		Flags:  mount.MountFlagRemount | mount.MountFlagReadOnly,
	}); err != nil {
		return fmt.Errorf("cannot remount as RO %s: %w", where, err)
	}

	return nil
}

func createSysroot(where string) error {
	log.Trace("start")
	defer log.Trace("end")

	// Mount /sysroot (tmpfs).
	mSysroot := mount.Mountpoints.Sysroot
	mSysroot.Target = filepath.Join(where, mSysroot.Target)
	if err := mount.Mount(mSysroot); err != nil {
		return fmt.Errorf("cannot mount %s: %w", mSysroot.Target, err)
	}

	// Mount and create /sysroot/usr as read-only.
	usrDst := filepath.Join(where, "/usr")
	if err := createUsr(usrDst); err != nil {
		return fmt.Errorf("cannot create %s, %w", usrDst, err)
	}

	// Mount /sysroot/run (tmpfs).
	// Required to write "simplek8s.yaml".
	mRun := mount.Mountpoints.Run
	mRun.Target = filepath.Join(where, mRun.Target)
	if err := mount.Mount(mRun); err != nil {
		return fmt.Errorf("cannot mount %s: %w", mRun.Target, err)
	}

	sr := &sysroot.Sysroot{
		Links: []sysroot.Link{
			// Legacy symlinks.
			{
				Overwrite: false,
				Path:      "/bin",
				Target:    "usr/bin",
				UID:       0,
				GID:       0,
				Hard:      false,
			}, {
				Overwrite: false,
				Path:      "/sbin",
				Target:    "usr/sbin",
				UID:       0,
				GID:       0,
				Hard:      false,
			}, {
				Overwrite: false,
				Path:      "/lib",
				Target:    "usr/lib",
				UID:       0,
				GID:       0,
				Hard:      false,
			}, {
				Overwrite: false,
				Path:      "/lib64",
				Target:    "lib",
				UID:       0,
				GID:       0,
				Hard:      false,
			},
		},
		// Files: []sysroot.File{{
		// 	Overwrite: false,
		// 	Filename:  "/etc/machine-id",
		// 	Content:   []byte("uninitialized\n"),
		// 	Mode:      0o644,
		// 	UID:       0,
		// 	GID:       0,
		// }},
		Mounts: []mount.MountPoint{
			// Persist into /var.
			{
				Target: "/var",
				Chmod:  0o755,
				Source: "tmpfs",
				Fstype: "tmpfs",
				Flags:  0,
				Data:   "size=90%",
			},
			// Binds from /var.
			{
				Target: "/etc",
				Chmod:  0o755,
				Source: "/var/etc",
				Fstype: "none",
				Flags:  mount.MountFlagRBind,
				Data:   "",
			}, {
				Target: "/home",
				Chmod:  0o755,
				Source: "/var/home",
				Fstype: "none",
				Flags:  mount.MountFlagRBind,
				Data:   "",
			}, {
				Target: "/mnt",
				Chmod:  0o755,
				Source: "/var/mnt",
				Fstype: "none",
				Flags:  mount.MountFlagRBind,
				Data:   "",
			}, {
				Target: "/opt",
				Chmod:  0o755,
				Source: "/var/opt",
				Fstype: "none",
				Flags:  mount.MountFlagRBind,
				Data:   "",
			}, {
				Target: "/root",
				Chmod:  0o755,
				Source: "/var/root",
				Fstype: "none",
				Flags:  mount.MountFlagRBind,
				Data:   "",
			}, {
				Target: "/usr/libexec/kubernetes",
				Chmod:  0o755,
				Source: "/var/usr/libexec/kubernetes",
				Fstype: "none",
				Flags:  mount.MountFlagRBind,
				Data:   "",
			}, {
				Target: "/usr/local",
				Chmod:  0o755,
				Source: "/var/usr/local",
				Fstype: "none",
				Flags:  mount.MountFlagRBind,
				Data:   "",
			},
		},
	}

	// Get timeout from rootwait cmdline.
	if !common.IsPathExists(procfs.CmdlineFilepath) {
		if err := mount.Mount(mount.Mountpoints.Proc); err != nil {
			return fmt.Errorf("cannot mount %s: %w", mount.Mountpoints.Proc.Target, err)
		}
	}
	cmdline, err := common.ReadFileAsString(procfs.CmdlineFilepath)
	if err != nil {
		return fmt.Errorf("cannot read cmdline: %w", err)
	}
	timeout, _ := procfs.GetCmdlineValue(string(cmdline), "rootwait", 10)

	// Show feedback to user about fetching simplek8s.yaml.
	fmt.Printf("Fetching simplek8s.yaml... (%ds)\n", timeout)

	// Retrive SimpleK8s bootstrap config.
	cfgPath, config, err := bootstrap.GetConfig(time.Duration(timeout) * time.Second)
	if err != nil {
		log.Warnf("cannot fetch bootstrap config: %v", err)
	} else if config != nil {
		fmt.Printf("Found simplek8s.yaml at: %s\n", cfgPath)
		log.DebugFn(func() string { return fmt.Sprintf("```yaml\n%s\n```", config.String()) })

		if err := bootstrap.FeedSysrootByBootstrapConfig(sr, *config); err != nil {
			return fmt.Errorf("cannot feed sysroot by bootstrap config: %w", err)
		}
	} else {
		fmt.Println("simplek8s.yaml not found. Booting a non persistent session...")
	}

	if simplek8s.IsDebug() {
		fmt.Println("sysroot:\n```json\n" + sr.String() + "\n```")
	}

	if err := systemd.Write(sr, where); err != nil {
		return fmt.Errorf("cannot write sysroot config: %w", err)
	}

	// Move pseudo filesystems mountpoints from initrd to sysroot.
	for _, s := range []string{
		mount.Mountpoints.Dev.Target,
		mount.Mountpoints.Proc.Target,
		mount.Mountpoints.Sys.Target,
	} {
		mp := mount.MountPoint{
			Target: filepath.Join(where, s),
			Chmod:  0o755,
			Source: s,
			Fstype: "none",
			Flags:  mount.MountFlagMove,
		}
		if err := mount.Mount(mp); err != nil {
			return fmt.Errorf("cannot mount %s at %s: %w", mp.Source, mp.Target, err)
		}
	}

	return nil
}

// Switch over to the next root and exec to the next init.
func SwitchRoot(where string) error {
	log.Trace("start")
	defer log.Trace("end")

	// Change workdir to sysroot.
	if err := os.Chdir(where); err != nil {
		return fmt.Errorf("cannot chdir to %s: %w", where, err)
	}

	// Replace rootfs by sysroot.
	if err := mount.Mount(mount.MountPoint{
		Target: "/",
		Chmod:  0o755,
		Source: where,
		Fstype: "none",
		Flags:  mount.MountFlagMove,
	}); err != nil {
		return fmt.Errorf("cannot replace rootfs by %s: %w", where, err)
	}

	// Change process rootfs.
	if err := unix.Chroot("."); err != nil {
		return fmt.Errorf("cannot chroot to %s: %w", where, err)
	}

	// Change-back to the new rootfs (sysroot).
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("cannot chdir to / after chroot: %w", err)
	}

	// Execute next init.
	init := "/sbin/init"
	if err := unix.Exec(init, []string{init}, os.Environ()); err != nil {
		return fmt.Errorf("cannot exec %s: %w", init, err)
	}

	return fmt.Errorf("unexpected exit from exec %s", init)
}
