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

package bootstrap

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/simplek8s/simplek8s-init/pkg/common"
	"github.com/simplek8s/simplek8s-init/pkg/linux/mount"
	"github.com/simplek8s/simplek8s-init/pkg/linux/passwd"
	sr "github.com/simplek8s/simplek8s-init/pkg/simplek8s/sysroot"

	log "github.com/sirupsen/logrus"
	"go.openly.dev/pointy"
)

// appendWarning records a non-fatal content error: the offending entry is
// skipped, boot continues, and the message is persisted for the issue banner.
func appendWarning(warnings *[]string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Warn(msg)
	*warnings = append(*warnings, msg)
}

func isAbsPath(p string) bool {
	return p != "" && filepath.IsAbs(p)
}

// parseFileMode parses a permission string as octal (ex: "0755", "755", "0644").
// It accepts an optional "0o"/"0O" prefix and surrounding whitespace.
func parseFileMode(s string, def fs.FileMode) (fs.FileMode, error) {
	t := strings.TrimSpace(s)
	t = strings.TrimPrefix(t, "0o")
	t = strings.TrimPrefix(t, "0O")
	if t == "" {
		return def, fmt.Errorf("empty file mode")
	}
	v, err := strconv.ParseUint(t, 8, 32)
	if err != nil {
		return def, err
	}
	return fs.FileMode(v), nil
}

func getNextGID(groups []passwd.Group, isSystem bool) int {
	log.WithFields(log.Fields{
		"groups":   groups,
		"isSystem": isSystem,
	}).Trace("start")
	defer log.Trace("end")

	current := 999
	if !isSystem {
		current = 1000
	}

	ids := []int{}
	for _, group := range groups {
		if (isSystem && group.GID >= 1000) || (!isSystem && group.GID < 1000) {
			continue
		}

		ids = append(ids, int(group.GID))
	}

	for slices.Contains(ids, current) {
		if isSystem {
			// When it is a system group, goes backward from 999 to 1.
			current--
			if current == 0 {
				// If system groups are exausted, always returns 999.
				return 999
			}
		} else {
			// When it is a normal group, goes forward from 1000 to 65533.
			current++
			if current == 65534 {
				// Skip 65534 and 65535:
				// doc: https://systemd.io/UIDS-GIDS/#special-distribution-uid-ranges
				current = 65536
			}
		}
	}
	return current
}

func getNextUID(users []passwd.User, isSystem bool) int {
	log.WithFields(log.Fields{
		"users":    users,
		"isSystem": isSystem,
	}).Trace("start")
	defer log.Trace("end")

	current := 0
	if !isSystem {
		current = 1000
	}

	ids := []int{}
	for _, user := range users {
		if (isSystem && user.UID >= 1000) || (!isSystem && user.UID < 1000) {
			continue
		}

		ids = append(ids, int(user.UID))
	}

	for slices.Contains(ids, current) {
		current++
	}
	return current
}

func getUIDByName(users []passwd.User, name string) *int {
	log.Trace("start")
	defer log.Trace("end")

	for _, user := range users {
		if user.Name == name {
			return &user.UID
		}
	}
	log.WithFields(log.Fields{
		"name":  name,
		"users": users,
	}).Error("cannot find uid by name")
	return nil
}

func getGIDByName(groups []passwd.Group, name string) *int {
	log.Trace("start")
	defer log.Trace("end")

	for _, group := range groups {
		if group.Name == name {
			return &group.GID
		}
	}
	log.WithFields(log.Fields{
		"name":   name,
		"groups": groups,
	}).Error("cannot find gid by name")
	return nil
}

func getUIDGIDFromString(permission string, groups []passwd.Group, users []passwd.User) (int, int) {
	log.Trace("start")
	defer log.Trace("end")

	uid := 0
	gid := 0

	ids := strings.Split(permission, ":")
	if value, err := strconv.Atoi(ids[0]); err == nil {
		uid = value
	} else {
		if value := getUIDByName(users, ids[0]); value != nil {
			uid = *value
		}
	}
	if len(ids) > 1 {
		if value, err := strconv.Atoi(ids[1]); err == nil {
			gid = value
		} else {
			if value := getGIDByName(groups, ids[1]); value != nil {
				gid = *value
			}
		}
	}

	return uid, gid
}

