package procfs

import (
	"os"
	"reflect"
	"testing"
)

func TestParsePartitions(t *testing.T) {

	tests := []struct {
		name    string
		data    []byte
		want    []Partitions
		wantErr bool
	}{
		{
			name: "ok",
			data: []byte(`major minor  #blocks  name

259        0  500107608 nvme0n1
259        1     262144 nvme0n1p1
259        2  499844423 nvme0n1p2
  8        0 3907018584 sda
  8        1 3907017543 sda1
  8       16 3907018584 sdb
  8       17 3906492416 sdb1
  8       18     524288 sdb2
  8       32 3907018584 sdc
`),
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
			data: []byte(`major minor  #blocks  name

259        0  500107608 nvme0n1
259        1     262144 nvme0n1p1
nvme0n1p2
  8        0 3907018584 sda

  8        1 3907017543 sda1
  816 3907018584 sdb
  8       17 3906492416 sdb1
  8       18     524288 sdb2
  8       32 3907018584 sdc
`),
			want:    []Partitions{},
			wantErr: true,
		},
	}

	tempDir := t.TempDir()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fname := tempDir + "/" + tc.name
			if err := os.WriteFile(fname, tc.data, 0644); err != nil {
				t.Error(err)
			}

			if got, err := parsePartitions(fname); !tc.wantErr && err != nil {
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

	t.Run("any file", func(t *testing.T) {
		if _, err := parsePartitions(""); err == nil {
			t.Error("expect err")
		}
	})
	t.Run("not a file", func(t *testing.T) {
		if _, err := parsePartitions(tempDir); err == nil {
			t.Error("expect err")
		}
	})
}
