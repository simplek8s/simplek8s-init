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

package bootstrap

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/goccy/go-yaml"
	"go.openly.dev/pointy"

	"github.com/simplek8s/simplek8s-init/pkg/linux/mount"
)

func Test_simple_marshal(t *testing.T) {
	want := []byte(`version: "1"
users:
- name: user1
  password_hash: password1
  ssh_authorized_keys:
  - something
`)
	c := Config{
		Version: "1",
		Users: []User{
			{
				Name:         "user1",
				PasswordHash: pointy.String("password1"),
				SSHAuthorizedKeys: []string{
					"something",
				},
			},
		},
	}
	got, err := yaml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got: %s, want: %s", got, want)
	}
}

func Test_simple_unmarshal(t *testing.T) {
	want := &Config{
		Version: "1",
		Users: []User{
			{
				Name:         "user1",
				PasswordHash: pointy.String("password1"),
				SSHAuthorizedKeys: []string{
					"something",
				},
			},
			{
				Name:         "user2",
				PasswordHash: pointy.String("password2"),
				SSHAuthorizedKeys: []string{
					"something_more",
				},
			},
		},
	}
	yml := `version: "1"
users:
- name: user1
  password_hash: password1
  ssh_authorized_keys:
  - something
- name: user2
  passwordHash: password2
  sshAuthorizedKeys:
  - something_more
`
	got, err := unmarshal([]byte(yml))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got: %v, want: %v", got, want)
	}
}

func Test_unmarshal(t *testing.T) {
	type args struct {
		yamlContent []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "static ip network",
			args: args{
				yamlContent: []byte(`
storage:
  files:
    - path: /etc/systemd/network/50-en-static.network
      content: |
        [Match]
        Name=en*

        [Network]
        Address=192.168.1.50/24
        Gateway=192.168.1.1
        DNS=8.8.8.8
`),
			},
			want: &Config{
				Version: ConfigVersion1,
				Storage: &Storage{
					Files: []File{
						{
							Path: "/etc/systemd/network/50-en-static.network",
							Content: pointy.String(`[Match]
Name=en*

[Network]
Address=192.168.1.50/24
Gateway=192.168.1.1
DNS=8.8.8.8
`),
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshal(tt.args.yamlContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("error: %v, wantErr: %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got: %v, want: %v", got, tt.want)
			}
		})
	}
}

func Test_resolveFlags(t *testing.T) {
	tests := []struct {
		name      string
		opts      []string
		wantFlags mount.MountFlag
		wantData  string
	}{
		{
			name:      "defaults is a no-op like mount(8)",
			opts:      []string{"defaults"},
			wantFlags: 0,
			wantData:  "",
		},
		{
			name:      "known flags go to Flags, fs data goes to Data",
			opts:      []string{"rw", "relatime", "discard"},
			wantFlags: mount.MountFlagRelATime,
			wantData:  "discard",
		},
		{
			name:      "whitespace around options is ignored",
			opts:      []string{"rw", " relatime ", " discard"},
			wantFlags: mount.MountFlagRelATime,
			wantData:  "discard",
		},
		{
			name:      "empty entries are skipped",
			opts:      []string{"ro", "", "nosuid"},
			wantFlags: mount.MountFlagReadOnly | mount.MountFlagNoSUID,
			wantData:  "",
		},
		{
			name:      "unknown key=value passes through as fs data",
			opts:      []string{"uid=1000"},
			wantFlags: 0,
			wantData:  "uid=1000",
		},
		{
			name:      "unknown bare word passes through as fs data",
			opts:      []string{"noexec", "foo"},
			wantFlags: mount.MountFlagNoExec,
			wantData:  "foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFlags, gotData := resolveFlags(tt.opts)
			if gotFlags != tt.wantFlags {
				t.Errorf("resolveFlags() flags = %v, want %v", gotFlags, tt.wantFlags)
			}
			if gotData != tt.wantData {
				t.Errorf("resolveFlags() data = %q, want %q", gotData, tt.wantData)
			}
		})
	}
}

func Test_getYamlContent(t *testing.T) {
	// Disk
	tmpdir := t.TempDir()
	diskFilename := filepath.Join(tmpdir, "disk.img")
	diskSize := int64(1024 * 1024 * 100)
	diskImage, err := diskfs.Create(diskFilename, diskSize, diskfs.SectorSizeDefault)
	if err != nil {
		t.Fatal(err)
	}

	// Partition
	pTable := &gpt.Table{
		ProtectiveMBR: true,
		Partitions: []*gpt.Partition{
			{
				Type: gpt.EFISystemPartition,
				Size: uint64(1024 * 1024 * 90),
				Name: "EFI",
			},
		},
	}
	if err := diskImage.Partition(pTable); err != nil {
		t.Fatal(err)
	}

	// Filesystem
	filesystem, err := diskImage.CreateFilesystem(disk.FilesystemSpec{
		Partition:   1,
		FSType:      filesystem.TypeFat32,
		VolumeLabel: "EFI System",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Directory
	if err := filesystem.Mkdir("/EFI/simplek8s"); err != nil {
		t.Fatal(err)
	}

	// File
	if file, err := filesystem.OpenFile("/EFI/simplek8s/simplek8s.yaml", os.O_SYNC|os.O_CREATE|os.O_RDWR); err != nil {
		t.Fatal(err)
	} else {
		defer file.Close()
		if _, err := file.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
	}

	// Test yaml content
	if where, yaml, err := getYamlContent([]string{diskFilename}); err != nil {
		t.Error(err)
	} else if yaml != nil {
		t.Log(where, yaml)
	} else {
		t.Fatal("yaml content is empty")
	}

	//TODO: Add test cases.
	//
	// type args struct {
	// 	directories  []string
	// 	blockDevices []string
	// }
	// tests := []struct {
	// 	name    string
	// 	args    args
	// 	want    []byte
	// 	wantErr bool
	// }{
	// 	{
	// 		name:    "disk with partitions, root path",
	// 		wantErr: false,
	// 	},
	// 	{
	// 		name:    "disk without partitions, root path",
	// 		wantErr: false,
	// 	},
	// 	{
	// 		name:    "disk efi partition, root path",
	// 		wantErr: false,
	// 	},
	// 	{
	// 		name:    "disk efi partition, simplek8s path",
	// 		wantErr: false,
	// 	},
	// 	{
	// 		name:    "disk efi partition, efi path",
	// 		wantErr: false,
	// 	},
	// 	{
	// 		name:    "disk efi partition, efi/simplek8s path",
	// 		wantErr: false,
	// 	},
	// 	{
	// 		name:    "disk efi partition, boot path",
	// 		wantErr: false,
	// 	},
	// 	{
	// 		name:    "disk efi partition, boot/simplek8s path",
	// 		wantErr: false,
	// 	},
	// }
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		got, err := getYamlContent(tt.args.directories, tt.args.blockDevices)
	// 		if (err != nil) != tt.wantErr {
	// 			t.Errorf("getYamlContent() error = %v, wantErr %v", err, tt.wantErr)
	// 			return
	// 		}
	// 		if !reflect.DeepEqual(got, tt.want) {
	// 			t.Errorf("getYamlContent() = %v, want %v", got, tt.want)
	// 		}
	// 	})
	// }
}
