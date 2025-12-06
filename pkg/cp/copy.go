// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

package cp

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/simplek8s/simplek8s-init/pkg/common"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// Wrappers that can be mocked in tests.
var (
	commonCreateSymlink = common.CreateSymlink
	sysLchown           = os.Lchown
	sysLutimes          = unix.Lutimes
	sysReadlink         = os.Readlink
	getUIDFromFileInfo  = func(fi fs.FileInfo) (int, bool) {
		if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
			return int(stat.Uid), true
		}
		return -1, false
	}
	getGIDFromFileInfo = func(fi fs.FileInfo) (int, bool) {
		if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
			return int(stat.Gid), true
		}
		return -1, false
	}
	getAccessModificationTimes = func(fi fs.FileInfo) ([]unix.Timeval, bool) {
		if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
			return []unix.Timeval{
				{Sec: stat.Atim.Sec, Usec: stat.Atim.Nsec / 1000},
				{Sec: stat.Mtim.Sec, Usec: stat.Mtim.Nsec / 1000},
			}, true
		}
		return nil, false
	}
)

type CopyOptions struct {
	// Overwrite destination files.
	Overwrite bool
	// Supports go:embed.
	Fsys fs.FS
	// Set the destionation directory mode (ex: 0o755).
	// Set as 0 to copy perm from source.
	DirPerm fs.FileMode
	// Set the destionation file mode (ex: 0o644).
	// Set as 0 to copy perm from source.
	FilePerm fs.FileMode
	// Set the destination user id owner.
	// Set as -1 to copy from source.
	Uid int
	// Set the destination group id owner.
	// Set as -1 to copy from source.
	Gid int
	// Exclude entries that match these regexp.
	Exclude []*regexp.Regexp

	PreserveAll    bool
	PreserveATime  bool
	PreserveMTime  bool
	PreserveUid    bool
	PreserveGid    bool
	PreserveXAttrs bool
}

func copyDir(dst string, fi fs.FileInfo, opt *CopyOptions) error {
	perm := fi.Mode().Perm()
	if opt.DirPerm != 0 {
		perm = opt.DirPerm
	}
	if err := os.MkdirAll(dst, perm); err != nil {
		log.WithFields(log.Fields{"dst": dst, "perm": perm}).Error(err)
		return err
	}
	return nil
}

func copyRegular(src, dst string, fi fs.FileInfo, opt *CopyOptions) error {
	perm := fi.Mode().Perm()
	if opt.FilePerm != 0 {
		perm = opt.FilePerm
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		log.WithField("dst", dst).Error(err)
		return err
	}

	// Remove possible exists file.
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		log.WithField("dst", dst).Error(err)
		return err
	}

	// Open src to read.
	log.WithFields(log.Fields{
		"src":      src,
		"opt.Fsys": opt.Fsys,
	}).Trace()
	fSrc, err := opt.Fsys.Open(src)
	if err != nil {
		log.WithFields(log.Fields{"src": src}).Error(err)
		return err
	}
	defer fSrc.Close()

	// Open dst to write.
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	fDst, err := os.OpenFile(dst, flag, perm)
	if err != nil {
		log.WithFields(log.Fields{"dst": dst, "flag": flag, "perm": perm}).Error(err)
		return err
	}
	defer fDst.Close()

	// Copy from src to dst.
	if _, err := io.Copy(fDst, fSrc); err != nil {
		log.WithFields(log.Fields{"fDst": fDst, "fSrc": fSrc}).Error(err)
		return err
	}

	return nil
}

func copySymlink(fullname, dst string, opt *CopyOptions) error {
	target, err := sysReadlink(fullname)
	if err != nil {
		log.WithField("fullname", fullname).Error(err)
		return err
	}

	if err := commonCreateSymlink(dst, target, true, opt.Uid, opt.Gid, false); err != nil {
		log.WithFields(log.Fields{
			"target": target,
			"dst":    dst,
		}).Error(err)
		return err
	}

	return nil
}

func preserveOwnership(fi fs.FileInfo, dst string, opt *CopyOptions) error {
	var uid, gid = -1, -1

	if opt.Uid >= 0 {
		// Explicit override UID.
		uid = opt.Uid
	} else if opt.PreserveUid {
		// Copy UID from src.
		if u, ok := getUIDFromFileInfo(fi); ok {
			uid = u
		}
	}

	if opt.Gid >= 0 {
		// Explicit override GID.
		gid = opt.Gid
	} else if opt.PreserveGid {
		// Copy GID from src.
		if g, ok := getGIDFromFileInfo(fi); ok {
			gid = g
		}
	}

	if uid >= 0 || gid >= 0 {
		if err := sysLchown(dst, uid, gid); err != nil {
			log.WithFields(log.Fields{
				"dst": dst,
				"uid": uid,
				"gid": gid,
			}).Error(err)
			return err
		}
	}

	return nil
}

// Preserve atime and mtime.
func preserveTimestamps(fi fs.FileInfo, dst string, opt *CopyOptions) error {
	if !(opt.PreserveATime || opt.PreserveMTime) {
		return nil
	}

	tv, ok := getAccessModificationTimes(fi)
	if !ok {
		err := fmt.Errorf("cannot stat file %s for metadata preservation", fi.Name())
		log.WithField("fi", fi).Error(err)
		return err
	}

	if err := sysLutimes(dst, tv); err != nil {
		log.WithFields(log.Fields{
			"dst": dst,
			"tv":  tv,
		}).Error(err)
		return err
	}

	return nil
}

