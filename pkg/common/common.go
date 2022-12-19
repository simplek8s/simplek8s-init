package common

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"text/template"

	log "github.com/sirupsen/logrus"
)

func GetEnv(key, fallback string) string {
	log.WithFields(log.Fields{
		"key":      key,
		"fallback": fallback,
	}).Debug("start")
	defer log.Debug("end")

	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func IsStringInList(value string, list []string) bool {
	log.WithFields(log.Fields{
		"value": value,
		"list":  list,
	}).Debug("start")
	defer log.Debug("end")

	for _, v := range list {
		if value == v {
			return true
		}
	}
	return false
}

func IsDir(path string) bool {
	log.WithFields(log.Fields{
		"path": path,
	}).Debug("start")
	defer log.Debug("end")

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
	log.WithFields(log.Fields{
		"templates":        templates,
		"templateFilename": templateFilename,
		"data":             data,
	}).Debug("start")
	defer log.Debug("end")

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
		"path":      path,
		"target":    target,
		"overwrite": overwrite,
		"uid":       uid,
		"gid":       gid,
		"hard":      hard,
	}).Debug("start")
	defer log.Debug("end")

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

func GetOwnUidGid() (uid int, gid int) {
	uid = syscall.Getuid()
	gid = syscall.Getuid()
	return
}

func IsCmdlineDebug() bool {
	return isCmdlineDebug("/proc/cmdline")
}

func isCmdlineDebug(filePath string) bool {
	log.WithField("filePath", filePath).Debug("start")
	defer log.Debug("end")

	re := regexp.MustCompile(`\s*debug\s*`)

	if cmdlineContent, err := os.ReadFile(filePath); err != nil {
		log.Warn(err)
		return false
	} else {
		return re.Match(cmdlineContent)
	}
}

func CheckFileExists(filePath string) bool {
	log.WithField("filePath", filePath).Debug("start")
	defer log.Debug("end")

	_, error := os.Stat(filePath)
	return !errors.Is(error, os.ErrNotExist)
}
