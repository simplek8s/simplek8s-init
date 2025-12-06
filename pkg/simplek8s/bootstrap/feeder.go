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
// Groups without GIDs or empty names will be skipped.
func feedByBootstrapConfigGroupsWithGIDs(config Config, sysroot *sr.Sysroot) {
	// First, create user defined groups with proper GIDs.
	for _, group := range config.Groups {
		// Skip empty group names.
		if group.Name == "" {
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
func feedByBootstrapConfigGroupsFromUsers(config Config, sysroot *sr.Sysroot) {
	for _, user := range config.Users {
		// Skip empty usernames.
		if user.Name == "" {
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
// Groups with empty names will be skipped.
func feedByBootstrapConfigGroupsWithoutGIDs(config Config, sysroot *sr.Sysroot) {
	for _, group := range config.Groups {
		// Skip empty group names.
		if group.Name == "" {
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

func feedByBootstrapConfigGroups(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	feedByBootstrapConfigGroupsWithGIDs(config, sysroot)
	feedByBootstrapConfigGroupsFromUsers(config, sysroot)
	feedByBootstrapConfigGroupsWithoutGIDs(config, sysroot)

	// Sort groups by its GIDs.
	sort.Slice(sysroot.Groups, func(i, j int) bool {
		return sysroot.Groups[i].GID < sysroot.Groups[j].GID
	})

	return nil
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
func feedByBootstrapConfigUsers(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	for _, user := range config.Users {
		// Skip empty usernames.
		if user.Name == "" {
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
				return fmt.Errorf("cannot find group %s", groupName)
			}
			// Try to update an already defined group.
			sysroot.Groups[i].UserList = append(sysroot.Groups[i].UserList, user.Name)
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

	return nil
}

func feedByBootstrapConfigLinks(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	if config.Storage == nil || config.Storage.Links == nil {
		return nil
	}

	// By default, UID and GID from own process.
	uid, gid := syscall.Getuid(), syscall.Getgid()

	for _, link := range config.Storage.Links {
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

	return nil
}

func feedByBootstrapConfigDirectories(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	if config.Storage == nil || config.Storage.Directories == nil {
		return nil
	}

	// By default, UID and GID from own process.
	uid, gid := syscall.Getuid(), syscall.Getgid()

	for _, directory := range config.Storage.Directories {
		isOverwrite := common.Get(directory.Overwrite, false)

		if directory.Owner != nil {
			uid, gid = getUIDGIDFromString(*directory.Owner, sysroot.Groups, sysroot.Users)
		}

		var mode fs.FileMode = 0o775
		if directory.Permissions != nil {
			if valueAsInt, err := strconv.Atoi(*directory.Permissions); err != nil {
				return err
			} else {
				mode = fs.FileMode(valueAsInt)
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

	return nil
}

// TODO:
//   - Fetch from HTTP when encoding is http
//   - Fetch from HTTPS when encoding is https
func getBytesFromEncoding(encoding *string, content *string) []byte {
	if content != nil {
		if encoding != nil && *encoding == "b64" {
			if c, err := base64.StdEncoding.DecodeString(*content); err == nil {
				return c
			}
		} else {
			return []byte(*content)
		}
	}
	return []byte{}
}

func feedByBootstrapConfigFiles(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	if config.Storage == nil || config.Storage.Files == nil {
		return nil
	}

	// By default, UID and GID from own process.
	uid, gid := syscall.Getuid(), syscall.Getgid()

	for _, file := range config.Storage.Files {
		filename := file.Path
		isOverwrite := common.Get(file.Overwrite, false)

		if file.Permissions != nil {
			uid, gid = getUIDGIDFromString(*file.Permissions, sysroot.Groups, sysroot.Users)
		}

		var mode fs.FileMode = 0o664
		if file.Permissions != nil {
			if valueAsInt, err := strconv.Atoi(*file.Permissions); err != nil {
				log.Error(err)
				return err
			} else {
				mode = fs.FileMode(valueAsInt)
			}
		}

		content := getBytesFromEncoding(file.Encoding, file.Content)

		sysroot.Files = updateOrAppendFile(sysroot.Files, sr.File{
			Overwrite: isOverwrite,
			Filename:  filename,
			Content:   content,
			Mode:      mode,
			UID:       uid,
			GID:       gid,
		})
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

func feedByBootstrapConfigMounts(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	if config.Storage == nil || config.Storage.Mounts == nil {
		return nil
	}

	for _, m := range config.Storage.Mounts {
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

	return nil
}

func markAsFeededByBootstrapConfig(sysroot *sr.Sysroot, config Config) error {
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
	return nil
}

func FeedSysrootByBootstrapConfig(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	for _, fn := range []func(sysroot *sr.Sysroot, config Config) error{
		feedByBootstrapConfigGroups,
		feedByBootstrapConfigUsers,
		feedByBootstrapConfigLinks,
		feedByBootstrapConfigDirectories,
		feedByBootstrapConfigFiles,
		feedByBootstrapConfigMounts,
		markAsFeededByBootstrapConfig,
	} {
		if err := fn(sysroot, config); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}
