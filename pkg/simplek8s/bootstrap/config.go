package bootstrap

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
	"github.com/goccy/go-yaml"
	"github.com/jlsalvador/simplek8s/pkg/linux/mount"
	"github.com/jlsalvador/simplek8s/pkg/linux/sysfs"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	CONFIG_VERSION_1         = "1"
	DEFAULT_BLOCKDEV_TIMEOUT = 1 * time.Second
)

type Groups struct {
	Gid    *int   `yaml:",omitempty"`
	Name   string `yaml:""`
	System *bool  `yaml:",omitempty"`
}
type Users struct {
	Uid                         *int     `yaml:",omitempty"`
	Gid                         *int     `yaml:",omitempty"`
	Name                        string   `yaml:""`
	PasswordHash                *string  `yaml:"password_hash,omitempty"`
	DeprecatedPasswordHash      *string  `yaml:"passwordHash,omitempty"`
	SshAuthorizedKeys           []string `yaml:"ssh_authorized_keys,omitempty"`
	DeprecatedSshAuthorizedKeys []string `yaml:"sshAuthorizedKeys,omitempty"`
	Groups                      []string `yaml:",omitempty"`
	System                      *bool    `yaml:",omitempty"`
}
type Mounts struct {
	What    string   `yaml:""`
	Where   string   `yaml:""`
	Type    *string  `yaml:",omitempty"`
	Options *string  `yaml:",omitempty"`
	After   []string `yaml:",omitempty"`
}
type Links struct {
	Overwrite *bool   `yaml:",omitempty"`
	Path      string  `yaml:""`
	Target    string  `yaml:""`
	Owner     *string `yaml:",omitempty"`
	Hard      *bool   `yaml:",omitempty"`
}
type Directories struct {
	Overwrite   *bool   `yaml:",omitempty"`
	Path        string  `yaml:""`
	Owner       *string `yaml:",omitempty"`
	Permissions *string `yaml:",omitempty"`
}
type Files struct {
	Overwrite   *bool   `yaml:",omitempty"`
	Path        string  `yaml:""`
	Encoding    *string `yaml:",omitempty"`
	Content     *string `yaml:",omitempty"`
	Owner       *string `yaml:",omitempty"`
	Permissions *string `yaml:",omitempty"`
}
type Storage struct {
	Mounts      []Mounts      `yaml:",omitempty"`
	Links       []Links       `yaml:",omitempty"`
	Directories []Directories `yaml:",omitempty"`
	Files       []Files       `yaml:",omitempty"`
}
type Config struct {
	Version string   `yaml:""`
	Groups  []Groups `yaml:",omitempty"`
	Users   []Users  `yaml:",omitempty"`
	Storage *Storage `yaml:",omitempty"`
}

func (c *Config) String() string {
	j, _ := yaml.MarshalWithOptions(c, yaml.Indent(2))
	return string(j)
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

	if len(blockDevices) > 0 {
		if _, err := os.Stat(blockDevices[0]); errors.Is(err, os.ErrNotExist) {
			mount.Mount(mount.Mountpoints.Dev)
			defer unix.Unmount(mount.Mountpoints.Dev.Target, 0)
		}
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
	config := &Config{
		Version: CONFIG_VERSION_1,
	}

	if err := yaml.Unmarshal(yamlContent, config); err != nil {
		log.WithFields(log.Fields{
			"content": string(yamlContent),
		}).Warn(err)
		return nil, err
	}

	for i := range config.Users {
		if config.Users[i].PasswordHash == nil && config.Users[i].DeprecatedPasswordHash != nil {
			config.Users[i].PasswordHash = config.Users[i].DeprecatedPasswordHash
			config.Users[i].DeprecatedPasswordHash = nil
		}
		if config.Users[i].SshAuthorizedKeys == nil && config.Users[i].DeprecatedSshAuthorizedKeys != nil {
			config.Users[i].SshAuthorizedKeys = config.Users[i].DeprecatedSshAuthorizedKeys
			config.Users[i].DeprecatedSshAuthorizedKeys = nil
		}
	}

	return config, nil
}

// This function will stay finding for block devices until `ctx`
// context is cancelled or `simplek8s.yaml` file is found and read.
func getFromBlockDevices(ctx context.Context) ([]byte, error) {
	log.Debug("start")
	defer log.Debug("end")

	for {
		select {
		case <-ctx.Done():
			// Context was canceled or deadline exceeded.
			log.Debug(ctx.Err())
			return nil, nil

		default:
			// Get all block devices.
			blockDevices, err := sysfs.GetBlockDevices()
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
		// TODO: Fetch from AWS user-data:
		// 		 https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
		// 		 http://169.254.169.254/latest/user-data

		if data, err := getFromBlockDevices(ctx); err != nil {
			log.Error(err)
			return nil, err
		} else if data != nil {
			return unmarshal(data)
		}
		return nil, nil
	}
}
