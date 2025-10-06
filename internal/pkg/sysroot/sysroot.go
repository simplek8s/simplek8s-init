// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package sysroot

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/jlsalvador/simplek8s/pkg/common"
	"github.com/jlsalvador/simplek8s/pkg/linux/passwd"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/bootstrap"
	log "github.com/sirupsen/logrus"
)

type Mount struct {
	What    string
	Where   string
	Type    string
	Options string
}

type Link struct {
	Overwrite bool
	Path      string
	Target    string
	Uid       int
	Gid       int
	Hard      bool
}

type Directory struct {
	Overwrite bool
	Path      string
	Mode      fs.FileMode
	Uid       int
	Gid       int
}

type File struct {
	Overwrite bool
	Filename  string
	Content   []byte
	Mode      fs.FileMode
	Uid       int
	Gid       int
}

type Sysroot struct {
	Shadows     []passwd.Shadow
	Groups      []passwd.Group
	Users       []passwd.User
	Mounts      []Mount
	Links       []Link
	Directories []Directory
	Files       []File
}

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

func updateOrAppendShadow(shadows []passwd.Shadow, shadow passwd.Shadow) []passwd.Shadow {
	log.WithFields(log.Fields{
		"shadows": shadows,
		"shadow":  shadow,
	}).Debug("start")
	defer log.Debug("end")

	newShadows := make([]passwd.Shadow, len(shadows))
	copy(newShadows, shadows)
	for index, shadowFromList := range newShadows {
		if shadow.Name == shadowFromList.Name {
			newShadows[index] = shadow
			return newShadows
		}
	}
	return append(newShadows, shadow)
}

func updateOrAppendGroup(groups []passwd.Group, group passwd.Group) []passwd.Group {
	log.WithFields(log.Fields{
		"groups": groups,
		"group":  group,
	}).Debug("start")
	defer log.Debug("end")

	newGroups := make([]passwd.Group, len(groups))
	copy(newGroups, groups)
	for index, groupFromList := range newGroups {
		if group.Name == groupFromList.Name {
			newGroups[index] = group
			return newGroups
		}
	}
	return append(newGroups, group)
}

func updateOrAppendUser(users []passwd.User, user passwd.User) []passwd.User {
	log.WithFields(log.Fields{
		"users": users,
		"user":  user,
	}).Debug("start")
	defer log.Debug("end")

	newUsers := make([]passwd.User, len(users))
	copy(newUsers, users)
	for index, userFromList := range newUsers {
		if user.Name == userFromList.Name {
			newUsers[index] = user
			return newUsers
		}
	}
	return append(newUsers, user)
}

func (sysroot *Sysroot) parseYAMLGroups(config bootstrap.Config) error {
	log.WithFields(log.Fields{
		"config": config,
	}).Debug("start")
	defer log.Debug("end")

	for _, group := range config.Groups {
		isSystem := getBoolByDefault(group.System, false)

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
		sysroot.Groups = updateOrAppendGroup(sysroot.Groups, group)
	}
	return nil
}

