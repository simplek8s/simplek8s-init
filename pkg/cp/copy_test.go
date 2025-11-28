// copy_test.go
package cp

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// --- Helpers / fake types ---

// fakeFS always returns error on Open (used to test open error in copyRegular).
type fakeFS struct {
	err error
}

func (f *fakeFS) Open(name string) (fs.File, error) {
	return nil, f.err
}

// fakeFileInfo implements fs.FileInfo for controlled Mode() values.
type fakeFileInfo struct {
	name  string
	mode  fs.FileMode
	size  int64
	isDir bool
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return f.size }
func (f fakeFileInfo) Mode() fs.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (f fakeFileInfo) IsDir() bool        { return f.isDir }
func (f fakeFileInfo) Sys() any           { return nil }

// --- Utility: install safe mocks and restore them on cleanup ---

func installSafeMocks(t *testing.T) {
	origSysLchown := sysLchown
	origSysLutimes := sysLutimes
	origSysReadlink := sysReadlink
	origCommonCreateSymlink := commonCreateSymlink
	origGetUID := getUIDFromFileInfo
	origGetGID := getGIDFromFileInfo
	origGetTimes := getAccessModificationTimes

	// Safe mocks: return nil (success) for syscalls; getUID/GID default fail unless set by test.
	sysLchown = func(name string, uid, gid int) error { return nil }
	sysLutimes = func(name string, tv []unix.Timeval) error { return nil }
	sysReadlink = func(name string) (string, error) { return "", errors.New("mock readlink") }
	commonCreateSymlink = func(path string, target string, overwrite bool, uid int, gid int, hard bool) error {
		// create a small symlink on disk for integration tests that call actual file checks,
		// but avoid chown/lutimes since those are mocked.
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		// create symlink to target; if fails (windows), return nil only to avoid failing tests.
		if runtime.GOOS != "windows" {
			_ = os.Symlink(target, path)
		}
		return nil
	}
	// default UID/GID getters return false; tests that need values will override them.
	getUIDFromFileInfo = func(fi fs.FileInfo) (int, bool) { return -1, false }
	getGIDFromFileInfo = func(fi fs.FileInfo) (int, bool) { return -1, false }
	getAccessModificationTimes = func(fi fs.FileInfo) ([]unix.Timeval, bool) { return nil, false }

	t.Cleanup(func() {
		sysLchown = origSysLchown
		sysLutimes = origSysLutimes
		sysReadlink = origSysReadlink
		commonCreateSymlink = origCommonCreateSymlink
		getUIDFromFileInfo = origGetUID
		getGIDFromFileInfo = origGetGID
		getAccessModificationTimes = origGetTimes
	})
}

// --- Tests ---

func Test_getOptionsWithDefaults_nil_and_provided(t *testing.T) {
	installSafeMocks(t)

	// nil options -> root should be "." and Fsys non-nil
	root, opt := getOptionsWithDefaults(nil, "/some/src")
	if root != "." {
		t.Fatalf("expected root '.', got %q", root)
	}
	if opt == nil || opt.Fsys == nil {
		t.Fatalf("expected opt.Fsys != nil")
	}

	// provided options with Fsys set -> root must equal src argument and preserve flags propagate
	fsmap := os.DirFS(".")
	in := &CopyOptions{Fsys: fsmap, DirPerm: 0755, FilePerm: 0644, Uid: 10, Gid: 11, PreserveAll: true}
	root2, opt2 := getOptionsWithDefaults(in, "someRoot")
	if root2 != "someRoot" {
		t.Fatalf("expected root equal to src when Fsys provided")
	}
	if !opt2.PreserveATime || !opt2.PreserveMTime || !opt2.PreserveUid || !opt2.PreserveGid {
		t.Fatalf("PreserveAll didn't propagate")
	}
	if opt2.Uid != 10 || opt2.Gid != 11 {
		t.Fatalf("Uid/Gid not propagated")
	}
}

