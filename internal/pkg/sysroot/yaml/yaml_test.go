package yaml

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/openlyinc/pointy"
)

func Test_unmarshal(t *testing.T) {
	type args struct {
		yamlContent []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *SimpleK8s
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
			want: &SimpleK8s{
				Version: VERSION_1,
				Storage: &simpleK8sStorage{
					Files: []simpleK8sFiles{
						{
							Path: "/etc/systemd/networkd/50-en-static.network",
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
				t.Errorf("unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getYamlContent(t *testing.T) {

	// Disk
	tmpdir := t.TempDir()
	diskFilename := filepath.Join(tmpdir, "disk.img")
	diskSize := int64(1024 * 1024 * 100)
	diskImage, err := diskfs.Create(diskFilename, diskSize, diskfs.Raw)
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
	if yaml, err := getYamlContent([]string{diskFilename}); err != nil {
		t.Error(err)
	} else if yaml != nil {
		t.Log(yaml)
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