func updateOrAppendFile(files []sr.File, file sr.File) []sr.File {
	log.Trace("start")
	defer log.Trace("end")

	return common.UpdateOrAppend(files, file, func(a, b sr.File) bool {
		return a.Filename == b.Filename
	})
}

// feedByBootstrapConfigGroupsWithGIDs appends into sysroot.Groups groups with
// proper GIDs.
//
// Groups without GIDs or empty names will be skipped with a warning.
func feedByBootstrapConfigGroupsWithGIDs(config Config, sysroot *sr.Sysroot, warnings *[]string) {
	// First, create user defined groups with proper GIDs.
	for _, group := range config.Groups {
		// Skip empty group names.
		if group.Name == "" {
			appendWarning(warnings, "skip group with empty name")
			continue
		}

		// Skip groups without GIDs.
		if group.GID == nil {
			continue
		}

		group := passwd.NewGroup(passwd.Group{
			Name: group.Name,
			GID:  *group.GID,
		})
		sysroot.Groups = common.UpdateOrAppend(sysroot.Groups, group, func(a, b passwd.Group) bool {
			return a.GID == b.GID
		})
	}
}

// feedByBootstrapConfigGroupsFromUsers appends user groups that are not
// already defined.
//
// Example: create the "root" group for the "root" user.
func feedByBootstrapConfigGroupsFromUsers(config Config, sysroot *sr.Sysroot, warnings *[]string) {
	for _, user := range config.Users {
		// Skip empty usernames.
		if user.Name == "" {
			appendWarning(warnings, "skip user with empty name")
			continue
		}

		// Skip user if there is already a group with the same user.GID.
		if user.GID != nil && slices.IndexFunc(sysroot.Groups, func(group passwd.Group) bool {
			return group.GID == *user.GID
		}) >= 0 {
			continue
		}

		var gid int
		if user.Name == "root" {
			gid = 0
		} else {
			isSystem := common.Get(user.System, false)
			gid = common.Get(user.GID, getNextGID(sysroot.Groups, isSystem))
		}

		sysroot.Groups = common.UpdateOrAppend(sysroot.Groups, passwd.NewGroup(passwd.Group{
			Name: user.Name,
			GID:  gid,
		}), func(a, b passwd.Group) bool {
			return a.GID == b.GID
		})
	}
}

// feedByBootstrapConfigGroupsWithoutGIDs appends into sysroot.Groups user
// defined groups without GIDs.
//
// Groups with empty names will be skipped with a warning.
func feedByBootstrapConfigGroupsWithoutGIDs(config Config, sysroot *sr.Sysroot, warnings *[]string) {
	for _, group := range config.Groups {
		// Skip empty group names.
		if group.Name == "" {
			appendWarning(warnings, "skip group with empty name")
			continue
		}

		// Skip groups with proper GIDs.
		if group.GID != nil {
			continue
		}

		isSystem := common.Get(group.System, false)
		gid := getNextGID(sysroot.Groups, isSystem)
		g := passwd.NewGroup(passwd.Group{
			Name: group.Name,
			GID:  gid,
		})
		sysroot.Groups = common.UpdateOrAppend(sysroot.Groups, g, func(a, b passwd.Group) bool {
			return a.GID == b.GID
		})
	}
}

func feedByBootstrapConfigGroups(sysroot *sr.Sysroot, config Config, warnings *[]string) {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	feedByBootstrapConfigGroupsWithGIDs(config, sysroot, warnings)
	feedByBootstrapConfigGroupsFromUsers(config, sysroot, warnings)
	feedByBootstrapConfigGroupsWithoutGIDs(config, sysroot, warnings)

	// Sort groups by its GIDs.
	sort.Slice(sysroot.Groups, func(i, j int) bool {
		return sysroot.Groups[i].GID < sysroot.Groups[j].GID
	})
}

func determineHomeAndShell(user User) (string, string) {
	isSystem := common.Get(user.System, false)

	switch {
	case user.Name == "root":
		return "/root", "/usr/bin/sh"

	case !isSystem:
		return fmt.Sprintf("/home/%s", user.Name), "/usr/bin/sh"

	default:
		return "/", "/usr/sbin/nologin"
	}
}

