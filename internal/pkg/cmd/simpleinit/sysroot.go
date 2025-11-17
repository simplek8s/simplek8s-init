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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"

	"simplek8s/pkg/common"
	"simplek8s/pkg/cp"
	"simplek8s/pkg/linux/mount"
	"simplek8s/pkg/linux/procfs"
	"simplek8s/pkg/log"
	"simplek8s/pkg/simplek8s"
	"simplek8s/pkg/simplek8s/bootstrap"
	"simplek8s/pkg/simplek8s/generateshadow"
	"simplek8s/pkg/simplek8s/sysroot"
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
	// Required to write systemd transient units and "simplek8s.yaml".
	mRun := mount.Mountpoints.Run
	mRun.Target = filepath.Join(where, mRun.Target)
	if err := mount.Mount(mRun); err != nil {
		return fmt.Errorf("cannot mount %s: %w", mRun.Target, err)
	}

	sr := &sysroot.Sysroot{
		Links: []sysroot.Link{{
			Overwrite: true,
			Path:      "/bin",
			Target:    "usr/bin",
			UID:       0,
			GID:       0,
			Hard:      false,
		}, {
			Overwrite: true,
			Path:      "/sbin",
			Target:    "usr/sbin",
			UID:       0,
			GID:       0,
			Hard:      false,
		}, {
			Overwrite: true,
			Path:      "/lib",
			Target:    "usr/lib",
			UID:       0,
			GID:       0,
			Hard:      false,
		}, {
			Overwrite: true,
			Path:      "/lib64",
			Target:    "lib",
			UID:       0,
			GID:       0,
			Hard:      false,
		}},
		Files: []sysroot.File{{
			Overwrite: false,
			Filename:  "/etc/machine-id",
			Content:   []byte("uninitialized\n"),
			Mode:      0o644,
			UID:       0,
			GID:       0,
		}},
		Mounts: []sysroot.Mount{{
			What:    "none",
			Where:   "/var",
			Type:    "tmpfs",
			Options: "size=90%",
		}, {
			What:    "/var/etc",
			Where:   "/etc",
			Type:    "none",
			Options: "rbind",
		}, {
			What:    "/var/home",
			Where:   "/home",
			Type:    "none",
			Options: "rbind",
		}, {
			What:    "/var/mnt",
			Where:   "/mnt",
			Type:    "none",
			Options: "rbind",
		}, {
			What:    "/var/opt",
			Where:   "/opt",
			Type:    "none",
			Options: "rbind",
		}, {
			What:    "/var/root",
			Where:   "/root",
			Type:    "none",
			Options: "rbind",
		}, {
			What:    "/var/usr/libexec/kubernetes",
			Where:   "/usr/libexec/kubernetes",
			Type:    "none",
			Options: "rbind",
		}, {
			What:    "/var/usr/local",
			Where:   "/usr/local",
			Type:    "none",
			Options: "rbind",
		}},
	}

	// Get timeout from rootwait cmdline.
	timeout, _ := procfs.GetCmdlineValue("rootwait", 10)

	// Show feedback to user about fetching simplek8s.yaml.
	fmt.Printf("Fetching simplek8s.yaml... (%ds)\n", timeout)

	// Retrive SimpleK8s bootstrap config.
	config, err := bootstrap.GetConfig(time.Duration(timeout) * time.Second)
	if err != nil {
		log.Warnf("cannot fetch bootstrap config: %v", err)
	} else if config != nil {
		log.DebugFn(func() string { return fmt.Sprintf("simplek8s.yaml:\n```yaml\n%s\n```", config.String()) })

		if err := bootstrap.FeedSysrootByBootstrapConfig(sr, *config); err != nil {
			return fmt.Errorf("cannot feed sysroot by bootstrap config: %w", err)
		}
	} else {
		// There is not bootstrap config.

		// Generate root password.
		plain, hash, err := generateshadow.GeneratePwd("root")
		if err != nil {
			return fmt.Errorf("can not generate root pwd: %w", err)
		}
		for _, f := range []sysroot.File{{
			Overwrite: true,
			Filename:  "/run/credstore/passwd.hashed-password.root",
			Content:   []byte(hash),
			Mode:      0o400,
			UID:       0,
			GID:       0,
		}, {
			Overwrite: true,
			Filename:  "/run/issue.d/80-root-random-password.issue",
			Content:   fmt.Appendf(nil, "\n\\e{red}You are running a non persistent session!\\e{reset}\n  Root pwd: %s\n", plain),
			Mode:      0o644,
			UID:       0,
			GID:       0,
		}} {
			sr.Files = common.UpdateOrAppend(sr.Files, f, func(a sysroot.File, b sysroot.File) bool {
				return a.Filename == b.Filename
			})
		}
	}

	if simplek8s.IsDebug() {
		fmt.Println("sysroot:\n```json\n" + sr.String() + "\n```")
	}

	if err := sr.Write(where); err != nil {
		return fmt.Errorf("cannot write sysroot config: %w", err)
	}

	return nil
}

// Switch over to the next root and exec to the next init.
func switchRoot(where string) error {
	log.Trace("start")
	defer log.Trace("end")

	// unix.Mount(newroot, "/", "", unix.MS_MOVE, "")

	if simplek8s.IsDebug() {
		fmt.Printf("Debug is true, dropping to a shell ...\nExecute `\033[1mexec chroot /sysroot /sbin/init\033[0m` to continue the boot sequence.\n\n")
		unix.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ())
	}

	// Enter chroot.
	if err := unix.Chroot(where); err != nil {
		return fmt.Errorf("cannot chroot into %s: %w", where, err)
	}

	// Change working directory into chroot.
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("cannot chdir to %s: %w", where, err)
	}

	// Execute next init.
	for _, init := range []string{"/sbin/init"} {
		if _, err := os.Stat(init); errors.Is(err, os.ErrNotExist) {
			continue
		}

		if err := unix.Exec(init, []string{init}, os.Environ()); err != nil {
			return fmt.Errorf("cannot exec %s: %w", init, err)
		}
	}

	return nil
}
