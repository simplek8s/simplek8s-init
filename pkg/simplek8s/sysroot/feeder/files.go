package feeder

import (
	"bufio"
	"os"
	"path/filepath"

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/passwd"
	sr "simplek8s/pkg/simplek8s/sysroot"

	log "github.com/sirupsen/logrus"
)

func parseEachLineFromFilename(filename string, eachLineFunc func(line string) error) error {
	log.Trace("start")
	defer log.Trace("end")

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

func parseFilenameShadow(sysroot *sr.Sysroot, filename string) error {
	log.Trace("start")
	defer log.Trace("end")

	return parseEachLineFromFilename(filename, func(line string) error {
		shadow := passwd.Shadow{}
		if err := passwd.UnmarshalShadow(line, &shadow); err != nil {
			log.WithFields(log.Fields{
				"filename": filename,
				"line":     line,
			}).Error(err)
			return err
		}
		sysroot.Shadows = common.UpdateOrAppend(sysroot.Shadows, shadow, func(a, b passwd.Shadow) bool {
			return a.Name == b.Name
		})
		return nil
	})
}

func parseFilenameGroup(sysroot *sr.Sysroot, filename string) error {
	log.Trace("start")
	defer log.Trace("end")

	return parseEachLineFromFilename(filename, func(line string) error {
		group := passwd.Group{}
		if err := passwd.UnmarshalGroup(line, &group); err != nil {
			log.WithFields(log.Fields{
				"filename": filename,
				"line":     line,
			}).Error(err)
			return err
		}
		sysroot.Groups = common.UpdateOrAppend(sysroot.Groups, group, func(a, b passwd.Group) bool {
			return a.Name == b.Name
		})
		return nil
	})
}

func parseFilenamePasswd(sysroot *sr.Sysroot, filename string) error {
	log.Trace("start")
	defer log.Trace("end")

	return parseEachLineFromFilename(filename, func(line string) error {
		user := passwd.User{}
		if err := passwd.UnmarshalUser(line, &user); err != nil {
			log.WithFields(log.Fields{
				"filename": filename,
				"line":     line,
			}).Error(err)
			return err
		}
		sysroot.Users = common.UpdateOrAppend(sysroot.Users, user, func(a, b passwd.User) bool {
			return a.Name == b.Name
		})
		return nil
	})
}

func FeedSysrootByFiles(sysroot *sr.Sysroot, where string) error {
	log.WithFields(log.Fields{
		"where": where,
	}).Trace("start")
	defer log.Trace("end")

	var filename string

	filename = filepath.Join(where, "/etc/shadow")
	if err := parseFilenameShadow(sysroot, filename); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"sysroot":  sysroot,
		}).Debug(err)
	}

	filename = filepath.Join(where, "/etc/group")
	if err := parseFilenameGroup(sysroot, filename); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"sysroot":  sysroot,
		}).Debug(err)
	}

	filename = filepath.Join(where, "/etc/passwd")
	if err := parseFilenamePasswd(sysroot, filename); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"sysroot":  sysroot,
		}).Debug(err)
	}

	return nil
}
