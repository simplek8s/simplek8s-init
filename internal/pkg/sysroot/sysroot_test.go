package sysroot

import (
	"bytes"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"
)

func TestIsIntInList(t *testing.T) {
	tdd := []struct {
		list  []int
		entry int
		found bool
	}{
		{
			[]int{1, 2, 3},
			2,
			true,
		},
		{
			[]int{1, 2, 3},
			4,
			false,
		},
	}

	for _, tc := range tdd {
		if got := isIntInList(tc.entry, tc.list); got != tc.found {
			t.Errorf("list %v, entry %d, got %t, want %t", tc.list, tc.entry, got, tc.found)
		}
	}
}

func TestWriteFile(t *testing.T) {
	var uid, gid int
	if current, err := user.Current(); err != nil {
		t.Error(err)
	} else {
		var err error
		if uid, err = strconv.Atoi(current.Uid); err != nil {
			t.Error(err)
		}
		if gid, err = strconv.Atoi(current.Gid); err != nil {
			t.Error(err)
		}
	}

	t.Run("ok", func(t *testing.T) {
		tempDir := t.TempDir()
		name := filepath.Join(tempDir, "directory", "file")
		want := []byte("This is an example")

		if err := writeFile(name, want, 0644, uid, gid); err != nil {
			t.Error(err)
		}
		if output, err := os.ReadFile(name); err != nil {
			t.Error(err)
		} else if !bytes.Equal(output, want) {
			t.Errorf("got %q, want %q", output, want)
		}
	})

	t.Run("can not create directory", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tempDir, "directory"), []byte("i am not a directory"), 0644); err != nil {
			t.Error(err)
		}
		name := filepath.Join(tempDir, "directory", "file")
		want := []byte("must fail")

		if err := writeFile(name, want, 0644, uid, gid); err == nil {
			t.Errorf("expecting error, can not create directory overwritting file")
		}
	})

	t.Run("invalid perm", func(t *testing.T) {
		tempDir := t.TempDir()
		name := filepath.Join(tempDir, "file")
		want := []byte("must fail")

		if err := writeFile(name, want, 0, uid, gid); err == nil {
			t.Errorf("expecting error, invalid perm")
		}
	})

	t.Run("can not chown", func(t *testing.T) {
		tempDir := t.TempDir()
		name := filepath.Join(tempDir, "file")
		want := []byte("must fail")

		if err := writeFile(name, want, 0644, 4294967296, 4294967296); err == nil {
			t.Errorf("expecting error, can not chown")
		}
	})
}
