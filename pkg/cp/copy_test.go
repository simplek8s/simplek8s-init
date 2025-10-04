package cp

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/jlsalvador/simplek8s/pkg/common"
	log "github.com/sirupsen/logrus"
)

//go:embed test_assets/*
var test_assets embed.FS

type test_filesystem struct {
	dirname     string
	directories []struct {
		name string
	}
	files []struct {
		name    string
		content []byte
	}
	links []struct {
		name   string
		target string
	}
}

func checkDirs(directories []struct{ name string }, dst string) error {
	for _, d := range directories {
		dirname := filepath.Join(dst, d.name)
		if info, err := os.Stat(dirname); err != nil {
			return err
		} else if !info.IsDir() {
			return fmt.Errorf("%v expected to be a directory", dirname)
		}
	}
	return nil
}

func checkFiles(files []struct {
	name    string
	content []byte
}, dst string) error {
	for _, f := range files {
		fname := filepath.Join(dst, f.name)
		if content, err := os.ReadFile(fname); err != nil {
			return err
		} else if !bytes.Equal(f.content, content) {
			return fmt.Errorf("dst %v %v is not equal to src %v %v", fname, content, f.name, f.content)
		}
	}
	return nil
}

func checkLinks(links []struct {
	name   string
	target string
}, dst string) error {
	for _, l := range links {
		lname := filepath.Join(dst, l.name)
		if target, err := os.Readlink(lname); err != nil {
			return err
		} else if l.target != target {
			return fmt.Errorf("dst %v %v is not equal to src %v %v", lname, target, l.name, l.target)
		}
	}
	return nil
}

// TODO: Test Overwrite when file dst exists
func TestCopyDir(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	// Fill src for the testcases
	src := test_filesystem{
		dirname: t.TempDir(),
		directories: []struct{ name string }{
			{"1"},
			{"1/2"},
			{"1/2/3"},
		},
		files: []struct {
			name    string
			content []byte
		}{
			{"a", []byte("this is a test")},
			{"1/b", []byte("this is another test")},
			{"1/2/c", []byte{0, 1, 2, 3}},
		},
		links: []struct {
			name   string
			target string
		}{
			{"1/2/3/d", "../../b"},
			{"1/e", "../2/3"},
		},
	}
	// Create dirs
	for _, d := range src.directories {
		if err := os.MkdirAll(filepath.Join(src.dirname, d.name), 0755); err != nil {
			t.Error(err)
		}
	}
	// Create files
	for _, f := range src.files {
		if err := os.WriteFile(filepath.Join(src.dirname, f.name), f.content, 0644); err != nil {
			t.Error(err)
		}
	}
	// Create links
	for _, l := range src.links {
		if err := os.Symlink(l.target, filepath.Join(src.dirname, l.name)); err != nil {
			t.Error(err)
		}
	}
	t.Logf("src: %v", src.dirname)

	// Copy all to dst
	t.Run("all local filesystem without options", func(t *testing.T) {
		t.Parallel()
		dst := t.TempDir()
		t.Logf("dst: %v", dst)
		if err := CopyDir(src.dirname, dst, nil); err != nil {
			t.Error(err)
		}

		// Validate directories
		if err := checkDirs(src.directories, dst); err != nil {
			t.Error(err)
		}

		// Validate files
		if err := checkFiles(src.files, dst); err != nil {
			t.Error(err)
		}

		// Validate links
		if err := checkLinks(src.links, dst); err != nil {
			t.Error(err)
		}
	})

	// Test Exclude
	t.Run("exclude", func(t *testing.T) {
		t.Parallel()
		dst := t.TempDir()
		t.Logf("dst: %v", dst)

		uid, gid := common.GetOwnUidGid()

		if err := CopyDir(src.dirname, dst, &CopyOptions{
			Exclude: []*regexp.Regexp{
				regexp.MustCompile(`.+/3.*`),
			},
			Uid: uid,
			Gid: gid,
		}); err != nil {
			t.Error(err)
		}

		// Validate dst
		dirIncluded := filepath.Join(dst, "1/2")
		if _, err := os.Stat(dirIncluded); err != nil {
			t.Error(err)
		}
		linkIncluded := filepath.Join(dst, "1/2/c")
		if _, err := os.Stat(linkIncluded); err != nil {
			t.Error(err)
		}
		dirExcluded := filepath.Join(dst, "1/2/3")
		if _, err := os.Stat(dirExcluded); !os.IsNotExist(err) {
			t.Error(err)
		}
		linkExcluded := filepath.Join(dst, "1/2/3/d")
		if _, err := os.Readlink(linkExcluded); !os.IsNotExist(err) {
			t.Error(err)
		}
	})

	// Copy embed to dst
	t.Run("all embed minimun options for embeds", func(t *testing.T) {
		t.Parallel()
		dst := t.TempDir()
		t.Logf("dst: %v", dst)

		want := test_filesystem{
			directories: []struct{ name string }{
				{"1"},
				{"1/2"},
			},
			files: []struct {
				name    string
				content []byte
			}{
				{"a", []byte("hello world")},
				{"1/b", []byte("how are you?")},
				{"1/2/c", []byte("this is a third file")},
			},
		}

		uid, gid := common.GetOwnUidGid()

		if err := CopyDir("test_assets", dst, &CopyOptions{
			Fsys:     test_assets,
			Uid:      uid,
			Gid:      gid,
			DirPerm:  0755,
			FilePerm: 0644,
		}); err != nil {
			t.Error(err)
		}

		// Validate directories
		if err := checkDirs(want.directories, dst); err != nil {
			t.Error(err)
		}

		// Validate files
		if err := checkFiles(want.files, dst); err != nil {
			t.Error(err)
		}
	})
}

func TestCopyFile(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "f1.txt")
	srcContent := []byte("hello single file")
	if err := os.WriteFile(srcFile, srcContent, 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("copy to specific file path", func(t *testing.T) {
		t.Parallel()
		dstDir := t.TempDir()
		dstFile := filepath.Join(dstDir, "f1_copy.txt")

		if err := Copy(srcFile, dstFile, nil); err != nil {
			t.Error(err)
		}

		got, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, srcContent) {
			t.Errorf("content mismatch: got=%q want=%q", got, srcContent)
		}
	})

	t.Run("copy to existing directory", func(t *testing.T) {
		t.Parallel()
		dstDir := t.TempDir()

		if err := Copy(srcFile, dstDir, nil); err != nil {
			t.Error(err)
		}

		dstFile := filepath.Join(dstDir, "f1.txt")
		got, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, srcContent) {
			t.Errorf("content mismatch: got=%q want=%q", got, srcContent)
		}
	})

	t.Run("copy to non-existent directory", func(t *testing.T) {
		t.Parallel()
		parentDir := t.TempDir()
		dstDir := filepath.Join(parentDir, "nested", "dir")
		dstFile := filepath.Join(dstDir, "f1.txt")

		if err := Copy(srcFile, dstFile, nil); err != nil {
			t.Error(err)
		}

		got, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, srcContent) {
			t.Errorf("content mismatch: got=%q want=%q", got, srcContent)
		}
	})
}