func (sysroot *Sysroot) parseYAMLUsers(config bootstrap.Config) error {
	log.WithFields(log.Fields{
		"config": config,
	}).Debug("start")
	defer log.Debug("end")

	for _, user := range config.Users {
		isSystem := getBoolByDefault(user.System, false)

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
		sysroot.Users = updateOrAppendUser(sysroot.Users, passwdUser)

		// Update shadow instance
		if user.PasswordHash != nil {
			passwdShadow := passwd.NewShadow(passwd.Shadow{
				Name:     user.Name,
				Password: *user.PasswordHash,
			})
			sysroot.Shadows = updateOrAppendShadow(sysroot.Shadows, passwdShadow)
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
				sysroot.Directories = append(sysroot.Directories, Directory{
					Overwrite: false,
					Path:      home + "/.ssh",
					Mode:      0700,
					Uid:       uid,
					Gid:       gid,
				})
				sysroot.Files = append(sysroot.Files, File{
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

func getBoolByDefault(pointer *bool, fallback bool) bool {
	log.Debug("start")
	defer log.Debug("end")

	if pointer != nil {
		return *pointer
	}
	return fallback
}

func updateOrAppendLink(links []Link, link Link) []Link {
	log.Debug("start")
	defer log.Debug("end")

	for i, l := range links {
		if l.Path == link.Path {
			// Replace
			links[i] = link
			return links
		}
	}
	// Not found, just append
	return append(links, link)
}

func (sysroot *Sysroot) parseYAMLLinks(config bootstrap.Config) error {
	log.WithFields(log.Fields{
		"config": config,
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

			isOverwrite := getBoolByDefault(link.Overwrite, false)
			isHard := getBoolByDefault(link.Hard, false)

			sysroot.Links = updateOrAppendLink(sysroot.Links, Link{
				Overwrite: isOverwrite,
				Path:      link.Path,
				Target:    link.Target,
				Uid:       uid,
				Gid:       gid,
				Hard:      isHard,
			})
		}
	}
	return nil
}

func updateOrAppendDirectory(directories []Directory, directory Directory) []Directory {
	log.Debug("start")
	defer log.Debug("end")

	for i, d := range directories {
		if d.Path == directory.Path {
			// Replace
			directories[i] = directory
			return directories
		}
	}
	// Not found, just append
	return append(directories, directory)
}

func (sysroot *Sysroot) parseYAMLDirectories(config bootstrap.Config) error {
	log.WithFields(log.Fields{
		"config": config,
	}).Debug("start")
	defer log.Debug("end")

	if config.Storage != nil {

		// Set UID and GID from own process by default
		defaultUid, defaultGid := common.GetOwnUidGid()

		for _, directory := range config.Storage.Directories {
			isOverwrite := getBoolByDefault(directory.Overwrite, false)

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

			sysroot.Directories = updateOrAppendDirectory(sysroot.Directories, Directory{
				Overwrite: isOverwrite,
				Path:      directory.Path,
				Mode:      mode,
				Uid:       uid,
				Gid:       gid,
			})
		}
	}
	return nil
}

func updateOrAppendFile(files []File, file File) []File {
	log.Debug("start")
	defer log.Debug("end")

	for i, d := range files {
		if d.Filename == file.Filename {
			// Replace
			files[i] = file
			return files
		}
	}
	// Not found, just append
	return append(files, file)
}

// TODO:
//   - Fetch from HTTP when encoding is http
//   - Fetch from HTTPS when encoding is https
func GetBytesFromEncoding(encoding *string, content *string) []byte {
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

func (sysroot *Sysroot) parseYAMLFiles(config bootstrap.Config) error {
	log.WithFields(log.Fields{
		"config": config,
	}).Debug("start")
	defer log.Debug("end")

	if config.Storage != nil {

		// Set UID and GID from own process by default
		defaultUid, defaultGid := common.GetOwnUidGid()

		for _, file := range config.Storage.Files {
			filename := file.Path
			isOverwrite := getBoolByDefault(file.Overwrite, false)

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

			content := GetBytesFromEncoding(file.Encoding, file.Content)

			sysroot.Files = updateOrAppendFile(sysroot.Files, File{
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

func (sysroot *Sysroot) FeedByBootstrapConfig(config bootstrap.Config) error {
	log.WithFields(log.Fields{
		"config": config,
	}).Debug("start")
	defer log.Debug("end")

	if err := sysroot.parseYAMLGroups(config); err != nil {
		log.Error(err)
		return err
	}
	if err := sysroot.parseYAMLUsers(config); err != nil {
		log.Error(err)
		return err
	}
	if err := sysroot.parseYAMLLinks(config); err != nil {
		log.Error(err)
		return err
	}
	if err := sysroot.parseYAMLDirectories(config); err != nil {
		log.Error(err)
		return err
	}
	if err := sysroot.parseYAMLFiles(config); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func parseEachLineFromFilename(filename string, eachLineFunc func(line string) error) error {
	log.Debug("start")
	defer log.Debug("end")

	file, err := os.Open(filename)
	if err != nil {
		log.WithField("filename", filename).Error(err)
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		textline := scanner.Text()
		if err := eachLineFunc(textline); err != nil {
			return err
		}
	}
	return nil
}

func (sysroot *Sysroot) parseFilenameShadow(filename string) error {
	log.Debug("start")
	defer log.Debug("end")

	return parseEachLineFromFilename(filename, func(line string) error {
		shadow := passwd.Shadow{}
		if err := passwd.UnmarshalShadow(line, &shadow); err != nil {
			log.WithFields(log.Fields{
				"filename": filename,
				"line":     line,
			}).Error(err)
			return err
		}
		sysroot.Shadows = updateOrAppendShadow(sysroot.Shadows, shadow)
		return nil
	})
}

func (sysroot *Sysroot) parseFilenameGroup(filename string) error {
	log.Debug("start")
	defer log.Debug("end")

	return parseEachLineFromFilename(filename, func(line string) error {
		group := passwd.Group{}
		if err := passwd.UnmarshalGroup(line, &group); err != nil {
			log.WithFields(log.Fields{
				"filename": filename,
				"line":     line,
			}).Error(err)
			return err
		}
		sysroot.Groups = updateOrAppendGroup(sysroot.Groups, group)
		return nil
	})
}

func (sysroot *Sysroot) parseFilenamePasswd(filename string) error {
	log.Debug("start")
	defer log.Debug("end")

	return parseEachLineFromFilename(filename, func(line string) error {
		user := passwd.User{}
		if err := passwd.UnmarshalUser(line, &user); err != nil {
			log.WithFields(log.Fields{
				"filename": filename,
				"line":     line,
			}).Error(err)
			return err
		}
		sysroot.Users = updateOrAppendUser(sysroot.Users, user)
		return nil
	})
}

func (sysroot *Sysroot) FeedByFiles(where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Debug("start")
	defer log.Debug("end")

	var filename string

	filename = filepath.Join(where, "/etc/shadow")
	if err := sysroot.parseFilenameShadow(filename); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"sysroot":  sysroot,
		}).Debug(err)
	}

	filename = filepath.Join(where, "/etc/group")
	if err := sysroot.parseFilenameGroup(filename); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"sysroot":  sysroot,
		}).Debug(err)
	}

	filename = filepath.Join(where, "/etc/passwd")
	if err := sysroot.parseFilenamePasswd(filename); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"sysroot":  sysroot,
		}).Debug(err)
	}

	return nil
}

func writeFile(filename string, content []byte, mode fs.FileMode, uid int, gid int) error {
	log.Debug("start")
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

func (sr *Sysroot) writeShadow(where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Debug("start")
	defer log.Debug("end")

	if len(sr.Shadows) == 0 {
		return nil
	}

	content := ""
	for _, s := range sr.Shadows {
		if line, err := s.Marshal(); err != nil {
			return err
		} else {
			content += fmt.Sprintln(line)
		}
	}

	dst := filepath.Join(where, "/etc/shadow")
	mode := fs.FileMode(0600)
	return writeFile(dst, []byte(content), mode, 0, 0)
}

func (sr *Sysroot) writeGroups(where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Debug("start")
	defer log.Debug("end")

	if len(sr.Groups) == 0 {
		return nil
	}

	content := ""
	for _, g := range sr.Groups {
		if line, err := g.Marshal(); err != nil {
			return err
		} else {
			content += fmt.Sprintln(line)
		}
	}

	dst := filepath.Join(where, "/etc/group")
	mode := fs.FileMode(0644)
	return writeFile(dst, []byte(content), mode, 0, 0)
}

func (sr *Sysroot) writeUsers(where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Debug("start")
	defer log.Debug("end")

	if len(sr.Users) == 0 {
		return nil
	}

	content := ""
	for _, user := range sr.Users {
		if line, err := user.Marshal(); err != nil {
			return err
		} else {
			content += fmt.Sprintln(line)
		}
	}

	dst := filepath.Join(where, "/etc/passwd")
	mode := fs.FileMode(0644)
	return writeFile(dst, []byte(content), mode, 0, 0)
}

func (sysroot *Sysroot) writeLinks(where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Debug("start")
	defer log.Debug("end")

	for _, l := range sysroot.Links {
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

func (sysroot *Sysroot) writeDirectories(where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Debug("start")
	defer log.Debug("end")

	for _, d := range sysroot.Directories {
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

func (sysroot *Sysroot) writeFiles(where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Debug("start")
	defer log.Debug("end")

	for _, f := range sysroot.Files {
		dst := filepath.Join(where, f.Filename)

		// If overwrite == false and file exists, skip it
		if !f.Overwrite {
			if _, err := os.Stat(dst); err == nil {
				continue
			}
		}

		if err := writeFile(dst, f.Content, f.Mode, f.Uid, f.Gid); err != nil {
			return err
		}
	}
	return nil
}

// Commit all changes to Sysroot.Path
func (sysroot *Sysroot) Write(where string) error {
	log.WithFields(log.Fields{
		"sysroot": sysroot,
	}).Debug("start")
	log.Debug("end")

	if !common.IsDir(where) {
		return fmt.Errorf("%q is not a directory", where)
	}

	if err := sysroot.writeLinks(where); err != nil {
		log.Error(err)
		return err
	}
	if err := sysroot.writeDirectories(where); err != nil {
		log.Error(err)
		return err
	}
	if err := sysroot.writeFiles(where); err != nil {
		log.Error(err)
		return err
	}
	if err := sysroot.writeShadow(where); err != nil {
		log.Error(err)
		return err
	}
	if err := sysroot.writeGroups(where); err != nil {
		log.Error(err)
		return err
	}
	if err := sysroot.writeUsers(where); err != nil {
		log.Error(err)
		return err
	}
	return nil
}
