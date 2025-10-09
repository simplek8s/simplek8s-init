// Copyright 2025 José Luis Salvador Rufo
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

// Package main will bootstrap from the initrd stage to the next stage.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jlsalvador/simplek8s/pkg/cp"
	"github.com/jlsalvador/simplek8s/pkg/linux/mount"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/bootstrap"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/sysroot"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/sysroot/feeder"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// Create and mount /sysroot.
func mountNextRoot(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	nextRoot := mount.MountPoint{
		Target: where,
		Chmod:  0755,
		Source: "tmpfs",
		Fstype: "tmpfs",
		Flags:  mount.MountFlagNoSUID | mount.MountFlagNoDev,
		Data:   "size=90%",
	}

	if err := mount.Mount(nextRoot); err != nil {
		return err
	}

	// if err := mount.MountPseudoFS(where); err != nil {
	// 	log.WithError(err).Error("can not mount pseudofs into " + where)
	// 	return err
	// }

	return nil
}

func populateUsr(where string) error {
	// Mount tmpfs as /usr.
	usr := mount.MountPoint{
		Target: where,
		Chmod:  0755,
		Source: "tmpfs",
		Fstype: "tmpfs",
		Flags:  unix.MS_NOSUID | unix.MS_NODEV,
		Data:   "size=90%",
	}

	if err := mount.Mount(usr); err != nil {
		log.WithError(err).Error("can not mount tmpfs into " + where)
	}

	// Populate /usr from initrd.
	if err := cp.Copy("/usr", where, &cp.CopyOptions{
		PreserveAll: true,
		Overwrite:   true,
	}); err != nil {
		log.WithError(err).Error("can not copy from /usr to " + where)
		return err
	}

	// Create /usr/libexec and /usr/local for future bind mounts from /var.
	for _, d := range []string{"/libexec", "/local"} {
		dst := filepath.Join(where, d)
		if err := os.MkdirAll(dst, 0755); err != nil {
			log.WithError(err).Error("can not create directory " + dst)
			return err
		}
	}

	// Remount /usr as RO.
	if err := mount.Mount(mount.MountPoint{
		Target: usr.Target,
		Chmod:  usr.Chmod,
		Source: "",
		Fstype: "",
		Flags:  unix.MS_REMOUNT | unix.MS_RDONLY,
		Data:   "",
	}); err != nil {
		log.WithError(err).Error("can not remount as RO " + filepath.Join(where, "/usr"))
		return err
	}

	return nil
}

func createNextRoot(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	if err := mountNextRoot(where); err != nil {
		log.WithError(err).Error("can not mount next root " + where)
		return err
	}

	usrDst := filepath.Join(where, "/usr")
	if err := populateUsr(usrDst); err != nil {
		log.WithError(err).Error("can not populate " + usrDst)
		return err
	}

	if err := sysroot.CreateLegacySymlinks(where); err != nil {
		log.WithError(err).Error("can not create deprecated symlinks into " + where)
		return err
	}

	sr := &sysroot.Sysroot{}

	// //TODO: Maybe remove FeedSysrootByFiles and only use the bootstrap config.
	// if err := feeder.FeedSysrootByFiles(sr, "/"); err != nil {
	// 	log.WithError(err).Error("can not feed by current initrd files")
	// 	return err
	// }

	if err := feeder.FeedSysrootByDefault(sr); err != nil {
		log.WithError(err).Error("can not feed sysroot by default")
		return err
	}

	// Retrive SimpleK8s bootstrap config.
	config, err := bootstrap.GetConfig()
	if err != nil {
		log.WithError(err).Warn("can not fetch bootstrap config")
	} else if config != nil {
		// //DEBUG: Print out the config structure.
		// fmt.Println("simplek8s.yaml:\n```yaml\n" + config.String() + "\n```")

		if err := bootstrap.FeedSysrootByBootstrapConfig(sr, *config); err != nil {
			log.WithError(err).Error("can not feed sysroot by bootstrap config")
			return err
		}
	}

	// //DEBUG: Print out the sysroot structure.
	// fmt.Println("sysroot:\n```json\n" + sr.String() + "\n```")

	if err := sr.Write(where); err != nil {
		log.WithError(err).Error("can not write sysroot config")
		return err
	}

	return nil
}

// Switch over to the next root and exec to the next init.
func switchRoot(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	// unix.Mount(newroot, "/", "", unix.MS_MOVE, "")

	if err := unix.Chroot(where); err != nil {
		log.WithError(err).Error("can not chroot into " + where)
		return err
	}

	// unix.Chdir("/")

	for _, init := range []string{"/sbin/init"} {
		if _, err := os.Stat(init); errors.Is(err, os.ErrNotExist) {
			continue
		}

		if err := unix.Exec(init, []string{init}, os.Environ()); err != nil {
			log.WithError(err).Error("can not exec " + init)
			return err
		}
	}

	return nil
}

// It:
//   - Opens /dev/kmsg for kernel‑message logging.
//   - Configures the logger.
//   - Verifies that the process is PID 1.
//   - Prepares the next root filesystem and switches to it.
//   - Exits cleanly.
func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
	log.SetLevel(log.InfoLevel)

	log.Debug("start")
	defer log.Debug("end")

	// Check if we are PID 1, warns if not.
	if os.Getpid() != 1 {
		log.Fatal("not PID 1")
	}

	where := "/sysroot"

	if err := createNextRoot(where); err != nil {
		log.WithError(err).Fatal("can not populate next root " + where)
	}

	// // DEBUG: Drop to shell.
	// unix.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ())

	if err := switchRoot(where); err != nil {
		log.WithError(err).Fatal("can not chroot to " + where)
	}

	fmt.Println("Exiting...")
	os.Exit(0)
}