func Test_copyDir_and_copyRegular_success(t *testing.T) {
	installSafeMocks(t)

	src := t.TempDir()
	dst := t.TempDir()

	// create structure:
	// src/a.txt
	// src/sub/b.txt
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run CopyDir via Copy (src is directory)
	if err := Copy(src, dst, nil); err != nil {
		t.Fatalf("Copy dir failed: %v", err)
	}

	// verify copied files
	got, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	if err != nil {
		t.Fatalf("missing a.txt: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("a.txt content mismatch")
	}
	got, err = os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
	if err != nil {
		t.Fatalf("missing sub/b.txt: %v", err)
	}
	if string(got) != "world" {
		t.Fatalf("sub/b.txt content mismatch")
	}
}

func Test_CopyFile_exclude_regex(t *testing.T) {
	installSafeMocks(t)

	src := t.TempDir()
	dst := t.TempDir()

	if err := os.WriteFile(filepath.Join(src, "keep.txt"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "skip.txt"), []byte("no"), 0644); err != nil {
		t.Fatal(err)
	}

	re, _ := regexp.Compile("skip.txt")
	opt := &CopyOptions{
		Exclude: []*regexp.Regexp{re},
	}

	// CopyDir expects src directory
	if err := CopyDir(src, dst, opt); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	// keep.txt must exist, skip.txt must not
	if _, err := os.Stat(filepath.Join(dst, "keep.txt")); err != nil {
		t.Fatalf("keep.txt missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "skip.txt")); !os.IsNotExist(err) {
		t.Fatalf("skip.txt should be excluded but exists or unexpected err: %v", err)
	}
}

func Test_Copy_overwrite_skip_when_dst_exists_and_no_overwrite(t *testing.T) {
	installSafeMocks(t)

	td := t.TempDir()

	srcFile := filepath.Join(td, "srcfile.txt")
	dstFile := filepath.Join(td, "dstfile.txt")

	if err := os.WriteFile(srcFile, []byte("fromsrc"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstFile, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	// Call Copy: src is a file, Overwrite=false -> should skip
	if err := Copy(srcFile, dstFile, &CopyOptions{Overwrite: false}); err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}

	// dst must remain with original content
	b, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(b) != "existing" {
		t.Fatalf("dst was overwritten unexpectedly")
	}
}

func Test_copyRegular_open_src_error(t *testing.T) {
	installSafeMocks(t)

	// craft a fake FileInfo that is regular file
	fi := fakeFileInfo{name: "f", mode: 0644, size: 0, isDir: false}

	// use fakeFS that returns an error on Open
	f := &fakeFS{err: errors.New("open fail")}
	opt := &CopyOptions{Fsys: f, FilePerm: 0}

	// ensure parent dir exists for dst
	td := t.TempDir()
	dst := filepath.Join(td, "out.txt")

	// call copyRegular which uses opt.Fsys.Open
	err := copyRegular("somefile", dst, fi, opt)
	if err == nil {
		t.Fatalf("expected open fail error")
	}
}

func Test_copySymlink_readlink_error_and_success(t *testing.T) {
	installSafeMocks(t)

	src := t.TempDir()
	dst := t.TempDir()

	// prepare a real file and a symlink to it (on disk, readlink will work)
	target := filepath.Join(src, "target.txt")
	if err := os.WriteFile(target, []byte("TGT"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(src, "mylink")
	if err := os.Symlink("target.txt", link); err != nil {
		t.Fatal(err)
	}

	// 1) Force sysReadlink to fail
	origReadlink := sysReadlink
	sysReadlink = func(name string) (string, error) {
		return "", errors.New("fake readlink error")
	}
	// restore after subtest
	defer func() { sysReadlink = origReadlink }()

	opt := &CopyOptions{Uid: os.Getuid(), Gid: os.Getgid()}

	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	// call copyEntry -> should return the readlink error
	if err := copyEntry(src, "mylink", fi, filepath.Join(dst, "mylink"), opt); err == nil {
		t.Fatalf("expected readlink error")
	}

	// 2) restore sysReadlink to real Readlink and mock CreateSymlink to succeed
	sysReadlink = os.Readlink
	origCreate := commonCreateSymlink
	commonCreateSymlink = func(path, target string, overwrite bool, uid int, gid int, hard bool) error {
		// create only the symlink, avoid chown since lchown is mocked
		if runtime.GOOS != "windows" {
			_ = os.Symlink(target, path)
		}
		return nil
	}
	defer func() { commonCreateSymlink = origCreate }()

	err = copyEntry(src, "mylink", fi, filepath.Join(dst, "mylink"), opt)
	if err != nil {
		t.Fatalf("copyEntry symlink failed: %v", err)
	}
	// verify destination is a symlink (if platform supports)
	linkDst := filepath.Join(dst, "mylink")
	if runtime.GOOS != "windows" {
		if _, err := os.Lstat(linkDst); err != nil {
			t.Fatalf("dst symlink missing: %v", err)
		}
	}
}

func Test_preserveOwnership_various_cases(t *testing.T) {
	installSafeMocks(t)

	td := t.TempDir()
	srcFile := filepath.Join(td, "s.txt")
	if err := os.WriteFile(srcFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(srcFile)
	if err != nil {
		t.Fatal(err)
	}

	// Backup original getters & sysLchown
	origGetUID := getUIDFromFileInfo
	origGetGID := getGIDFromFileInfo
	origSysLchown := sysLchown

	// make getUID/GID return something
	getUIDFromFileInfo = func(fi fs.FileInfo) (int, bool) { return 1234, true }
	getGIDFromFileInfo = func(fi fs.FileInfo) (int, bool) { return 4321, true }

	called := struct{ uid, gid int }{-1, -1}
	sysLchown = func(path string, uid, gid int) error {
		called.uid = uid
		called.gid = gid
		return nil
	}

	t.Cleanup(func() {
		getUIDFromFileInfo = origGetUID
		getGIDFromFileInfo = origGetGID
		sysLchown = origSysLchown
	})

	// Case 1: PreserveUid and PreserveGid true -> should call lchown with values from FI
	opt := &CopyOptions{PreserveUid: true, PreserveGid: true, Uid: -1, Gid: -1}
	if err := preserveOwnership(fi, filepath.Join(td, "dst"), opt); err != nil {
		t.Fatalf("preserveOwnership returned error: %v", err)
	}
	if called.uid != 1234 || called.gid != 4321 {
		t.Fatalf("unexpected uid/gid used: %v", called)
	}

	// Case 2: explicit Uid/Gid override
	called.uid, called.gid = -1, -1
	opt2 := &CopyOptions{Uid: 7, Gid: 8}
	if err := preserveOwnership(fi, filepath.Join(td, "dst2"), opt2); err != nil {
		t.Fatalf("preserveOwnership override returned error: %v", err)
	}
	if called.uid != 7 || called.gid != 8 {
		t.Fatalf("override not used: %v", called)
	}

	// Case 3: lchown returns error -> propagate
	sysLchown = func(path string, uid, gid int) error { return errors.New("lchown fail") }
	if err := preserveOwnership(fi, filepath.Join(td, "dst3"), opt2); err == nil {
		t.Fatalf("expected error when lchown fails")
	}
}
