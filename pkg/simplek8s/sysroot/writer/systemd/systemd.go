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

// Package systemd populates a new root that will boot systemd.
package systemd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/mount"
	"simplek8s/pkg/linux/passwd"
	"simplek8s/pkg/simplek8s/generateshadow"
	"simplek8s/pkg/simplek8s/sysroot"
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

// Because of the next requirements, we are going to mount all
// mountpoints in the initrd stage:
//
//   - /sysroot/etc must be mounted by initrd, because systemd requires rootfs
//     or /etc to be mounted before exec /sbin/init.
//   - /sysroot/etc requires /sysroot/var to be mounted, because /sysroot/etc
//     binds to /sysroot/var/etc by default.
//   - simplek8s.yaml could defines more complex environments to mount /etc.
func writeMounts(mounts []mount.MountPoint, where string) error {
	log.WithFields(log.Fields{
		"mounts": mounts,
		"where":  where,
	}).Trace("start")
	defer log.Trace("end")

	// Ensure that /dev is populated by udev.
	// This step is required to search disks by label.
	if err := udev.PopulateDev(); err != nil {
		return fmt.Errorf("cannot populate /dev by udev: %w", err)
	}

	for _, m := range mounts {
		if m.Flags&mount.MountFlagBind != 0 {
			// Fix source for bind mounts.
			m.Source = filepath.Join(where, m.Source)

			// Ensure the source directory exists for bind mounts.
			if err := os.MkdirAll(m.Source, m.Chmod); err != nil {
				return fmt.Errorf("cannot create directory %s: %w", m.Target, err)
			}
		}

		if strings.HasPrefix(m.Target, "/") {
			// Fix target for /sysroot mounts.
			// Normally, all mountpoints must be relative to /sysroot.
			m.Target = filepath.Join(where, m.Target)
		}

		log.WithField("mountpoint", m).Trace()
		if err := mount.Mount(m); err != nil {
			return fmt.Errorf("cannot mount %s in %s: %w", m.Source, m.Target, err)
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

	for _, s := range shadows {
		dst := filepath.Join(where, fmt.Sprintf("/run/credstore/passwd.hashed-password.%s", s.Name))
		if err := ensureWriteFile(dst, fmt.Appendf(nil, "%s", s.Password), 0o400, 0, 0); err != nil {
			return fmt.Errorf("cannot write credential %s: %w", dst, err)
		}

		// Drop-in for systemd-sysusers.service to ImportCredential.
		// The "root" user is already defined by the service itself.
		if s.Name != "root" {
			dst = filepath.Join(where, fmt.Sprintf("/run/systemd/system/systemd-sysusers.service.d/10-import-credential-%s.conf", s.Name))
			if err := ensureWriteFile(dst, fmt.Appendf(nil, "[Service]\nImportCredential=passwd.hashed-password.%s\n", s.Name), 0o644, 0, 0); err != nil {
				return fmt.Errorf("cannot write drop-in %s: %w", dst, err)
			}
		}
	}

	return nil
}

func writeGroups(groups []passwd.Group, where string) error {
	log.WithFields(log.Fields{
		"groups": groups,
		"where":  where,
	}).Trace("start")
	defer log.Trace("end")

	for _, g := range groups {
		dst := filepath.Join(where, fmt.Sprintf("/run/sysusers.d/group-%s.conf", g.Name))
		data := fmt.Appendf(nil, "g %s %d\n", g.Name, g.GID)
		for _, u := range g.UserList {
			data = fmt.Appendf(data, "m %s %s\n", u, g.Name)
		}
		if err := ensureWriteFile(dst, data, 0o644, 0, 0); err != nil {
			return fmt.Errorf("cannot write group %s: %w", dst, err)
		}
	}

	return nil
}

func writeUsers(users []passwd.User, where string) error {
	log.WithFields(log.Fields{
		"users": users,
		"where": where,
	}).Trace("start")
	defer log.Trace("end")

	for _, u := range users {
		// Create sysusers.d configuration file to create the user.
		dst := filepath.Join(where, fmt.Sprintf("/run/sysusers.d/user-%s.conf", u.Name))
		data := fmt.Appendf(nil, "u %s %d:%d \"%s\" %s %s\n", u.Name, u.UID, u.GID, u.Gecos, u.Home, u.Shell)
		if err := ensureWriteFile(dst, data, 0x644, 0, 0); err != nil {
			return fmt.Errorf("cannot write sysusers.d configuration %s: %w", dst, err)
		}

		// Create tmpfiles.d configuration file to create user's home directory.
		dst = filepath.Join(where, fmt.Sprintf("/run/tmpfiles.d/home-%s.conf", u.Name))
		data = fmt.Appendf(nil, "d %s 0750 %d %d\n", u.Home, u.UID, u.GID)
		if err := ensureWriteFile(dst, data, 0x644, 0, 0); err != nil {
			return fmt.Errorf("cannot write tmpfiles configuration %s: %w", dst, err)
		}
	}

	return nil
}

func writeLinks(links []sysroot.Link, where string) error {
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

func writeDirectories(directories []sysroot.Directory, where string) error {
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

func writeFiles(files []sysroot.File, where string) error {
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

func writeNonPersistentSession(sr *sysroot.Sysroot) error {
	// Do nothing if simplek8s.yaml is found.
	if slices.IndexFunc(sr.Files, func(f sysroot.File) bool {
		return f.Filename == "/run/simplek8s/simplek8s.yaml"
	}) >= 0 {
		return nil
	}

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

	return nil
}

// Write populates a directory `where` with the contents of a Sysroot using
// Systemd toolsets as tmpfiles and sysusers.
//
// Mountpoints will be mounted without using systemd because systemd does not
// reloads /etc.
func Write(sr *sysroot.Sysroot, where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Trace("start")
	defer log.Trace("end")

	if !common.IsDir(where) {
		return fmt.Errorf("%q is not a directory", where)
	}

	if err := writeNonPersistentSession(sr); err != nil {
		log.WithError(err).Error("cannot write non persistent session")
		return err
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
