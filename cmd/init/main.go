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
	"github.com/jlsalvador/simplek8s/pkg/linux"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/bootstrap"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/sysroot"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/sysroot/feeder"
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
		Data:   "size=90%",
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

func populateUsr(where string) error {
	usr := linux.MountPoint{
		Target: where,
		Chmod:  0755,
		Source: "tmpfs",
		Fstype: "tmpfs",
		Flags:  unix.MS_NOSUID | unix.MS_NODEV,
		Data:   "size=90%",
	}

	if err := linux.Mount(usr); err != nil {
		log.WithError(err).Error("can not mount tmpfs into " + where)
	}

	if err := cp.Copy("/usr", where, &cp.CopyOptions{
		PreserveAll: true,
		Overwrite:   true,
	}); err != nil {
		log.Error(err)
		return err
	}

	if err := linux.Mount(linux.MountPoint{
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

func populateNextRoot(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	//TODO: Mount `{where}/var`.

	//TODO: Mount `{where}/var` binds into `{where}/`.

	usrDst := filepath.Join(where, "/usr")
	if err := populateUsr(usrDst); err != nil {
		log.WithError(err).Error("can not populate " + usrDst)
		return err
	}

	if err := linux.CreateDeprecatedSymlinks(where); err != nil {
		log.WithError(err).Error("can not create deprecated symlinks into " + where)
		return err
	}

	sr := &sysroot.Sysroot{}

	//TODO: Maybe remove FeedSysrootByFiles and only use the bootstrap config.
	if err := feeder.FeedSysrootByFiles(sr, "/"); err != nil {
		log.WithError(err).Error("can not feed by current initrd files")
		return err
	}

	//DEBUG: Print out the sysroot structure.
	// o, _ := json.MarshalIndent(sr, "", " ")
	// fmt.Printf("sr: %s\n", o)

	// Retrive SimpleK8s bootstrap config.
	config, err := bootstrap.GetConfig()
	if err != nil {
		log.WithError(err).Error("can not fetch bootstrap config")
	} else if config != nil {
		if err := bootstrap.FeedSysrootByBootstrapConfig(sr, *config); err != nil {
			log.WithError(err).Error("can not feed by bootstrap config")
			return err
		}
	}

	//DEBUG: Drop to shell.
	// unix.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ())

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

type KmsgFormatter struct {
	Ident string
}

func (f *KmsgFormatter) Format(entry *log.Entry) ([]byte, error) {
	var level int
	switch entry.Level {
	case log.PanicLevel, log.FatalLevel:
		level = 0
	case log.ErrorLevel:
		level = 3
	case log.WarnLevel:
		level = 4
	case log.InfoLevel:
		level = 6
	default:
		level = 7 // Debug/Trace
	}

	ft := log.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: true,
	}
	msg, _ := ft.Format(entry)

	return fmt.Appendf(nil, "<%d>%s[%d]: %s", level, f.Ident, os.Getpid(), msg), nil
}

// main will:
//   - Mount /sysroot
//   - Populate /sysroot
//   - Switch root to /sysroot
func main() {
	if err := linux.Mount(linux.Mountpoints.Dev); err != nil {
		log.SetOutput(os.Stdout)
		log.WithError(err).Warn("Falling back to stdout, /dev/kmsg not available")
	}

	// kernel-parameters for debug: ignore_loglevel ignore_rlimit_data

	kmsg, err := os.OpenFile("/dev/kmsg", os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.SetOutput(os.Stdout)
		log.WithError(err).Warn("Falling back to stdout, /dev/kmsg not available")
	} else {
		defer kmsg.Close()
	}
	log.SetOutput(kmsg)

	log.SetFormatter(&KmsgFormatter{Ident: "simplek8s-init"})
	log.SetLevel(log.InfoLevel)

	log.Debug("start")
	defer log.Debug("end")

	// Check if we are PID 1, warns if not.
	if os.Getpid() != 1 {
		log.Fatal("not PID 1")
	}

	where, err := mountNextRoot()
	if err != nil {
		log.WithError(err).Fatal("can not mount next root")
	}

	if err := populateNextRoot(where); err != nil {
		log.WithError(err).Fatal("can not populate next root " + where)
	}

	//DEBUG: Drop to shell.
	// unix.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ())

	if err := switchRoot(where); err != nil {
		log.WithError(err).Fatal("can not chroot to " + where)
	}

	fmt.Println("Exiting...")
	os.Exit(0)
}
