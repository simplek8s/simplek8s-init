package yaml

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/jlsalvador/simplek8s/internal/pkg/linux/procfs"
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
	Storage struct {
		Mounts []struct {
			What    string  `yaml:"what,omitempty"`
			Where   string  `yaml:"where,omitempty"`
			Type    *string `yaml:"type,omitempty"`
			Options *string `yaml:"options,omitempty"`
		} `yaml:"mounts,omitempty"`
		Links []struct {
			Overwrite *bool   `yaml:"overwrite,omitempty"`
			Path      string  `yaml:"path,omitempty"`
			Target    string  `yaml:"target,omitempty"`
			Owner     *string `yaml:"owner,omitempty"`
			Hard      *bool   `yaml:"hard,omitempty"`
		} `yaml:"links,omitempty"`
		Directories []struct {
			Overwrite   *bool   `yaml:"overwrite,omitempty"`
			Path        string  `yaml:"path,omitempty"`
			Owner       *string `yaml:"owner,omitempty"`
			Permissions *string `yaml:"permissions,omitempty"`
		} `yaml:"directories,omitempty"`
		Files []struct {
			Overwrite   *bool   `yaml:"overwrite,omitempty"`
			Path        string  `yaml:"path,omitempty"`
			Encoding    *string `yaml:"encoding,omitempty"`
			Content     *string `yaml:"content,omitempty"`
			Owner       *string `yaml:"owner,omitempty"`
			Permissions *string `yaml:"permissions,omitempty"`
		} `yaml:"files,omitempty"`
	} `yaml:"storage,omitempty"`
}

func getDevices() ([]string, error) {
	var err error
	disks := []string{}

	partitions, err := procfs.ParsePartitions()
	if err != nil {
		return nil, err
	}

	for _, partition := range partitions {
		disks = append(disks, "/dev/"+partition.Name)
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

		for partitionIndex := 0; partitionIndex <= len(ps); partitionIndex++ {
			log.WithField("partitionIndex", partitionIndex).Debug()

			fs, err := disk.GetFilesystem(partitionIndex)
			if err != nil {
				log.Debug("fs err", err)
				continue
			}

			if fs.Type() != filesystem.TypeFat32 {
				log.WithField("partitionIndex", partitionIndex).Debug("skipping")
				continue
			}

			// Search yaml first into `/`, then `/simplek8s/`
			for _, path := range []string{"/", "/simplek8s/"} {

				fis, err := fs.ReadDir(path)
				if err != nil {
					log.WithField("readDirErr", err).Warn()
					continue
				}

				for _, fi := range fis {
					filename := fi.Name()
					fullFilename := filepath.Join(path, filename)
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
						"blockDevice":    blockDevice,
						"partitionIndex": partitionIndex,
						"fullFilename":   fullFilename,
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
	}
	return nil, nil
}

// Search across all FAT32 partitions the `simplek8s.yaml` file and
// returns it as a `SimpleK8s` type struct.
func GetYamlSimpleK8s() (*SimpleK8s, error) {

	// Search for `simplek8s.yaml` content across all FAT32 partitions
	b, err := getYamlContent()
	if err != nil {
		log.WithFields(log.Fields{
			"content": string(b),
		}).Warn(err)
		return nil, err
	}
	if b == nil {
		log.WithFields(log.Fields{
			"content": string(b),
		}).Warn("simplek8s.yaml is empty")
		return nil, nil
	}

	// Unmarshal the yaml content
	simpleK8s := new(SimpleK8s)
	if err := yaml.Unmarshal(b, &simpleK8s); err != nil {
		log.WithFields(log.Fields{
			"content": string(b),
		}).Warn(err)
		return nil, err
	}

	log.WithField("GetYamlSimpleK8s", simpleK8s).Info()
	return simpleK8s, nil
}
