package cp

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"simplek8s/pkg/common"

	log "github.com/sirupsen/logrus"
)

type CopyOptions struct {
	// Overwrite destination files.
	Overwrite bool
	// Supports go:embed.
	Fsys fs.FS
	// Set the destionation directory mode (ex: 0755).
	// Set as 0 to copy perm from source.
	DirPerm fs.FileMode
	// Set the destionation file mode (ex: 0644).
	// Set as 0 to copy perm from source.
	FilePerm fs.FileMode
	// Set the destination user id owner.
	Uid int
	// Set the destination group id owner.
	Gid int
	// Exclude entries that match these regexp.
	Exclude []*regexp.Regexp

	PreserveAll    bool
	PreservePerm   bool
	PreserveATime  bool
	PreserveMTime  bool
	PreserveUid    bool
	PreserveGid    bool
	PreserveXAttrs bool
}

// Doc:
//   - https://github.com/moby/moby/blob/master/daemon/graphdriver/copy/copy.go
func copyEntry(srcPath string, src string, fi fs.FileInfo, dst string, opt *CopyOptions) error {
	log.WithFields(log.Fields{
		"srcPath": srcPath,
		"src":     src,
		"fi":      fi,
		"dst":     dst,
		"opt":     opt,
	}).Trace("start")
	defer log.Trace("end")

	// Skip exist entries if overwrite is false
	info, _ := os.Stat(dst)
	if !opt.Overwrite && info != nil {
		log.WithFields(log.Fields{
			"dst": dst,
		}).Debug("skip overwrite")
		return nil
	}

	fullname := filepath.Join(srcPath, src)
	mode := fi.Mode()
	perm := mode.Perm()

	var stat *syscall.Stat_t
	if opt.PreserveAll || opt.PreserveUid || opt.PreserveGid || opt.PreserveATime || opt.PreserveMTime {
		if s, ok := fi.Sys().(*syscall.Stat_t); !ok {
			log.WithFields(log.Fields{
				"fullname": fullname,
			}).Warn("cannot stat file, we cannot preserve: uid, gid, atime, mtime")
		} else {
			stat = s
		}
	}

	switch {
	case mode.IsDir():
		if opt.DirPerm != 0x0 {
			perm = opt.DirPerm
		}

		if err := os.MkdirAll(dst, perm); err != nil {
			log.WithFields(log.Fields{
				"dst":  dst,
				"perm": perm,
			}).Error(err)
			return err
		}
	case mode.IsRegular():
		if opt.FilePerm != 0x0 {
			perm = opt.FilePerm
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			log.WithFields(log.Fields{
				"dst": dst,
			}).Error(err)
			return err
		}

		// Remove possible exists file
		if common.IsPathExists(dst) {
			if err := os.Remove(dst); err != nil {
				log.WithFields(log.Fields{
					"dst": dst,
				}).Error(err)
				return err
			}
		}

		log.WithField("src", src).WithField("opt.Fsys", opt.Fsys).Trace()
		fSrc, err := opt.Fsys.Open(src)
		if err != nil {
			log.WithFields(log.Fields{
				"src": src,
			}).Error(err)
			return err
		}
		defer fSrc.Close()

		flag := os.O_WRONLY | os.O_CREATE | os.O_EXCL
		fDst, err := os.OpenFile(dst, flag, perm)
		if err != nil {
			log.WithFields(log.Fields{
				"dst":  dst,
				"flag": flag,
				"perm": perm,
			}).Error(err)
			return err
		}
		defer fDst.Close()

		if _, err := io.Copy(fDst, fSrc); err != nil {
			log.WithFields(log.Fields{
				"fDst": fDst,
				"fSrc": fSrc,
			}).Error(err)
			return err
		}
	case mode&os.ModeSymlink != 0:
		//TODO: Error when fs.FS is not supported

		target, err := os.Readlink(fullname)
		if err != nil {
			log.WithFields(log.Fields{
				"fullname": fullname,
			}).Error(err)
			return err
		}
		if err := common.CreateSymlink(dst, target, true, opt.Uid, opt.Gid, false); err != nil {
			log.WithFields(log.Fields{
				"taget": target,
				"dst":   dst,
			}).Error(err)
			return err
		}
	case mode&os.ModeNamedPipe != 0:
		fallthrough
	case mode&os.ModeSocket != 0:
		//TODO
		panic("unimplemented")
	case mode&os.ModeDevice != 0:
		//TODO
		panic("unimplemented")
	default:
		//TODO
		panic("unimplemented")
	}

	// Copy uid and gid
	if opt.PreserveUid || opt.PreserveGid || opt.Uid >= 0 || opt.Gid >= 0 {
		uid := opt.Uid
		gid := opt.Gid

		// Get current file uid and gid to remplace invalid values
		if stat != nil {
			if uid < 0 {
				uid = int(stat.Uid)
			}
			if gid < 0 {
				gid = int(stat.Gid)
			}
		}

		// Change uid and gid
		if uid < 0 || gid < 0 {
			log.WithFields(log.Fields{
				"fullname": fullname,
				"uid":      uid,
				"gid":      gid,
			}).Warn("cannot set chown")
		} else {
			if err := os.Lchown(dst, uid, gid); err != nil {
				log.WithFields(log.Fields{
					"dst": dst,
					"uid": uid,
					"gid": gid,
				}).Error(err)
				return err
			}
		}
	}

	// Copy atime & mtime
	if opt.PreserveATime || opt.PreserveMTime {
		if stat != nil {
			tv := []unix.Timeval{
				{Sec: stat.Atim.Sec, Usec: stat.Atim.Nsec},
				{Sec: stat.Mtim.Sec, Usec: stat.Mtim.Nsec},
			}
			if err := unix.Lutimes(dst, tv); err != nil {
				log.WithFields(log.Fields{
					"dst": dst,
					"tv":  tv,
				}).Error(err)
				return err
			}
		}
	}

	//TODO: Copy xattrs

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
		PreservePerm:   true,
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
		opt.PreservePerm = options.PreserveAll || options.PreservePerm
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

	cPath := strings.TrimLeft(path, root)
	cPath = strings.TrimLeft(cPath, string(filepath.Separator))
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
