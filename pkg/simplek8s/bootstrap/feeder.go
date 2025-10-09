package bootstrap

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"

	"github.com/jlsalvador/simplek8s/pkg/common"
	"github.com/jlsalvador/simplek8s/pkg/linux/passwd"
	sr "github.com/jlsalvador/simplek8s/pkg/simplek8s/sysroot"
	log "github.com/sirupsen/logrus"
)

func getNextGid(groups []passwd.Group, isSystem bool) int {
	log.WithFields(log.Fields{
		"groups":   groups,
		"isSystem": isSystem,
	}).Debug("start")
	defer log.Debug("end")

	current := 0
	if !isSystem {
		current = 1000
	}

	ids := []int{}
	for _, group := range groups {
		if (isSystem && group.Gid >= 1000) || (!isSystem && group.Gid < 1000) {
			continue
		}

		ids = append(ids, int(group.Gid))
	}

	for slices.Contains(ids, current) {
		current++
	}
	return current
}

func getNextUid(users []passwd.User, isSystem bool) int {
	log.WithFields(log.Fields{
		"users":    users,
		"isSystem": isSystem,
	}).Debug("start")
	defer log.Debug("end")

	current := 0
	if !isSystem {
		current = 1000
	}

	ids := []int{}
	for _, user := range users {
		if (isSystem && user.Uid >= 1000) || (!isSystem && user.Uid < 1000) {
			continue
		}

		ids = append(ids, int(user.Uid))
	}

	for slices.Contains(ids, current) {
		current++
	}
	return current
}

func getUidByName(users []passwd.User, name string) *int {
	log.Debug("start")
	defer log.Debug("end")

	for _, user := range users {
		if user.Name == name {
			return &user.Uid
		}
	}
	log.WithFields(log.Fields{
		"name":  name,
		"users": users,
	}).Error("can not find uid by name")
	return nil
}

func getGidByName(groups []passwd.Group, name string) *int {
	log.Debug("start")
	defer log.Debug("end")

	for _, group := range groups {
		if group.Name == name {
			return &group.Gid
		}
	}
	log.WithFields(log.Fields{
		"name":   name,
		"groups": groups,
	}).Error("can not find gid by name")
	return nil
}

func getUidGidFromString(permission string, groups []passwd.Group, users []passwd.User) (int, int) {
	log.Debug("start")
	defer log.Debug("end")

	uid := 0
	gid := 0

	ids := strings.Split(permission, ":")
	if value, err := strconv.Atoi(ids[0]); err == nil {
		uid = value
	} else {
		if value := getUidByName(users, ids[0]); value != nil {
			uid = *value
		}
	}
	if len(ids) > 1 {
		if value, err := strconv.Atoi(ids[1]); err == nil {
			gid = value
		} else {
			if value := getGidByName(groups, ids[1]); value != nil {
				gid = *value
			}
		}
	}

	return uid, gid
}

func updateOrAppendFile(files []sr.File, file sr.File) []sr.File {
	log.Debug("start")
	defer log.Debug("end")

	return common.UpdateOrAppend(files, file, func(a, b sr.File) bool {
		return a.Filename == b.Filename
	})
}

func feedByBootstrapConfigGroups(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Debug("start")
	defer log.Debug("end")

	for _, group := range config.Groups {
		isSystem := common.GetOrDefault(group.System, false)

		var gid int
		if group.Gid != nil {
			gid = *group.Gid
		} else {
			gid = getNextGid(sysroot.Groups, isSystem)
		}

		group := passwd.Group{
			Name: group.Name,
			Gid:  gid,
		}
		sysroot.Groups = common.UpdateOrAppend(sysroot.Groups, group, func(a, b passwd.Group) bool {
			return a.Name == b.Name
		})
	}
	return nil
}