func determineUIDGID(sysroot *sr.Sysroot, user User) (int, int) {
	isSystem := common.Get(user.System, false)

	switch {
	case user.Name == "root":
		gid := getGIDByName(sysroot.Groups, "root")
		if gid == nil {
			group := passwd.NewGroup(passwd.Group{
				Name:     "root",
				Password: "",
				GID:      0,
				UserList: []string{"root"},
			})
			sysroot.Groups = common.UpdateOrAppend(sysroot.Groups, group, func(a, b passwd.Group) bool {
				return a.GID == b.GID
			})
			gid = pointy.Int(0)
		}
		return common.Get(user.UID, 0), common.Get(user.GID, *gid)

	default:
		gid := common.Get(user.GID, -1)
		if gid == -1 {
			// Find a group with the same user.Name.
			gid = common.Get(getGIDByName(sysroot.Groups, user.Name), -1)
		}
		if gid == -1 {
			// Create a new group with the same user.Name.
			group := passwd.NewGroup(passwd.Group{
				Name:     user.Name,
				Password: "",
				GID:      getNextGID(sysroot.Groups, isSystem),
				UserList: []string{user.Name},
			})
			sysroot.Groups = append(sysroot.Groups, group)
			gid = group.GID
		}
		return common.Get(user.UID, getNextUID(sysroot.Users, isSystem)), gid
	}
}

func determineGecos(user User) string {
	if user.Name == "root" && user.Gecos == nil {
		return "Super User"
	}
	return common.Get(user.Gecos, "")
}

// Users could feeds:
//   - groups: user group.
//   - directories: home users.
//   - files: mostly ~/.ssh/authorized_keys.
//
// Invalid entries are skipped with a warning so a single typo cannot
// take the whole boot down.
func feedByBootstrapConfigUsers(sysroot *sr.Sysroot, config Config, warnings *[]string) {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	for _, user := range config.Users {
		// Skip empty usernames.
		if user.Name == "" {
			appendWarning(warnings, "skip user with empty name")
			continue
		}

		home, shell := determineHomeAndShell(user)
		uid, gid := determineUIDGID(sysroot, user)
		gecos := determineGecos(user)

		// Add or update user to sysroot.Users.
		sysroot.Users = common.UpdateOrAppend(sysroot.Users, passwd.NewUser(passwd.User{
			Name:     user.Name,
			Password: "x",
			UID:      uid,
			GID:      gid,
			Gecos:    gecos,
			Home:     home,
			Shell:    shell,
		}), func(a, b passwd.User) bool {
			return a.Name == b.Name
		})

		// Update shadow instance.
		if user.PasswordHash != nil {
			passwdShadow := passwd.NewShadow(passwd.Shadow{
				Name:     user.Name,
				Password: *user.PasswordHash,
			})
			sysroot.Shadows = common.UpdateOrAppend(sysroot.Shadows, passwdShadow, func(a, b passwd.Shadow) bool {
				return a.Name == b.Name
			})
		} else if len(user.SSHAuthorizedKeys) > 0 {
			// A user with SSH keys but no password hash must not
			// stay locked: systemd-sysusers defaults such users to
			// "!unprovisioned", and sshd denies even public key
			// authentication to locked ("!...") accounts. Set the
			// impossible-but-unlocked "*" password so keys work
			// while password login stays impossible.
			passwdShadow := passwd.NewShadow(passwd.Shadow{
				Name:     user.Name,
				Password: "*",
			})
			sysroot.Shadows = common.UpdateOrAppend(sysroot.Shadows, passwdShadow, func(a, b passwd.Shadow) bool {
				return a.Name == b.Name
			})
		}

		// Include own user name as group.
		userGroups := common.UpdateOrAppend(user.Groups, user.Name, func(a, b string) bool {
			return a == b
		})

		// Update groups instances user list.
		for _, groupName := range userGroups {
			i := slices.IndexFunc(sysroot.Groups, func(group passwd.Group) bool {
				return group.Name == groupName
			})
			if i < 0 {
				appendWarning(warnings, "skip unknown group %q for user %q", groupName, user.Name)
				continue
			}
			// Try to update an already defined group.
			// Avoid duplicate memberships on re-feeds.
			if !slices.Contains(sysroot.Groups[i].UserList, user.Name) {
				sysroot.Groups[i].UserList = append(sysroot.Groups[i].UserList, user.Name)
			}
		}

		// Create home directory.
		sysroot.Directories = common.UpdateOrAppend(sysroot.Directories, sr.Directory{
			Overwrite: false,
			Path:      home,
			Mode:      0o750,
			UID:       uid,
			GID:       gid,
		}, func(a, b sr.Directory) bool {
			return a.Path == b.Path
		})

		// Set the SSH Authorized keys.
		if len(user.SSHAuthorizedKeys) > 0 {
			content := []byte{}
			for _, sshPublicKey := range user.SSHAuthorizedKeys {
				content = fmt.Appendln(content, sshPublicKey)
			}

			if len(content) > 0 {
				sysroot.Directories = common.UpdateOrAppend(sysroot.Directories, sr.Directory{
					Overwrite: false,
					Path:      home + "/.ssh",
					Mode:      0o700,
					UID:       uid,
					GID:       gid,
				}, func(a, b sr.Directory) bool {
					return a.Path == b.Path
				})
				sysroot.Files = common.UpdateOrAppend(sysroot.Files, sr.File{
					Overwrite: false,
					Filename:  home + "/.ssh/authorized_keys",
					Content:   content,
					Mode:      0o600,
					UID:       uid,
					GID:       gid,
				}, func(a, b sr.File) bool {
					return a.Filename == b.Filename
				})
			}
		}
	}

}

