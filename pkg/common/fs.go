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

package common

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// Returns true if path exists.
func IsPathExists(path string) bool {
	log.WithFields(log.Fields{
		"path": path,
	}).Trace("start")
	defer log.Trace("end")

	_, error := os.Stat(path)
	return !errors.Is(error, os.ErrNotExist)
}

// Returns true if path is a directory.
func IsDir(path string) bool {
	log.WithFields(log.Fields{
		"path": path,
	}).Trace("start")
	defer log.Trace("end")

	if len(path) == 0 {
		log.Debugf("path %q is empty", path)
		return false
	}

	if stat, err := os.Stat(path); err != nil {
		log.Debug(err)
		return false
	} else if !stat.IsDir() {
		log.Debugf("path %q is not a directory", path)
		return false
	}
	return true
}

// Read each line from a filepath and executes a fn func that receives the line.
func ForEachLineOfFilepath(filepath string, fn func(line string) error) error {
	log.WithFields(log.Fields{
		"filepath": filepath,
		"fn":       fn,
	}).Trace("start")
	defer log.Trace("end")

	// Open file for a reader.
	file, err := os.Open(filepath)
	if err != nil {
		log.Debug(err)
		return err
	}
	defer file.Close()

	return ForEachLineOfReader(file, fn)
}

// Read each line from a reader and executes a fn func that receives the line.
func ForEachLineOfReader(r io.Reader, fn func(line string) error) error {
	log.WithFields(log.Fields{
		"r":  r,
		"fn": fn,
	}).Trace("start")
	defer log.Trace("end")

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		textline := scanner.Text()
		if err := fn(textline); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// We can use a mock for it.
var osLchown = os.Lchown

func CreateSymlink(path string, target string, overwrite bool, uid int, gid int, hard bool) error {
	log.WithFields(log.Fields{
		"path":      path,
		"target":    target,
		"overwrite": overwrite,
		"uid":       uid,
		"gid":       gid,
		"hard":      hard,
	}).Trace("start")
	defer log.Trace("end")

	// Overwrite?
	if info, _ := os.Stat(path); info != nil {
		if !overwrite {
			// Don't overwrite exist file.
			return nil
		} else {
			// Remove exist file.
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	// Create destination directory.
	dirname := filepath.Dir(path)
	if err := os.MkdirAll(dirname, 0755); err != nil {
		log.Debug(err)
		return err
	}

	// Create hard or soft link.
	fn := os.Symlink
	if hard {
		fn = os.Link
	}
	if err := fn(target, path); err != nil {
		log.Debug(err)
		return err
	}

	// Set owner.
	if err := osLchown(path, uid, gid); err != nil {
		log.Debug(err)
		return err
	}

	return nil
}