func feedByBootstrapConfigUsers(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Debug("start")
	defer log.Debug("end")

	for _, user := range config.Users {
		isSystem := common.GetOrDefault(user.System, false)

		home := ""  // Default value from `NewUser()`
		shell := "" // Default value from `NewUser()`
		uid := 0
		gid := 0
		if user.Name == "root" {
			home = "/root"
			shell = "/usr/bin/sh"
		} else {
			if !isSystem {
				home = fmt.Sprintf("/home/%s", user.Name)
				shell = "/usr/bin/sh"
			}

			if user.Uid != nil {
				uid = *user.Uid
			} else {
				uid = getNextUid(sysroot.Users, isSystem)
			}

			if user.Gid != nil {
				uid = *user.Gid
			} else {
				gid = getNextGid(sysroot.Groups, isSystem)
			}
		}

		passwdUser := passwd.NewUser(passwd.User{
			Name:     user.Name,
			Password: "x",
			Uid:      uid,
			Gid:      gid,
			Home:     home,
			Shell:    shell,
		})
		sysroot.Users = common.UpdateOrAppend(sysroot.Users, passwdUser, func(a, b passwd.User) bool {
			return a.Name == b.Name
		})

		// Update shadow instance
		if user.PasswordHash != nil {
			passwdShadow := passwd.NewShadow(passwd.Shadow{
				Name:     user.Name,
				Password: *user.PasswordHash,
			})
			sysroot.Shadows = common.UpdateOrAppend(sysroot.Shadows, passwdShadow, func(a, b passwd.Shadow) bool {
				return a.Name == b.Name
			})
		}

		// Update groups instances
		for _, groupName := range user.Groups {
			found := false
			// Try to update an already defined group
			for index := range sysroot.Groups {
				if groupName == sysroot.Groups[index].Name {
					found = true
					sysroot.Groups[index].UserList = append(sysroot.Groups[index].UserList, passwdUser.Name)
					break
				}
			}
			// Or create a new group
			if !found {
				sysroot.Groups = append(sysroot.Groups, passwd.Group{
					Name:     groupName,
					Password: "",
					Gid:      getNextGid(sysroot.Groups, isSystem),
					UserList: []string{passwdUser.Name},
				})
			}
		}

		// SSH Authorized keys
		if len(user.SshAuthorizedKeys) > 0 {
			content := ""
			for _, sshPublicKey := range user.SshAuthorizedKeys {
				content += fmt.Sprintln(sshPublicKey)
			}
			if len(content) > 0 {
				sysroot.Directories = append(sysroot.Directories, sr.Directory{
					Overwrite: false,
					Path:      home + "/.ssh",
					Mode:      0700,
					Uid:       uid,
					Gid:       gid,
				})
				sysroot.Files = append(sysroot.Files, sr.File{
					Overwrite: false,
					Filename:  home + "/.ssh/authorized_keys",
					Content:   []byte(content),
					Mode:      0600,
					Uid:       uid,
					Gid:       gid,
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
	}).Debug("start")
	defer log.Debug("end")

	if config.Storage != nil {

		// Set UID and GID from own process by default
		defaultUid, defaultGid := common.GetOwnUidGid()

		for _, link := range config.Storage.Links {
			uid := defaultUid
			gid := defaultGid
			if link.Owner != nil {
				uid, gid = getUidGidFromString(*link.Owner, sysroot.Groups, sysroot.Users)
			}

			isOverwrite := common.GetOrDefault(link.Overwrite, false)
			isHard := common.GetOrDefault(link.Hard, false)

			sysroot.Links = common.UpdateOrAppend(sysroot.Links, sr.Link{
				Overwrite: isOverwrite,
				Path:      link.Path,
				Target:    link.Target,
				Uid:       uid,
				Gid:       gid,
				Hard:      isHard,
			}, func(a, b sr.Link) bool {
				return a.Path == b.Path
			})
		}
	}
	return nil
}

func feedByBootstrapConfigDirectories(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Debug("start")
	defer log.Debug("end")

	if config.Storage != nil {

		// Set UID and GID from own process by default
		defaultUid, defaultGid := common.GetOwnUidGid()

		for _, directory := range config.Storage.Directories {
			isOverwrite := common.GetOrDefault(directory.Overwrite, false)

			uid := defaultUid
			gid := defaultGid
			if directory.Owner != nil {
				uid, gid = getUidGidFromString(*directory.Owner, sysroot.Groups, sysroot.Users)
			}

			var mode fs.FileMode = 0775
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
				Uid:       uid,
				Gid:       gid,
			}, func(a, b sr.Directory) bool {
				return a.Path == b.Path
			})
		}
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
	}).Debug("start")
	defer log.Debug("end")

	if config.Storage != nil {

		// Set UID and GID from own process by default
		defaultUid, defaultGid := common.GetOwnUidGid()

		for _, file := range config.Storage.Files {
			filename := file.Path
			isOverwrite := common.GetOrDefault(file.Overwrite, false)

			uid := defaultUid
			gid := defaultGid
			if file.Permissions != nil {
				uid, gid = getUidGidFromString(*file.Permissions, sysroot.Groups, sysroot.Users)
			}

			var mode fs.FileMode = 0664
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
				Uid:       uid,
				Gid:       gid,
			})
		}
	}
	return nil
}

func feedByBootstrapConfigMounts(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Debug("start")
	defer log.Debug("end")

	for _, m := range config.Storage.Mounts {
		sysroot.Mounts = append(sysroot.Mounts, sr.Mount{
			What:    m.What,
			Where:   m.Where,
			Type:    common.GetOrDefault(m.Type, ""),
			Options: common.GetOrDefault(m.Options, ""),
		})
	}

	return nil
}

func FeedSysrootByBootstrapConfig(sysroot *sr.Sysroot, config Config) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
		"config":  config,
	}).Debug("start")
	defer log.Debug("end")

	if err := feedByBootstrapConfigGroups(sysroot, config); err != nil {
		log.Error(err)
		return err
	}
	// Caution. Users could creates or update groups.
	if err := feedByBootstrapConfigUsers(sysroot, config); err != nil {
		log.Error(err)
		return err
	}
	if err := feedByBootstrapConfigLinks(sysroot, config); err != nil {
		log.Error(err)
		return err
	}
	if err := feedByBootstrapConfigDirectories(sysroot, config); err != nil {
		log.Error(err)
		return err
	}
	if err := feedByBootstrapConfigFiles(sysroot, config); err != nil {
		log.Error(err)
		return err
	}
	if err := feedByBootstrapConfigMounts(sysroot, config); err != nil {
		log.Error(err)
		return err
	}
	return nil
}