func feedByBootstrapConfigLinks(sysroot *sr.Sysroot, config Config, warnings *[]string) {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	if config.Storage == nil || config.Storage.Links == nil {
		return
	}

	for _, link := range config.Storage.Links {
		if !isAbsPath(link.Path) {
			appendWarning(warnings, "skip link with non-absolute path %q", link.Path)
			continue
		}
		if link.Target == "" {
			appendWarning(warnings, "skip link %q with empty target", link.Path)
			continue
		}
		// By default, UID and GID from own process.
		// Reset on each iteration so a previous `owner` does not leak
		// into the next entry when it has no explicit owner.
		uid, gid := syscall.Getuid(), syscall.Getgid()
		if link.Owner != nil {
			uid, gid = getUIDGIDFromString(*link.Owner, sysroot.Groups, sysroot.Users)
		}

		isOverwrite := common.Get(link.Overwrite, false)
		isHard := common.Get(link.Hard, false)

		sysroot.Links = common.UpdateOrAppend(sysroot.Links, sr.Link{
			Overwrite: isOverwrite,
			Path:      link.Path,
			Target:    link.Target,
			UID:       uid,
			GID:       gid,
			Hard:      isHard,
		}, func(a, b sr.Link) bool {
			return a.Path == b.Path
		})
	}

}

func feedByBootstrapConfigDirectories(sysroot *sr.Sysroot, config Config, warnings *[]string) {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	if config.Storage == nil || config.Storage.Directories == nil {
		return
	}

	for _, directory := range config.Storage.Directories {
		if !isAbsPath(directory.Path) {
			appendWarning(warnings, "skip directory with non-absolute path %q", directory.Path)
			continue
		}
		// By default, UID and GID from own process.
		// Reset on each iteration so a previous `owner` does not leak
		// into the next entry when it has no explicit owner.
		uid, gid := syscall.Getuid(), syscall.Getgid()
		isOverwrite := common.Get(directory.Overwrite, false)

		if directory.Owner != nil {
			uid, gid = getUIDGIDFromString(*directory.Owner, sysroot.Groups, sysroot.Users)
		}

		mode := fs.FileMode(0o775)
		if directory.Permissions != nil {
			var err error
			if mode, err = parseFileMode(*directory.Permissions, mode); err != nil {
				appendWarning(warnings, "skip directory %q: invalid permissions %q", directory.Path, *directory.Permissions)
				continue
			}
		}

		sysroot.Directories = common.UpdateOrAppend(sysroot.Directories, sr.Directory{
			Overwrite: isOverwrite,
			Path:      directory.Path,
			Mode:      mode,
			UID:       uid,
			GID:       gid,
		}, func(a, b sr.Directory) bool {
			return a.Path == b.Path
		})
	}

}

// TODO:
//   - Fetch from HTTP when encoding is http
//   - Fetch from HTTPS when encoding is https
func getBytesFromEncoding(encoding *string, content *string) ([]byte, error) {
	if content == nil {
		return []byte{}, nil
	}
	if encoding != nil && *encoding == "b64" {
		c, err := base64.StdEncoding.DecodeString(*content)
		if err != nil {
			return nil, fmt.Errorf("cannot decode base64 content: %w", err)
		}
		return c, nil
	}
	return []byte(*content), nil
}

