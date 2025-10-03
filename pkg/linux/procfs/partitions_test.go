package procfs

import (
	"io"
	"os"
	"reflect"
	"testing"
)

func TestParsePartitions(t *testing.T) {

	tests := []struct {
		name    string
		data    string
		want    []Partitions
		wantErr bool
	}{
		{
			name: "ok",
			data: `major minor  #blocks  name

259        0  500107608 nvme0n1
259        1     262144 nvme0n1p1
259        2  499844423 nvme0n1p2
  8        0 3907018584 sda
  8        1 3907017543 sda1
  8       16 3907018584 sdb
  8       17 3906492416 sdb1
  8       18     524288 sdb2
  8       32 3907018584 sdc
`,
			want: []Partitions{
				{259, 0, 500107608, "nvme0n1"},
				{259, 1, 262144, "nvme0n1p1"},
				{259, 2, 499844423, "nvme0n1p2"},
				{8, 0, 3907018584, "sda"},
				{8, 1, 3907017543, "sda1"},
				{8, 16, 3907018584, "sdb"},
				{8, 17, 3906492416, "sdb1"},
				{8, 18, 524288, "sdb2"},
				{8, 32, 3907018584, "sdc"},
			},
			wantErr: false,
		},
		{
			name: "bad",
			data: `major minor  #blocks  name

259        0  500107608 nvme0n1
259        1     262144 nvme0n1p1
nvme0n1p2
  8        0 3907018584 sda

  8        1 3907017543 sda1
  816 3907018584 sdb
  8       17 3906492416 sdb1
  8       18     524288 sdb2
  8       32 3907018584 sdc
`,
			want:    []Partitions{},
			wantErr: true,
		},
	}

	tempDir := t.TempDir()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fname := tempDir + "/" + tc.name
			file, err := os.OpenFile(fname, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				t.Errorf("failed to create temporary file: %v", err)
			}
			defer file.Close()
			if _, err := file.WriteString(tc.data); err != nil {
				t.Errorf("failed to seek to start of file: %v", err)
			}
			if _, err = file.Seek(0, io.SeekStart); err != nil {
				t.Errorf("failed to write mock data: %v", err)
			}

			if got, err := parsePartitionsFromFile(file); !tc.wantErr && err != nil {
				t.Errorf("unexpected err, %q", err)
			} else if tc.wantErr {
				if err == nil {
					t.Errorf("want err")
				}
			} else if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
