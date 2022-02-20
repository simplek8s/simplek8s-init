package sysroot

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jlsalvador/simplek8s/linux/passwd"
	log "github.com/sirupsen/logrus"
)

type Directory struct {
	Path string
	Mode fs.FileMode
	Uid  int
	Gid  int
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
	Path        string
	Shadows     []passwd.Shadow
	Groups      []passwd.Group
	Users       []passwd.User
	Directories []Directory
	Files       []File
}

func isIntInList(value int, list []int) bool {
	for _, v := range list {
		if value == v {
			return true
		}
	}
	return false
}

func getNextGid(groups []passwd.Group, isSystem bool) int {
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

	for isIntInList(current, ids) {
		current++
	}
	return current
}

func getNextUid(users []passwd.User, isSystem bool) int {
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

	for isIntInList(current, ids) {
		current++
	}
	return current
}

func updateOrAppendShadow(shadows []passwd.Shadow, shadow passwd.Shadow) []passwd.Shadow {
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

func (sysroot *Sysroot) parseYAMLGroups(simpleK8s SimpleK8s) error {
	for _, group := range simpleK8s.Groups {
		isSystem := false
		if group.System != nil {
			isSystem = *group.System
		}

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

func (sysroot *Sysroot) parseYAMLUsers(simpleK8s SimpleK8s) error {
	for _, user := range simpleK8s.Users {
		isSystem := false
		if user.System != nil {
			isSystem = *user.System
		}

		home := "/var/empty"
		shell := "/bin/false"
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

		passwdUser := passwd.User{
			Name:     user.Name,
			Password: "x",
			Uid:      uid,
			Gid:      gid,
			Home:     home,
			Shell:    shell,
		}
		sysroot.Users = updateOrAppendUser(sysroot.Users, passwdUser)

		// Update shadow instance
		if user.PasswordHash != nil {
			passwdShadow := passwd.Shadow{
				Name:     user.Name,
				Password: *user.PasswordHash,
			}
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
					Path: home + "/.ssh",
					Mode: 0700,
					Uid:  uid,
					Gid:  gid,
				})
				sysroot.Files = append(sysroot.Files, File{
					Filename: home + "/.ssh/authorized_keys",
					Content:  []byte(content),
					Mode:     0600,
					Uid:      uid,
					Gid:      gid,
				})
			}
		}
	}
	return nil
}

func getUidByName(users []passwd.User, name string) *int {
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

func (sysroot *Sysroot) parseYAMLDirectories(simpleK8s SimpleK8s) error {
	for _, directory := range simpleK8s.Directories {

		uid := 0
		gid := 0
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

		sysroot.Directories = append(sysroot.Directories, Directory{
			Path: directory.Path,
			Mode: mode,
			Uid:  uid,
			Gid:  gid,
		})
	}
	return nil
}

func (sysroot *Sysroot) parseYAMLFiles(simpleK8s SimpleK8s) error {
	for _, file := range simpleK8s.Files {
		filename := file.Path

		isOverwrite := true
		if file.Overwrite != nil {
			isOverwrite = *file.Overwrite
		}

		uid := 0
		gid := 0
		if file.Permissions != nil {
			uid, gid = getUidGidFromString(*file.Permissions, sysroot.Groups, sysroot.Users)
		}

		var mode fs.FileMode = 0644
		if file.Permissions != nil {
			if valueAsInt, err := strconv.Atoi(*file.Permissions); err != nil {
				return err
			} else {
				mode = fs.FileMode(valueAsInt)
			}
		}

		content := []byte{}
		if file.Content != nil {
			if file.Encoding != nil && *file.Encoding == "b64" {
				var err error
				content, err = base64.StdEncoding.DecodeString(*file.Content)
				if err != nil {
					return err
				}
			} else {
				content = []byte(*file.Content)
			}
		}

		sysroot.Files = append(sysroot.Files, File{
			Overwrite: isOverwrite,
			Filename:  filename,
			Content:   content,
			Mode:      mode,
			Uid:       uid,
			Gid:       gid,
		})
	}
	return nil
}

func (sysroot *Sysroot) parseYAML(simpleK8s SimpleK8s) error {
	if err := sysroot.parseYAMLGroups(simpleK8s); err != nil {
		return err
	}
	if err := sysroot.parseYAMLUsers(simpleK8s); err != nil {
		return err
	}
	if err := sysroot.parseYAMLDirectories(simpleK8s); err != nil {
		return err
	}
	if err := sysroot.parseYAMLFiles(simpleK8s); err != nil {
		return err
	}
	return nil
}

func parseEachLineFromFilename(filename string, eachLineFunc func(line string) error) error {
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

func (sysroot *Sysroot) parseFiles() error {
	var filename string

	filename = sysroot.Path + "/etc/shadow"
	if err := sysroot.parseFilenameShadow(filename); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"sysroot":  sysroot,
		}).Error(err)
	}

	filename = sysroot.Path + "/etc/group"
	if err := sysroot.parseFilenameGroup(filename); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"sysroot":  sysroot,
		}).Error(err)
	}

	filename = sysroot.Path + "/etc/passwd"
	if err := sysroot.parseFilenamePasswd(filename); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"sysroot":  sysroot,
		}).Error(err)
	}

	return nil
}

