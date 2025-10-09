// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package sysroot

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// Creates the next symlinks:
//   - usr/bin  -> bin
//   - usr/sbin -> sbin
//   - usr/lib  -> lib
//   - lib      -> lib64
func CreateLegacySymlinks(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	for _, sl := range []struct {
		old string
		new string
	}{
		{"usr/bin", filepath.Join(where, "bin")},
		{"usr/sbin", filepath.Join(where, "sbin")},
		{"usr/lib", filepath.Join(where, "lib")},
		{"lib", filepath.Join(where, "lib64")},
	} {
		if err := os.Symlink(sl.old, sl.new); err != nil {
			uerr := fmt.Errorf("can not create symlink %s as %s", sl.old, sl.new)
			log.WithError(err).Error(uerr)
			return uerr
		}
	}
	return nil
}