func feedByBootstrapConfigFiles(sysroot *sr.Sysroot, config Config, warnings *[]string) {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	if config.Storage == nil || config.Storage.Files == nil {
		return
	}

	for _, file := range config.Storage.Files {
		filename := file.Path
		if !isAbsPath(filename) {
			appendWarning(warnings, "skip file with non-absolute path %q", filename)
			continue
		}
		isOverwrite := common.Get(file.Overwrite, false)

		// By default, UID and GID from own process.
		// Reset on each iteration so a previous `owner` does not leak
		// into the next entry when it has no explicit owner.
		uid, gid := syscall.Getuid(), syscall.Getgid()
		if file.Owner != nil {
			uid, gid = getUIDGIDFromString(*file.Owner, sysroot.Groups, sysroot.Users)
		}

		mode := fs.FileMode(0o664)
		if file.Permissions != nil {
			var err error
			if mode, err = parseFileMode(*file.Permissions, mode); err != nil {
				appendWarning(warnings, "skip file %q: invalid permissions %q", filename, *file.Permissions)
				continue
			}
		}

		content, err := getBytesFromEncoding(file.Encoding, file.Content)
		if err != nil {
			appendWarning(warnings, "skip file %q: %v", filename, err)
			continue
		}

		sysroot.Files = updateOrAppendFile(sysroot.Files, sr.File{
			Overwrite: isOverwrite,
			Filename:  filename,
			Content:   content,
			Mode:      mode,
			UID:       uid,
			GID:       gid,
		})
	}

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

func feedByBootstrapConfigMounts(sysroot *sr.Sysroot, config Config, warnings *[]string) {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	if config.Storage == nil || config.Storage.Mounts == nil {
		return
	}

	for _, m := range config.Storage.Mounts {
		if m.What == "" || !isAbsPath(m.Where) {
			appendWarning(warnings, "skip mount with invalid what %q where %q", m.What, m.Where)
			continue
		}
		flags, data := resolveFlags(strings.Split(common.Get(m.Options, ""), ","))
		sysroot.Mounts = common.UpdateOrAppend(sysroot.Mounts, mount.MountPoint{
			Target: m.Where,
			Chmod:  0o755,
			Source: m.What,
			Fstype: common.Get(m.Type, ""),
			Flags:  flags,
			Data:   data,
		}, func(a, b mount.MountPoint) bool {
			return a.Target == b.Target
		})
	}

}

func markAsFeededByBootstrapConfig(sysroot *sr.Sysroot, config Config) {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	sysroot.Files = append(sysroot.Files, sr.File{
		Overwrite: true,
		Filename:  "/run/simplek8s/simplek8s.yaml",
		Content:   []byte(config.String()),
		Mode:      0o400,
		UID:       0,
		GID:       0,
	})
}

func writeApplyWarnings(sysroot *sr.Sysroot, warnings []string) {
	if len(warnings) == 0 {
		return
	}
	body := strings.Join(warnings, "\n") + "\n"
	sysroot.Files = common.UpdateOrAppend(sysroot.Files, sr.File{
		Overwrite: true,
		Filename:  "/run/simplek8s/apply-warnings.log",
		Content:   []byte(body),
		Mode:      0o400,
		UID:       0,
		GID:       0,
	}, func(a, b sr.File) bool {
		return a.Filename == b.Filename
	})
	banner := fmt.Sprintf("\n\\e{yellow}simplek8s.yaml applied with %d warning(s). See /run/simplek8s/apply-warnings.log\\e{reset}\n", len(warnings))
	sysroot.Files = common.UpdateOrAppend(sysroot.Files, sr.File{
		Overwrite: true,
		Filename:  "/run/issue.d/81-apply-warnings.issue",
		Content:   []byte(banner),
		Mode:      0o644,
		UID:       0,
		GID:       0,
	}, func(a, b sr.File) bool {
		return a.Filename == b.Filename
	})
}

func FeedSysrootByBootstrapConfig(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	var warnings []string
	feedByBootstrapConfigGroups(sysroot, config, &warnings)
	feedByBootstrapConfigUsers(sysroot, config, &warnings)
	feedByBootstrapConfigLinks(sysroot, config, &warnings)
	feedByBootstrapConfigDirectories(sysroot, config, &warnings)
	feedByBootstrapConfigFiles(sysroot, config, &warnings)
	feedByBootstrapConfigMounts(sysroot, config, &warnings)
	markAsFeededByBootstrapConfig(sysroot, config)
	writeApplyWarnings(sysroot, warnings)
	return nil
}
