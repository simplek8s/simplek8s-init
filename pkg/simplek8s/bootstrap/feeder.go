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

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/passwd"
	sr "simplek8s/pkg/simplek8s/sysroot"

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

// feedByBootstrapConfigGroupsWithGUIDs appends into sysroot.Groups user
// defined groups with proper GIDs.
//
// Groups without GIDs or empty names will be skipped.
func feedByBootstrapConfigGroupsWithGUIDs(sysroot *sr.Sysroot, config Config) {
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
			return a.Name == b.Name
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

		// Skip user if there is already a (user defined) group with the same user.GID.
		if user.GID != nil && slices.IndexFunc(sysroot.Groups, func(group passwd.Group) bool {
			return group.GID == *user.GID
		}) >= 0 {
			continue
		}

		isSystem := common.Get(user.System, false)
		gid := common.Get(user.GID, getNextGID(sysroot.Groups, isSystem))

		// Find GID that does not collides with the already defined groups.
		if slices.IndexFunc(sysroot.Groups, func(g passwd.Group) bool {
			return g.GID == gid
		}) >= 0 {
			gid = getNextGID(sysroot.Groups, isSystem)
		}

		sysroot.Groups = append(sysroot.Groups, passwd.NewGroup(passwd.Group{
			Name: user.Name,
			GID:  gid,
			UserList: []string{
				user.Name,
			},
		}))
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
		group := passwd.NewGroup(passwd.Group{
			Name: group.Name,
			GID:  gid,
		})
		sysroot.Groups = common.UpdateOrAppend(sysroot.Groups, group, func(a, b passwd.Group) bool {
			return a.Name == b.Name
		})
	}
}

func feedByBootstrapConfigGroups(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Trace("start")
	defer log.Trace("end")

	feedByBootstrapConfigGroupsWithGUIDs(sysroot, config)
	feedByBootstrapConfigGroupsFromUsers(config, sysroot)
	feedByBootstrapConfigGroupsWithoutGIDs(config, sysroot)

	// Sort groups by its GIDs.
	sort.Slice(sysroot.Groups, func(i, j int) bool {
		return sysroot.Groups[i].GID < sysroot.Groups[j].GID
	})

	return nil
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

		home := ""  // Will set the default when `NewUser()`
		shell := "" // Will set the default when `NewUser()`
		isSystem := common.Get(user.System, false)

		// Set the home and shell values depending of user.{Name,System}.
		if user.Name == "root" {
			// Just for the root user.
			home = "/root"
			shell = "/usr/bin/sh"
			if user.UID == nil {
				user.UID = pointy.Int(0)
			}
			if user.GID == nil {
				user.GID = pointy.Int(0)
			}
		} else if !isSystem {
			// Normal user.
			home = fmt.Sprintf("/home/%s", user.Name)
			shell = "/usr/bin/sh"
		} else {
			// System user.
			home = "/"
			shell = "/usr/sbin/nologin"
		}

		uid := common.Get(user.UID, getNextUID(sysroot.Users, isSystem))
		gid := common.Get(user.GID, -1)

		// user.GID is nil, try to find a group with the same user.Name.
		if gid == -1 {
			gid = common.Get(getGIDByName(sysroot.Groups, user.Name), -1)
		}

		// Create a new group if there are not groups with the same user.Name
		// or user.GID.
		if gid == -1 {
			passwdGroup := passwd.NewGroup(passwd.Group{
				Name:     user.Name,
				Password: "",
				GID:      getNextGID(sysroot.Groups, isSystem),
				UserList: []string{user.Name},
			})
			sysroot.Groups = append(sysroot.Groups, passwdGroup)
			gid = passwdGroup.GID
		}

		// Add or update user to sysroot.Users.
		passwdUser := passwd.NewUser(passwd.User{
			Name:     user.Name,
			Password: "x",
			UID:      uid,
			GID:      gid,
			Home:     home,
			Shell:    shell,
		})
		sysroot.Users = common.UpdateOrAppend(sysroot.Users, passwdUser, func(a, b passwd.User) bool {
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

		// Update groups instances.
		for _, groupName := range user.Groups {
			i := slices.IndexFunc(sysroot.Groups, func(group passwd.Group) bool {
				return group.Name == groupName
			})

			if i >= 0 {
				// Try to update an already defined group.
				sysroot.Groups[i].UserList = append(sysroot.Groups[i].UserList, passwdUser.Name)
			} else {
				// Create a new group.
				sysroot.Groups = append(sysroot.Groups, passwd.Group{
					Name:     groupName,
					Password: "",
					GID:      getNextGID(sysroot.Groups, isSystem),
					UserList: []string{passwdUser.Name},
				})
			}
		}

		// Set the SSH Authorized keys.
		if len(user.SSHAuthorizedKeys) > 0 {
			content := []byte{}
			for _, sshPublicKey := range user.SSHAuthorizedKeys {
				content = fmt.Appendln(content, sshPublicKey)
			}

			if len(content) > 0 {
				sysroot.Directories = append(sysroot.Directories, sr.Directory{
					Overwrite: false,
					Path:      home + "/.ssh",
					Mode:      0o700,
					UID:       uid,
					GID:       gid,
				})
				sysroot.Files = append(sysroot.Files, sr.File{
					Overwrite: false,
					Filename:  home + "/.ssh/authorized_keys",
					Content:   content,
					Mode:      0o600,
					UID:       uid,
					GID:       gid,
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
		sysroot.Mounts = common.UpdateOrAppend(sysroot.Mounts, sr.Mount{
			What:    m.What,
			Where:   m.Where,
			Type:    common.Get(m.Type, ""),
			Options: common.Get(m.Options, ""),
		}, func(a, b sr.Mount) bool {
			return a.Where == b.Where
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
