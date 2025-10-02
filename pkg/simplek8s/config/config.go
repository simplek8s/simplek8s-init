package config

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
	"github.com/jlsalvador/simplek8s/pkg/linux"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	VERSION_1                = "1"
	DEFAULT_BLOCKDEV_TIMEOUT = 1 * time.Second
)

type Groups struct {
	Gid    *int   `yaml:"gid,omitempty"`
	Name   string `yaml:"name"`
	System *bool  `yaml:"system,omitempty"`
}
type Users struct {
	Uid               *int     `yaml:"uid,omitempty"`
	Gid               *int     `yaml:"gid,omitempty"`
	Name              string   `yaml:"name,omitempty"`
	PasswordHash      *string  `yaml:"passwordHash,omitempty"`
	SshAuthorizedKeys []string `yaml:"sshAuthorizedKeys,omitempty"`
	Groups            []string `yaml:"groups,omitempty"`
	System            *bool    `yaml:"system,omitempty"`
}
type Mounts struct {
	What    string   `yaml:"what,omitempty"`
	Where   string   `yaml:"where,omitempty"`
	Type    *string  `yaml:"type,omitempty"`
	Options *string  `yaml:"options,omitempty"`
	After   []string `yaml:"after,omitempty"`
}
type Links struct {
	Overwrite *bool   `yaml:"overwrite,omitempty"`
	Path      string  `yaml:"path,omitempty"`
	Target    string  `yaml:"target,omitempty"`
	Owner     *string `yaml:"owner,omitempty"`
	Hard      *bool   `yaml:"hard,omitempty"`
}
type Directories struct {
	Overwrite   *bool   `yaml:"overwrite,omitempty"`
	Path        string  `yaml:"path,omitempty"`
	Owner       *string `yaml:"owner,omitempty"`
	Permissions *string `yaml:"permissions,omitempty"`
}
type Files struct {
	Overwrite   *bool   `yaml:"overwrite,omitempty"`
	Path        string  `yaml:"path,omitempty"`
	Encoding    *string `yaml:"encoding,omitempty"`
	Content     *string `yaml:"content,omitempty"`
	Owner       *string `yaml:"owner,omitempty"`
	Permissions *string `yaml:"permissions,omitempty"`
}
type Storage struct {
	Mounts      []Mounts      `yaml:"mounts,omitempty"`
	Links       []Links       `yaml:"links,omitempty"`
	Directories []Directories `yaml:"directories,omitempty"`
	Files       []Files       `yaml:"files,omitempty"`
}
type Config struct {
	Version string   `yaml:"version"`
	Groups  []Groups `yaml:"groups,omitempty"`
	Users   []Users  `yaml:"users,omitempty"`
	Storage *Storage `yaml:"storage,omitempty"`
}

// waitForAnyFile waits for any of the provided paths to exist, with a timeout.
func waitForAnyFile(paths []string, timeout time.Duration) error {
	log.Debug("start")
	defer log.Debug("end")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

func unmarshal(yamlContent []byte) (*Config, error) {
	simpleK8s := new(Config)
	if err := yaml.Unmarshal(yamlContent, &simpleK8s); err != nil {
		log.WithFields(log.Fields{
			"content": string(yamlContent),
		}).Warn(err)
		return nil, err
	}
	return simpleK8s, nil
}

// TODO: Add AWS user-data from API support.
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
// http://169.254.169.254/latest/user-data
func getConfigContentFromUserData(ctx context.Context) ([]byte, error) {
	log.Debug("start")
	defer log.Debug("end")

	userDataPath := "/var/lib/cloud/user-data"
	userDataContent, err := os.ReadFile(userDataPath)
	return userDataContent, err
}

// This function will stay finding for block devices until `ctx`
// context is cancelled or `simplek8s.yaml` file is found and read.
func getFromBlockDevices(ctx context.Context) ([]byte, error) {
	log.Debug("start")
	defer log.Debug("end")

	linux.MountPseudoFS("/")
	defer linux.UnmountPseudoFS("/")

	for {
		select {
		case <-ctx.Done():
			// Context was canceled or deadline exceeded.
			log.Debug(ctx.Err())
			return nil, nil

		default:
			// Get all block devices.
			blockDevices, err := linux.GetBlockDevices()
			if err != nil {
				log.Error(err)
				return nil, err
			}

			//TODO: Add support for cmdline blockdev=<device>.
			// if blockdev, err := procfs.GetCmdlineValue("blockdev", ""); err != nil {
			// 	log.Error(err)
			// 	return nil, err
			// } else if blockdev != "" {
			// 	blockDevices = append(blockDevices, blockdev)
			// }

			if len(blockDevices) == 0 {
				// No block devices, try again later.
				time.Sleep(1 * time.Second)
				continue
			}

			// Search for `simplek8s.yaml` content across all block devices.
			data, err := getYamlContent(blockDevices)
			if err != nil {
				log.Error(err)
				return nil, err
			}
			if data == nil {
				// File not found, try again later.
				time.Sleep(1 * time.Second)
				continue
			}

			// File found.
			return data, nil
		}
	}
}

// Retrieve SimpleK8s Config.
func GetConfig() (*Config, error) {
	log.Debug("start")
	defer log.Debug("end")

	//TODO: Config timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		// Context was canceled or deadline exceeded.
		log.Debug(ctx.Err())
		return nil, nil

	default:
		if data, err := getFromBlockDevices(ctx); err != nil {
			log.Error(err)
			return nil, err
		} else if data != nil {
			return unmarshal(data)
		}
		return nil, nil
	}
}
