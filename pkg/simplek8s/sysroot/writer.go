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

// Package sysroot populates a new root to boot.
package sysroot

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/mount"
	"simplek8s/pkg/linux/passwd"
	"simplek8s/pkg/simplek8s/udev"

	log "github.com/sirupsen/logrus"
)

// Creates any necessary parent directories if they do not exist.
// Set the correct permission and ownership for the filename.
func ensureWriteFile(filename string, content []byte, mode fs.FileMode, uid int, gid int) error {
	log.WithFields(log.Fields{
		"filename": filename,
		"content":  content,
		"mode":     mode,
		"uid":      uid,
		"gid":      gid,
	}).Trace("start")
	defer log.Trace("end")

	path := filepath.Dir(filename)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filename, content, mode); err != nil {
		return err
	}
	if err := os.Chown(filename, uid, gid); err != nil {
		return err
	}
	return nil
}

func resolveFlags(opts []string) (mount.MountFlag, string) {
	var flags mount.MountFlag
	data := []string{}

	for _, opt := range opts {
		if f, ok := mount.MountFlags[opt]; ok {
			flags |= f
		} else {
			data = append(data, opt)
		}
	}

	return flags, strings.Join(data, ",")
}

// Because all of these requirements, we are going to mount all
// mountpoints in the initrd stage:
//
//   - /sysroot/etc must be mounted by initrd, because systemd requires rootfs
//     or /etc to be mounted before exec /sbin/init.
//   - /sysroot/etc requires /sysroot/var to be mounted, because /sysroot/etc
//     binds to /sysroot/var/etc by default.
//   - simplek8s.yaml could defines more complex environments to mount /etc.
func writeMounts(mounts []Mount, where string) error {
	log.WithFields(log.Fields{
		"mounts": mounts,
		"where":  where,
	}).Trace("start")
	defer log.Trace("end")

	// Fill /dev by udev.
	// This step is required for example to search disk by label or UUID.
	if err := udev.PopulateDev(); err != nil {
		return fmt.Errorf("cannot populate /dev by udev: %w", err)
	}

	for _, m := range mounts {
		flags, data := resolveFlags(strings.Split(m.Options, ","))
		mp := mount.MountPoint{
			Target: filepath.Join(where, m.Where),
			Chmod:  0o755,
			Source: m.What,
			Fstype: m.Type,
			Flags:  flags,
			Data:   data,
		}

		if mp.Flags&mount.MountFlagBind != 0 {
			// Fix source for bind mounts.
			mp.Source = filepath.Join(where, mp.Source)

			// Ensure the source directory exists for bind mounts.
			if err := os.MkdirAll(mp.Source, mp.Chmod); err != nil {
				return fmt.Errorf("cannot create directory %s: %w", mp.Target, err)
			}
		}

		if err := mount.Mount(mp); err != nil {
			return fmt.Errorf("cannot mount %s in %s: %w", mp.Source, mp.Target, err)
		}
	}

	return nil
}

func writeShadows(shadows []passwd.Shadow, where string) error {
	log.WithFields(log.Fields{
		"shadows": shadows,
		"where":   where,
	}).Trace("start")
	defer log.Trace("end")

	if len(shadows) == 0 {
		return nil
	}

	content := ""
	for _, s := range shadows {
		if line, err := s.Marshal(); err != nil {
			return err
		} else {
			content += fmt.Sprintln(line)
		}
	}

	dst := filepath.Join(where, "/etc/shadow")
	mode := fs.FileMode(0o600)
	return ensureWriteFile(dst, []byte(content), mode, 0, 0)
}

func writeGroups(groups []passwd.Group, where string) error {
	log.WithFields(log.Fields{
		"groups": groups,
		"where":  where,
	}).Trace("start")
	defer log.Trace("end")

	if len(groups) == 0 {
		return nil
	}

	content := ""
	for _, g := range groups {
		if line, err := g.Marshal(); err != nil {
			return err
		} else {
			content += fmt.Sprintln(line)
		}
	}

	dst := filepath.Join(where, "/etc/group")
	mode := fs.FileMode(0o644)
	return ensureWriteFile(dst, []byte(content), mode, 0, 0)
}

func writeUsers(users []passwd.User, where string) error {
	log.WithFields(log.Fields{
		"users": users,
		"where": where,
	}).Trace("start")
	defer log.Trace("end")

	if len(users) == 0 {
		return nil
	}

	content := ""
	for _, user := range users {
		if line, err := user.Marshal(); err != nil {
			return err
		} else {
			content += fmt.Sprintln(line)
		}
	}

	dst := filepath.Join(where, "/etc/passwd")
	mode := fs.FileMode(0o644)
	return ensureWriteFile(dst, []byte(content), mode, 0, 0)
}

func writeLinks(links []Link, where string) error {
	log.WithFields(log.Fields{
		"links": links,
		"where": where,
	}).Trace("start")
	defer log.Trace("end")

	for _, l := range links {
		dst := filepath.Join(where, l.Path)
		if err := common.CreateSymlink(
			dst,
			l.Target,
			l.Overwrite,
			l.UID,
			l.GID,
			l.Hard,
		); err != nil {
			return err
		}
	}

	return nil
}

func writeDirectories(directories []Directory, where string) error {
	log.WithFields(log.Fields{
		"directories": directories,
		"where":       where,
	}).Trace("start")
	defer log.Trace("end")

	for _, d := range directories {
		dst := filepath.Join(where, d.Path)
		if err := os.MkdirAll(dst, d.Mode); err != nil {
			return err
		}
		if err := os.Chown(dst, d.UID, d.UID); err != nil {
			return err
		}
	}

	return nil
}

func writeFiles(files []File, where string) error {
	log.WithFields(log.Fields{
		"files": files,
		"where": where,
	}).Trace("start")
	defer log.Trace("end")

	for _, f := range files {
		dst := filepath.Join(where, f.Filename)

		// If overwrite == false and file exists, skip it.
		if !f.Overwrite && common.IsPathExists(dst) {
			return nil
		}

		if err := ensureWriteFile(dst, f.Content, f.Mode, f.UID, f.GID); err != nil {
			return err
		}
	}

	return nil
}

// Write writes into the directory `where`:
//   - Links
//   - Directories
//   - Files
//   - Shadows
//   - Groups
//   - Users
func (sr *Sysroot) Write(where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Trace("start")
	defer log.Trace("end")

	if !common.IsDir(where) {
		return fmt.Errorf("%q is not a directory", where)
	}

	if err := writeMounts(sr.Mounts, where); err != nil {
		log.WithError(err).Error("cannot write mounts")
		return err
	}
	if err := writeLinks(sr.Links, where); err != nil {
		log.WithError(err).Error("cannot write links")
		return err
	}
	if err := writeDirectories(sr.Directories, where); err != nil {
		log.WithError(err).Error("cannot write directories")
		return err
	}
	if err := writeFiles(sr.Files, where); err != nil {
		log.WithError(err).Error("cannot write files")
		return err
	}
	if err := writeShadows(sr.Shadows, where); err != nil {
		log.WithError(err).Error("cannot write " + filepath.Join(where, "/etc/shadow"))
		return err
	}
	if err := writeGroups(sr.Groups, where); err != nil {
		log.WithError(err).Error("cannot write " + filepath.Join(where, "/etc/groups"))
		return err
	}
	if err := writeUsers(sr.Users, where); err != nil {
		log.WithError(err).Error("cannot write " + filepath.Join(where, "/etc/passwd"))
		return err
	}

	return nil
}
