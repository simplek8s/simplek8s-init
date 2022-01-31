package sysroot

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type SimpleK8s struct {
	Groups []struct {
		Gid    *int   `yaml:"gid,omitempty"`
		Name   string `yaml:"name"`
		System *bool  `yaml:"system,omitempty"`
	} `yaml:"groups,omitempty"`
	Users []struct {
		Uid               *int     `yaml:"uid,omitempty"`
		Gid               *int     `yaml:"gid,omitempty"`
		Name              string   `yaml:"name,omitempty"`
		PasswordHash      *string  `yaml:"passwordHash,omitempty"`
		SshAuthorizedKeys []string `yaml:"sshAuthorizedKeys,omitempty"`
		Groups            []string `yaml:"groups,omitempty"`
		System            *bool    `yaml:"system,omitempty"`
	} `yaml:"users,omitempty"`
	Directories []struct {
		Path        string  `yaml:"path,omitempty"`
		Overwrite   bool    `yaml:"overwrite,omitempty"`
		Owner       *string `yaml:"owner,omitempty"`
		Permissions *string `yaml:"permissions,omitempty"`
	} `yaml:"directories,omitempty"`
	Files []struct {
		Path        string  `yaml:"path,omitempty"`
		Overwrite   *bool   `yaml:"overwrite,omitempty"`
		Encoding    *string `yaml:"encoding,omitempty"`
		Content     *string `yaml:"content,omitempty"`
		Owner       *string `yaml:"owner,omitempty"`
		Permissions *string `yaml:"permissions,omitempty"`
	} `yaml:"files,omitempty"`
	Links []struct {
		Path      string  `yaml:"path,omitempty"`
		Overwrite bool    `yaml:"overwrite,omitempty"`
		Target    *string `yaml:"target,omitempty"`
		Hard      bool    `yaml:"hard,omitempty"`
	} `yaml:"links,omitempty"`
}

func getDevices() ([]string, error) {
	disks := make([]string, 0)
	prefix := "/dev/block"

	fis, err := ioutil.ReadDir(prefix)
	if err != nil {
		return disks, err
	}

	for _, fi := range fis {
		disks = append(disks, prefix+"/"+fi.Name())
	}

	return disks, nil
}

// Could returns `nil, nil` if it can't find any `simplek8s.yaml` file.
func getYamlContent() ([]byte, error) {
	blockDevices, err := getDevices()
	log.WithField("devices", blockDevices).Debug()
	if err != nil {
		log.WithField("getYamlContent", err).Warn()
		return nil, err
	}

	rSimpleK8sYaml, err := regexp.Compile("^simplek8s.ya?ml$")
	if err != nil {
		log.WithField("getYamlContent", err).Warn()
		return nil, err
	}

	for _, blockDevice := range blockDevices {
		log.WithField("device", blockDevice).Debug()

		disk, err := diskfs.OpenWithMode(blockDevice, diskfs.ReadOnly)
		if err != nil {
			log.WithField("diskErr", err).Debug()
			continue
		}

		pt, err := disk.GetPartitionTable()
		if err != nil {
			log.WithField("partitionTableErr", err).Debug()
			continue
		}
		ps := pt.GetPartitions()

		for i := 0; i <= len(ps); i++ {
			log.WithField("partitionIndex", i).Debug()

			fs, err := disk.GetFilesystem(i)
			if err != nil {
				log.Debug("fs err", err)
				continue
			}
			label := fs.Label()
			log.WithField("label", label).Debug()

			if fs.Type() != filesystem.TypeFat32 {
				log.WithField("label", label).Debug("skipping")
				continue
			}

			path := "/"

			fis, err := fs.ReadDir(path)
			if err != nil {
				log.WithField("readDirErr", err).Warn()
				continue
			}

			for _, fi := range fis {
				filename := fi.Name()
				fullFilename := strings.TrimRight(path, "/") + "/" + filename
				isDir := fi.IsDir()
				rMath := rSimpleK8sYaml.MatchString(filename)
				log.WithFields(log.Fields{
					"fullFilename": fullFilename,
					"isDir":        isDir,
					"rMath":        rMath,
				}).Debug()

				if isDir || !rMath {
					log.WithField("fullFilename", fullFilename).Debug("skipping")
					continue
				}

				log.WithFields(log.Fields{
					"blockDevice":  blockDevice,
					"label":        label,
					"fullFilename": fullFilename,
				}).Info("simplek8s yaml found")

				file, err := fs.OpenFile(fullFilename, os.O_RDONLY)
				if err != nil {
					log.WithField("openFileErr", err).Warn()
					continue
				}
				defer file.Close()

				b, err := ioutil.ReadAll(file)
				if err != nil {
					log.WithField("readAllErr", err).Warn()
					continue
				}
				log.WithField("readAll", string(b)).Debug()
				return b, nil
			}
		}
	}
	return nil, nil
}

// Search across all FAT32 partitions the simplek8s.yaml and returns
// it as a `SimpleK8s` type struct.
func GetYamlSimpleK8s(simpleK8s *SimpleK8s) error {

	// Search for `simplek8s.yaml` content across all FAT32 partitions
	b, err := getYamlContent()
	if err != nil {
		log.WithFields(log.Fields{
			"content": string(b),
		}).Warn(err)
		return err
	}
	if b == nil {
		log.WithFields(log.Fields{
			"content": string(b),
		}).Warn("simplek8s.yaml is empty")
		return nil
	}

	// Unmarshal the yaml content
	if err := yaml.Unmarshal(b, &simpleK8s); err != nil {
		log.WithFields(log.Fields{
			"content": string(b),
		}).Warn(err)
		return err
	}

	log.WithField("GetYamlSimpleK8s", simpleK8s).Info()
	return nil
}
