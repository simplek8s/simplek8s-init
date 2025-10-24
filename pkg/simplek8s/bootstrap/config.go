// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package bootstrap get Config from a "simplek8s.yaml".
package bootstrap

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/mount"
	"simplek8s/pkg/linux/sysfs"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"
)

const (
	ConfigVersion1         = "1"
	DefaultBlockdevTimeout = 1 * time.Second
)

type Group struct {
	GID    *int   `yaml:",omitempty"`
	Name   string `yaml:""`
	System *bool  `yaml:",omitempty"`
}
type User struct {
	UID                         *int     `yaml:",omitempty"`
	GID                         *int     `yaml:",omitempty"`
	Name                        string   `yaml:""`
	PasswordHash                *string  `yaml:"password_hash,omitempty"`
	DeprecatedPasswordHash      *string  `yaml:"passwordHash,omitempty"` // Deprecated: use PasswordHash
	SSHAuthorizedKeys           []string `yaml:"ssh_authorized_keys,omitempty"`
	DeprecatedSSHAuthorizedKeys []string `yaml:"sshAuthorizedKeys,omitempty"` // Deprecated: use SSHAuthorizedKeys
	Groups                      []string `yaml:",omitempty"`
	System                      *bool    `yaml:",omitempty"`
}
type Mount struct {
	What    string   `yaml:""`
	Where   string   `yaml:""`
	Type    *string  `yaml:",omitempty"`
	Options *string  `yaml:",omitempty"`
	After   []string `yaml:",omitempty"`
}
type Link struct {
	Overwrite *bool   `yaml:",omitempty"`
	Path      string  `yaml:""`
	Target    string  `yaml:""`
	Owner     *string `yaml:",omitempty"`
	Hard      *bool   `yaml:",omitempty"`
}
type Directory struct {
	Overwrite   *bool   `yaml:",omitempty"`
	Path        string  `yaml:""`
	Owner       *string `yaml:",omitempty"`
	Permissions *string `yaml:",omitempty"`
}
type File struct {
	Overwrite   *bool   `yaml:",omitempty"`
	Path        string  `yaml:""`
	Encoding    *string `yaml:",omitempty"`
	Content     *string `yaml:",omitempty"`
	Owner       *string `yaml:",omitempty"`
	Permissions *string `yaml:",omitempty"`
}
type Storage struct {
	Mounts      []Mount     `yaml:",omitempty"`
	Links       []Link      `yaml:",omitempty"`
	Directories []Directory `yaml:",omitempty"`
	Files       []File      `yaml:",omitempty"`
}
type Config struct {
	Version string   `yaml:""`
	Groups  []Group  `yaml:",omitempty"`
	Users   []User   `yaml:",omitempty"`
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
	}).Trace("start")
	defer log.Trace("end")

	directories := []string{
		"/",
		"/simplek8s/",
		"/EFI/",
		"/EFI/simplek8s/",
		"/boot/",
		"/boot/simplek8s/",
		"/boot/EFI/",
		"/boot/EFI/simplek8s/",
	}

	rSimpleK8sYaml := regexp.MustCompile("^simplek8s.ya?ml$")

	if len(blockDevices) == 0 {
		return nil, nil
	}

	// diskfs requires "/dev" to open raw devices.
	if !common.IsPathExists(blockDevices[0]) {
		err := mount.Mount(mount.Mountpoints.Dev)
		if err != nil {
			return nil, err
		}
		defer mount.Unmount(mount.Mountpoints.Dev.Target, 0)
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
		Version: ConfigVersion1,
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
		if config.Users[i].SSHAuthorizedKeys == nil && config.Users[i].DeprecatedSSHAuthorizedKeys != nil {
			config.Users[i].SSHAuthorizedKeys = config.Users[i].DeprecatedSSHAuthorizedKeys
			config.Users[i].DeprecatedSSHAuthorizedKeys = nil
		}
	}

	return config, nil
}

// This function will stay finding for block devices until `ctx`
// context is cancelled or `simplek8s.yaml` file is found and read.
func getFromBlockDevices(ctx context.Context) ([]byte, error) {
	log.Trace("start")
	defer log.Trace("end")

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

			// TODO: Add support for cmdline blockdev=<device>.
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

// GetConfig retrieves SimpleK8s Config.
func GetConfig(timeout time.Duration) (*Config, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		// Context was canceled or deadline exceeded.
		log.Debug(ctx.Err())
		return nil, nil

	default:
		// TODO: Fetch from URLs (static and kernel args):
		// 		 https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
		// 		 http://169.254.169.254/latest/user-data
		//		 https://cloudinit.readthedocs.io/en/latest/reference/datasources/nocloud.html

		if data, err := getFromBlockDevices(ctx); err != nil {
			log.Error(err)
			return nil, err
		} else if data != nil {
			return unmarshal(data)
		}
		return nil, nil
	}
}
