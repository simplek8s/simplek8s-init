package common

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"text/template"

	log "github.com/sirupsen/logrus"
)

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func IsStringInList(value string, list []string) bool {
	for _, v := range list {
		if value == v {
			return true
		}
	}
	return false
}

func IsDir(path string) bool {
	if len(path) == 0 {
		log.Debugf("path %q is empty", path)
		return false
	}

	if stat, err := os.Stat(path); err != nil {
		log.Debug(err)
		return false
	} else if !stat.IsDir() {
		log.Debugf("path %q is not a directory", path)
		return false
	}
	return true
}

func RenderTemplate(templates fs.FS, templateFilename string, data any) ([]byte, error) {
	log.WithField("start", "RenderTemplate").Debug()
	defer log.WithField("stop", "RenderTemplate").Debug()

	// Parse template
	tmpl, err := template.ParseFS(templates, templateFilename)
	if err != nil {
		log.WithFields(log.Fields{
			"templates": templates,
			"tmpl":      tmpl,
		}).Error(err)
		return nil, err
	}
	log.Debug(tmpl)

	// Render template
	content := new(bytes.Buffer)
	if err := tmpl.Execute(content, data); err != nil {
		log.WithFields(log.Fields{
			"tmpl": tmpl,
			"data": data,
		}).Error(err)
		return nil, err
	}
	log.Debug(content)

	return content.Bytes(), nil
}

func CreateSymlink(path string, target string, overwrite bool, uid int, gid int, hard bool) error {
	log.WithFields(log.Fields{
		"start":     "CreateSymlink",
		"path":      path,
		"target":    target,
		"overwrite": overwrite,
		"uid":       uid,
		"gid":       gid,
		"hard":      hard,
	}).Debug()
	defer log.WithField("stop", "CreateSymlink").Debug()

	// Overwrite?
	if info, _ := os.Stat(path); info != nil {
		if !overwrite {
			// Dont overwrite exist file
			return nil
		} else {
			// Remove exist file
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	// Create destination directory
	dirname := filepath.Dir(path)
	if err := os.MkdirAll(dirname, 0755); err != nil {
		log.WithFields(log.Fields{
			"path": path,
		}).Error(err)
		return err
	}

	if hard {
		// Create hardlink
		if err := os.Link(target, path); err != nil {
			log.WithFields(log.Fields{
				"target": target,
				"path":   path,
			}).Error(err)
			return err
		}
	} else {
		// Create symlink
		if err := os.Symlink(target, path); err != nil {
			log.WithFields(log.Fields{
				"target": target,
				"path":   path,
			}).Error(err)
			return err
		}
	}

	// Owner
	if err := os.Lchown(path, uid, gid); err != nil {
		log.WithFields(log.Fields{
			"path": path,
			"uid":  uid,
			"gid":  gid,
		}).Error(err)
		return err
	}

	return nil
}

func GetOwnUidGid() (int, int, error) {
	user, err := user.Current()
	if err != nil {
		return -1, -1, err
	}
	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return -1, -1, err
	}
	gid, err := strconv.Atoi(user.Gid)
	if err != nil {
		return -1, -1, err
	}
	return uid, gid, nil
}

func IsCmdlineDebug() bool {
	if cmdlineContent, err := os.ReadFile("/proc/cmdline"); err != nil {
		log.Warn(err)
	} else if match, err := regexp.Match(`\s*debug\s*`, cmdlineContent); err != nil {
		log.Warn(err)
	} else if match {
		return true
	}
	return false
}

func CheckFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	return !errors.Is(error, os.ErrNotExist)
}
