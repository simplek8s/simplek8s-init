package yaml

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/jlsalvador/simplek8s/internal/pkg/linux/procfs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	VERSION_1 = "1"
)

type simpleK8sGroups struct {
	Gid    *int   `yaml:"gid,omitempty"`
	Name   string `yaml:"name"`
	System *bool  `yaml:"system,omitempty"`
}
type simpleK8sUsers struct {
	Uid               *int     `yaml:"uid,omitempty"`
	Gid               *int     `yaml:"gid,omitempty"`
	Name              string   `yaml:"name,omitempty"`
	PasswordHash      *string  `yaml:"passwordHash,omitempty"`
	SshAuthorizedKeys []string `yaml:"sshAuthorizedKeys,omitempty"`
	Groups            []string `yaml:"groups,omitempty"`
	System            *bool    `yaml:"system,omitempty"`
}
type simpleK8sMounts struct {
	What    string   `yaml:"what,omitempty"`
	Where   string   `yaml:"where,omitempty"`
	Type    *string  `yaml:"type,omitempty"`
	Options *string  `yaml:"options,omitempty"`
	After   []string `yaml:"after,omitempty"`
}
type simpleK8sLinks struct {
	Overwrite *bool   `yaml:"overwrite,omitempty"`
	Path      string  `yaml:"path,omitempty"`
	Target    string  `yaml:"target,omitempty"`
	Owner     *string `yaml:"owner,omitempty"`
	Hard      *bool   `yaml:"hard,omitempty"`
}
type simpleK8sDirectories struct {
	Overwrite   *bool   `yaml:"overwrite,omitempty"`
	Path        string  `yaml:"path,omitempty"`
	Owner       *string `yaml:"owner,omitempty"`
	Permissions *string `yaml:"permissions,omitempty"`
}
type simpleK8sFiles struct {
	Overwrite   *bool   `yaml:"overwrite,omitempty"`
	Path        string  `yaml:"path,omitempty"`
	Encoding    *string `yaml:"encoding,omitempty"`
	Content     *string `yaml:"content,omitempty"`
	Owner       *string `yaml:"owner,omitempty"`
	Permissions *string `yaml:"permissions,omitempty"`
}
type simpleK8sStorage struct {
	Mounts      []simpleK8sMounts      `yaml:"mounts,omitempty"`
	Links       []simpleK8sLinks       `yaml:"links,omitempty"`
	Directories []simpleK8sDirectories `yaml:"directories,omitempty"`
	Files       []simpleK8sFiles       `yaml:"files,omitempty"`
}
type SimpleK8s struct {
	Version string            `yaml:"version"`
	Groups  []simpleK8sGroups `yaml:"groups,omitempty"`
	Users   []simpleK8sUsers  `yaml:"users,omitempty"`
	Storage *simpleK8sStorage `yaml:"storage,omitempty"`
}

func getDevices() ([]string, error) {
	log.Debug("start")
	defer log.Debug("end")

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
func getYamlContent(blockDevices []string) ([]byte, error) {
	log.WithFields(log.Fields{
		"blockDevices": blockDevices,
	}).Debug("start")
	defer log.Debug("end")

	var directories = []string{
		"/",
		"/simplek8s/",
		"/EFI/",
		"/EFI/simplek8s/",
		"/boot/",
		"/boot/simplek8s/",
		"/boot/EFI/",
		"/boot/EFI/simplek8s/",
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

			// Search first simplek8s.yaml file
			for _, path := range directories {

				fis, err := fs.ReadDir(path)
				if err != nil {
					log.WithFields(log.Fields{
						"readDirErr": err,
						"path":       path,
					}).Debug("skipping")
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
						log.WithFields(log.Fields{
							"fullFilename": fullFilename,
							"filename":     filename,
						}).Debug("skipping")
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

					b, err := io.ReadAll(file)
					if err != nil {
						log.WithField("readAllErr", err).Warn()
						continue
					}

					// Trim NUL chars
					b = bytes.Trim(b, "\x00")
					log.WithField("readAll", string(b)).Debug()

					return b, nil
				}
			}
		}
	}
	return nil, nil
}

func unmarshal(yamlContent []byte) (*SimpleK8s, error) {
	simpleK8s := new(SimpleK8s)
	if err := yaml.Unmarshal(yamlContent, &simpleK8s); err != nil {
		log.WithFields(log.Fields{
			"content": string(yamlContent),
		}).Warn(err)
		return nil, err
	}
	return simpleK8s, nil
}

// Search across all FAT32 partitions the `simplek8s.yaml` file and
// returns it as a `SimpleK8s` type struct.
func GetYamlSimpleK8s() (*SimpleK8s, error) {
	log.Debug("start")
	defer log.Debug("end")

	// Get all block devices
	blockDevices, err := getDevices()
	log.WithField("blockDevices", blockDevices).Debug()
	if err != nil {
		log.WithField("getYamlContent", err).Warn()
		return nil, err
	}

	// Search for `simplek8s.yaml` content across all block devices (filter by FAT32 partitions)
	var yamlContent []byte
	if yamlContent, err = getYamlContent(blockDevices); err != nil {
		log.WithFields(log.Fields{
			"content": string(yamlContent),
		}).Warn(err)
		return nil, err
	} else if yamlContent == nil {
		log.WithFields(log.Fields{
			"content": string(yamlContent),
		}).Warn("can not find simplek8s.yaml")
		return nil, nil
	}

	// Unmarshal the yaml content
	yamlSk8s, err := unmarshal(yamlContent)
	if err != nil {
		return nil, err
	}

	log.WithField("GetYamlSimpleK8s", yamlSk8s).Info()
	return yamlSk8s, nil
}