func writeFile(filename string, content []byte, mode fs.FileMode) error {
	path := filepath.Dir(filename)
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filename, content, mode); err != nil {
		return err
	}
	if err := os.Chmod(filename, mode); err != nil {
		return err
	}
	if err := os.Chown(filename, 0, 0); err != nil {
		return err
	}
	return nil
}

func writeFileEtcShadow(sysroot Sysroot) error {
	filename := sysroot.Path + "/etc/shadow"
	mode := fs.FileMode(0600)
	content := ""
	for _, shadow := range sysroot.Shadows {
		if line, err := shadow.Marshal(); err != nil {
			return err
		} else {
			content += fmt.Sprintln(line)
		}
	}
	return writeFile(filename, []byte(content), mode)
}

func writeFileEtcGroup(sysroot Sysroot) error {
	filename := sysroot.Path + "/etc/group"
	mode := fs.FileMode(0644)
	content := ""
	for _, group := range sysroot.Groups {
		if line, err := group.Marshal(); err != nil {
			return err
		} else {
			content += fmt.Sprintln(line)
		}
	}
	return writeFile(filename, []byte(content), mode)
}

func writeFileEtcPasswd(sysroot Sysroot) error {
	filename := sysroot.Path + "/etc/passwd"
	mode := fs.FileMode(0644)
	content := ""
	for _, user := range sysroot.Users {
		if line, err := user.Marshal(); err != nil {
			return err
		} else {
			content += fmt.Sprintln(line)
		}
	}
	return writeFile(filename, []byte(content), mode)
}

func writeDirectories(sysroot Sysroot) error {
	for _, directory := range sysroot.Directories {
		if err := os.MkdirAll(sysroot.Path+directory.Path, directory.Mode); err != nil {
			return err
		}
		if err := os.Chown(sysroot.Path+directory.Path, directory.Uid, directory.Uid); err != nil {
			return err
		}
	}
	return nil
}

func writeFiles(sysroot Sysroot) error {
	for _, file := range sysroot.Files {
		filename := sysroot.Path + file.Filename

		// If overwrite == false and file exists, skip it
		if !file.Overwrite {
			if _, err := os.Stat(filename); err == nil {
				continue
			}
		}

		if err := os.WriteFile(filename, file.Content, file.Mode); err != nil {
			return err
		}
		if err := os.Chmod(filename, file.Mode); err != nil {
			return err
		}
		if err := os.Chown(filename, file.Uid, file.Gid); err != nil {
			return err
		}
	}
	return nil
}

func (sysroot *Sysroot) write() error {
	log.WithFields(log.Fields{
		"shadows":     sysroot.Shadows,
		"groups":      sysroot.Groups,
		"users":       sysroot.Users,
		"directories": sysroot.Directories,
		"files":       sysroot.Files,
	}).Debug()

	if err := writeFileEtcShadow(*sysroot); err != nil {
		log.Error(err)
		return err
	}
	if err := writeFileEtcGroup(*sysroot); err != nil {
		log.Error(err)
		return err
	}
	if err := writeFileEtcPasswd(*sysroot); err != nil {
		log.Error(err)
		return err
	}
	if err := writeDirectories(*sysroot); err != nil {
		log.Error(err)
		return err
	}
	if err := writeFiles(*sysroot); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func New(path string) (*Sysroot, error) {
	if len(path) == 0 {
		err := errors.New("path is required")
		log.WithFields(log.Fields{
			"path": path,
		}).Panic(err)
		return nil, err
	}

	result := Sysroot{
		Path: path,
		Shadows: []passwd.Shadow{
			{
				Name:     "root",
				Password: "!!",
			},
		},
		Groups: []passwd.Group{
			{
				Name:     "root",
				Password: "",
				Gid:      0,
				UserList: []string{},
			},
		},
		Users: []passwd.User{
			{
				Name: "root",
				Uid:  0,
				Gid:  0,
				Gecos: []string{
					"Super User",
				},
				Home:  "/root",
				Shell: "/usr/bin/sh",
			},
		},
	}

	return &result, nil
}

func (sysroot *Sysroot) YamlParser() error {
	log.Debug("init simplek8s.yaml parser")

	simpleK8s := SimpleK8s{}
	if err := GetYamlSimpleK8s(&simpleK8s); err != nil {
		log.WithField("simpleK8s", simpleK8s).Error(err)
		return err
	}

	if err := sysroot.parseFiles(); err != nil {
		log.WithFields(log.Fields{
			"sysroot": sysroot,
		}).Error(err)
		return err
	}

	if err := sysroot.parseYAML(simpleK8s); err != nil {
		log.WithFields(log.Fields{
			"simpleK8s": simpleK8s,
			"sysroot":   sysroot,
		}).Error(err)
		return err
	}

	if err := sysroot.write(); err != nil {
		log.WithFields(log.Fields{
			"simpleK8s": simpleK8s,
			"sysroot":   sysroot,
		}).Error(err)
		return err
	}
	return nil
}
