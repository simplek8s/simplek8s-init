// Copyright 2020 José Luis Salvador Rufo

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

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

	"github.com/jlsalvador/simplek8s/internal/pkg/initrd"
	"github.com/jlsalvador/simplek8s/pkg/linux"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/bootstrap"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// Create and mount /sysroot.
func mountNextRoot() (where string, err error) {
	log.Debug("start")
	defer log.Debug("end")

	nextRoot := linux.MountPoint{
		Target: "/sysroot",
		Chmod:  0755,
		Source: "tmpfs",
		Fstype: "tmpfs",
		Flags:  unix.MS_NOSUID | unix.MS_NODEV,
		Data:   "size=90%,mode=755",
	}

	if err := linux.Mount(nextRoot); err != nil {
		return "", err
	}

	// if err := linux.MountPseudoFS(where); err != nil {
	// 	log.WithError(err).Error("can not mount pseudofs into " + where)
	// 	return "", err
	// }

	return nextRoot.Target, nil
}

// Copy usr files from initrd and mount required filesystems into /sysroot.
func populateNextRoot(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	// Retrive SimpleK8s bootstrap config.
	config, err := bootstrap.GetConfig()
	if err != nil {
		log.WithError(err).Error("can not fetch bootstrap config")
	} else {
		log.WithField("config", config).Info("fetched bootstrap config")
	}

	//TODO: Mount /var.

	//TODO: Mount /var binds into /.

	if err := os.Mkdir(filepath.Join(where, "/usr"), 0755); err != nil {
		log.WithError(err).Error("can not create directory /usr into " + where)
		return err
	}

	if err := linux.CreateDeprecatedSymlinks(where); err != nil {
		log.WithError(err).Error("can not create deprecated symlinks into " + where)
		return err
	}

	if err := initrd.PopulateRoot(where); err != nil {
		log.WithError(err).Error("can not populate " + filepath.Join(where, "/usr"))
		return err
	}

	//TODO: populate using bootstrap config.

	// // Remount /usr as RO.
	// if err := unix.Mount("", filepath.Join(where, "/usr"), "", 0, "remount,ro"); err != nil {
	// 	log.WithError(err).Error("can not remount as RO " + filepath.Join(where, "/usr"))
	// 	return err
	// }

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

// main will:
//   - Mount /sysroot
//   - Populate /sysroot
//   - Switch root to /sysroot
func main() {
	log.Debug("start")
	defer log.Debug("end")

	// Check if we are PID 1, warns if not.
	if os.Getpid() != 1 {
		log.Fatal("not PID 1")
	}

	// Enable debug logging.
	// log.SetLevel(log.DebugLevel)
	// log.SetReportCaller(true)

	//DEBUG: Drop to shell.
	// unix.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ())

	where, err := mountNextRoot()
	if err != nil {
		log.WithError(err).Fatal("can not mount next root")
	}

	if err := populateNextRoot(where); err != nil {
		log.WithError(err).Fatal("can not populate next root " + where)
	}

	if err := switchRoot(where); err != nil {
		log.WithError(err).Fatal("can not chroot to " + where)
	}

	fmt.Println("Exiting...")
	os.Exit(0)
}
