package sysroot

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/coreos/go-systemd/v22/unit"
	"github.com/jlsalvador/simplek8s/pkg/common"
	"github.com/jlsalvador/simplek8s/pkg/linux/mount"
	"github.com/jlsalvador/simplek8s/pkg/linux/passwd"
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
	}).Debug("start")
	defer log.Debug("end")

	path := filepath.Dir(filename)
	if err := os.MkdirAll(path, 0755); err != nil {
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

func writeMounts(mounts []Mount, where string) error {
	log.WithFields(log.Fields{
		"mounts": mounts,
		"where":  where,
	}).Debug("start")
	defer log.Debug("end")

	// We will use systemd units to mount these mountpoints.

	// These units must be saved into /run/systemd/system/.
	// So we need to mount `${where}/run` before.
	runMnt := mount.Mountpoints.Run
	runMnt.Target = filepath.Join(where, runMnt.Target)
	if err := mount.Mount(runMnt); err != nil {
		log.WithError(err).Error("can not mount " + runMnt.Target)
		return err
	}

	// Create systemd mount unit for each mountpoint into `${where}/run/systemd/system/`.
	for _, m := range mounts {
		var dst string
		var content string

		if m.Where == "/var" {
			dst = filepath.Join(where, "/run/systemd/system/var.mount.d/drop-in.conf")
			content = fmt.Sprintf(`[Mount]
What=%s
Where=%s
Type=%s
Options=%s
`, m.What, m.Where, m.Type, m.Options)
		} else {
			escapedName := unit.UnitNameEscape(m.Where)
			unitName := fmt.Sprintf("%s.mount", escapedName)
			dst = filepath.Join(where, "/run/systemd/system/", unitName)
			content = fmt.Sprintf(`[Unit]
Description=%s mountpoint
DefaultDependencies=no
Conflicts=umount.target
Before=local-fs.target
Before=umount.target
After=network.target

[Mount]
What=%s
Where=%s
Type=%s
Options=%s
`, escapedName, m.What, m.Where, m.Type, m.Options)
		}

		if err := ensureWriteFile(dst, []byte(content), 0644, 0, 0); err != nil {
			log.WithError(err).Error("can not write mount unit " + dst)
			return err
		}
	}

	return nil
}

func writeShadows(shadows []passwd.Shadow, where string) error {
	log.WithFields(log.Fields{
		"shadows": shadows,
		"where":   where,
	}).Debug("start")
	defer log.Debug("end")

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
	mode := fs.FileMode(0600)
	return ensureWriteFile(dst, []byte(content), mode, 0, 0)
}

func writeGroups(groups []passwd.Group, where string) error {
	log.WithFields(log.Fields{
		"groups": groups,
		"where":  where,
	}).Debug("start")
	defer log.Debug("end")

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
	mode := fs.FileMode(0644)
	return ensureWriteFile(dst, []byte(content), mode, 0, 0)
}

func writeUsers(users []passwd.User, where string) error {
	log.WithFields(log.Fields{
		"users": users,
		"where": where,
	}).Debug("start")
	defer log.Debug("end")

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
	mode := fs.FileMode(0644)
	return ensureWriteFile(dst, []byte(content), mode, 0, 0)
}

func writeLinks(links []Link, where string) error {
	log.WithFields(log.Fields{
		"links": links,
		"where": where,
	}).Debug("start")
	defer log.Debug("end")

	for _, l := range links {
		dst := filepath.Join(where, l.Path)
		if err := common.CreateSymlink(
			dst,
			l.Target,
			l.Overwrite,
			l.Uid,
			l.Gid,
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
	}).Debug("start")
	defer log.Debug("end")

	for _, d := range directories {
		dst := filepath.Join(where, d.Path)
		if err := os.MkdirAll(dst, d.Mode); err != nil {
			return err
		}
		if err := os.Chown(dst, d.Uid, d.Uid); err != nil {
			return err
		}
	}

	return nil
}

func writeFiles(files []File, where string) error {
	log.WithFields(log.Fields{
		"files": files,
		"where": where,
	}).Debug("start")
	defer log.Debug("end")

	for _, f := range files {
		dst := filepath.Join(where, f.Filename)

		// If overwrite == false and file exists, skip it.
		if !f.Overwrite && common.CheckFileExists(dst) {
			return nil
		}

		if err := ensureWriteFile(dst, f.Content, f.Mode, f.Uid, f.Gid); err != nil {
			return err
		}
	}

	return nil
}

// Writes into the directory `where`:
//   - Links
//   - Directories
//   - Files
//   - Shadows
//   - Groups
//   - Users
//   - sshAuthorizedKeys of each User if apply
func (sr *Sysroot) Write(where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Debug("start")
	log.Debug("end")

	if !common.IsDir(where) {
		return fmt.Errorf("%q is not a directory", where)
	}

	if err := writeMounts(sr.Mounts, where); err != nil {
		log.WithError(err).Error("can not write mounts")
		return err
	}
	if err := writeLinks(sr.Links, where); err != nil {
		log.WithError(err).Error("can not write links")
		return err
	}
	if err := writeDirectories(sr.Directories, where); err != nil {
		log.WithError(err).Error("can not write directories")
		return err
	}
	if err := writeFiles(sr.Files, where); err != nil {
		log.WithError(err).Error("can not write files")
		return err
	}
	if err := writeShadows(sr.Shadows, where); err != nil {
		log.WithError(err).Error("can not write " + filepath.Join(where, "/etc/shadow"))
		return err
	}
	if err := writeGroups(sr.Groups, where); err != nil {
		log.WithError(err).Error("can not write " + filepath.Join(where, "/etc/groups"))
		return err
	}
	if err := writeUsers(sr.Users, where); err != nil {
		log.WithError(err).Error("can not write " + filepath.Join(where, "/etc/passwd"))
		return err
	}

	//TODO: Write sshAuthorizedKeys of each User

	return nil
}