// Doc:
//   - https://github.com/moby/moby/blob/master/daemon/graphdriver/copy/copy.go
func copyEntry(srcPath string, src string, fi fs.FileInfo, dst string, opt *CopyOptions) error {
	log.WithFields(log.Fields{
		"srcPath": srcPath,
		"src":     src,
		"dst":     dst,
	}).Trace("start")
	defer log.Trace("end")

	// Early return: overwrite check.
	if !opt.Overwrite {
		if _, err := os.Stat(dst); err == nil {
			log.WithField("dst", dst).Debug("skip overwrite")
			return nil
		}
	}

	fullname := filepath.Join(srcPath, src)
	mode := fi.Mode()

	var err error
	switch {
	case mode.IsDir():
		err = copyDir(dst, fi, opt)

	case mode.IsRegular():
		err = copyRegular(src, dst, fi, opt)

	case mode&os.ModeSymlink != 0:
		err = copySymlink(fullname, dst, opt)

	default:
		err = fmt.Errorf("unsupported file type (%v) at %s", mode, fullname)
	}
	if err != nil {
		return err
	}

	// After writing, preserve metadata.
	if err := preserveOwnership(fi, dst, opt); err != nil {
		return err
	}
	if err := preserveTimestamps(fi, dst, opt); err != nil {
		return err
	}
	//TODO: Preserve xattrs.

	return nil
}

func getOptionsWithDefaults(options *CopyOptions, src string) (string, *CopyOptions) {
	var root string
	opt := &CopyOptions{
		Overwrite:      false,
		Fsys:           nil,
		DirPerm:        0,
		FilePerm:       0,
		Uid:            -1,
		Gid:            -1,
		Exclude:        []*regexp.Regexp{},
		PreserveAll:    false,
		PreserveATime:  false,
		PreserveMTime:  false,
		PreserveUid:    false,
		PreserveGid:    false,
		PreserveXAttrs: false,
	}
	if options != nil {
		opt.Overwrite = options.Overwrite
		if options.Fsys != nil {
			root = src
			opt.Fsys = options.Fsys
		}
		if options.DirPerm != 0 {
			opt.DirPerm = options.DirPerm
		}
		if options.FilePerm != 0 {
			opt.FilePerm = options.FilePerm
		}
		if options.Exclude != nil {
			opt.Exclude = options.Exclude
		}

		opt.Uid = options.Uid
		opt.Gid = options.Gid

		opt.PreserveAll = options.PreserveAll
		opt.PreserveATime = options.PreserveAll || options.PreserveATime
		opt.PreserveMTime = options.PreserveAll || options.PreserveMTime
		opt.PreserveUid = options.PreserveAll || options.PreserveUid
		opt.PreserveGid = options.PreserveAll || options.PreserveGid
		opt.PreserveXAttrs = options.PreserveAll || options.PreserveXAttrs
	}
	if opt.Fsys == nil {
		root = "."
		opt.Fsys = os.DirFS(src)
	}
	return root, opt
}

func copyFile(src string, dst string, fi fs.FileInfo, path string, root string, opt *CopyOptions) error {
	log.WithFields(log.Fields{
		"src":  src,
		"dst":  dst,
		"fi":   fi,
		"path": path,
		"root": root,
		"opt":  opt,
	}).Trace("start")
	defer log.Trace("end")

	cPath := strings.TrimPrefix(path, root)
	cPath = strings.TrimPrefix(cPath, string(filepath.Separator))
	cPath = filepath.Clean(cPath)
	dstFullname := filepath.Join(dst, cPath)

	// Exclude
	for _, r := range opt.Exclude {
		fullSrc := filepath.Join(src, path)
		if r.Match([]byte(fullSrc)) {
			log.WithFields(log.Fields{
				"r":       r,
				"fullSrc": fullSrc,
			}).Debug("exclude")
			return nil
		}
	}

	return copyEntry(src, path, fi, dstFullname, opt)
}

// Copy the whole `src` directory content into the directory `dst`
func CopyDir(src string, dst string, options *CopyOptions) error {
	log.WithFields(log.Fields{
		"src":     src,
		"dst":     dst,
		"options": options,
	}).Trace("start")
	defer log.Trace("end")

	root, opt := getOptionsWithDefaults(options, src)

	// Walk into `src` directory
	return fs.WalkDir(opt.Fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		fi, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(src, dst, fi, path, root, opt)
	})
}

func Copy(src string, dst string, options *CopyOptions) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return CopyDir(src, dst, options)
	}

	// src is a file.

	root, opt := getOptionsWithDefaults(options, filepath.Dir(src))
	relPath := filepath.Base(src)

	dstInfo, err := os.Stat(dst)
	if err == nil {
		// dst exists.
		if dstInfo.IsDir() {
			// copy src inside dst directory.
			return copyFile(filepath.Dir(src), dst, fi, relPath, root, opt)
		}
		// if dst is a file, overwrite it.
		return copyEntry(filepath.Dir(src), relPath, fi, dst, opt)
	}

	if os.IsNotExist(err) {
		// dst not exists, so dst must be a file
		return copyEntry(filepath.Dir(src), relPath, fi, dst, opt)
	}

	return err
}
