package yaml

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/jlsalvador/simplek8s/pkg/linux/procfs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	VERSION_1                = "1"
	DEFAULT_BLOCKDEV_TIMEOUT = 5 // seconds
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

func waitForAnyFile(paths []string, timeout int) error {
	log.Debug("start")
	defer log.Debug("end")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	found := make(chan string, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for _, path := range paths {
					if _, err := os.Stat(path); err == nil {
						found <- path
						return
					}
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
	select {
	case path := <-found:
		log.Debugf("path %s detected.", path)
	case <-ctx.Done():
		return errors.New("timeout waiting for paths")
	}

	return nil
}

func getBlockDevices() ([]string, error) {
	log.Debug("start")
	defer log.Debug("end")

	devices := []string{
		//TODO: Support network devices.
		"/dev/sda", "/dev/vda", "/dev/nvme0n1", "/dev/mmcblk0",
	}

	// Support for cmdline blockdev=<device>.
	if blockdev, err := procfs.GetCmdlineValue[string]("blockdev", ""); err != nil {
		return nil, err
	} else if blockdev != "" {
		devices = []string{blockdev}
	}

	// Support for cmdline blockdev_timeout=<seconds>.
	blockdev_timeout, err := procfs.GetCmdlineValue[int]("blockdev_timeout", DEFAULT_BLOCKDEV_TIMEOUT)
	if err != nil {
		return nil, err
	}

	// Wait for any common block devices.
	if err := waitForAnyFile(devices, blockdev_timeout); err != nil {
		return nil, err
	}

	// Then, list all block devices.
	// Returns full block device too, not just partitions. Ex: ["/dev/vda", "/dev/vda1", "/dev/vda2"].
	if partitions, err := procfs.ParsePartitions(); err != nil {
		return nil, err
	} else {
		blockdevs := []string{}
		for _, partition := range partitions {
			blockdevs = append(blockdevs, "/dev/"+partition.Name)
		}
		return blockdevs, nil
	}
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

		disk, err := diskfs.Open(blockDevice, diskfs.WithOpenMode(diskfs.ReadOnly))
		if err != nil {
			log.WithField("diskErr", err).Debug()
			continue
		}

		nPartitions := 0
		if pt, err := disk.GetPartitionTable(); err != nil {
			// Maybe the block device has not partition table because the filesystem
			// is on the entire block device. Ex: `blockDevices = ["/dev/vda1"]`.
			log.WithField("partitionTableErr", err).Debug()
		} else {
			nPartitions = len(pt.GetPartitions())
		}

		for partitionIndex := 0; partitionIndex <= nPartitions; partitionIndex++ {
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

	// Get all block devices.
	blockDevices, err := getBlockDevices()
	if err != nil {
		log.Warn(err)
		return nil, nil
	}
	log.WithField("blockDevices", blockDevices).Debug()

	// Search for `simplek8s.yaml` content across all block devices.
	var yamlContent []byte
	if yamlContent, err = getYamlContent(blockDevices); err != nil {
		log.WithFields(log.Fields{
			"yamlContent": string(yamlContent),
		}).Error(err)
		return nil, err
	} else if yamlContent == nil {
		log.WithFields(log.Fields{
			"yamlContent": string(yamlContent),
		}).Warn("can not find simplek8s.yaml")
		return nil, nil
	}

	// Unmarshal the yaml content.
	yamlSk8s, err := unmarshal(yamlContent)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return yamlSk8s, nil
}
