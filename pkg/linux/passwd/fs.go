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

package passwd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"simplek8s/pkg/common"
)

// ReadShadowFile fetchs all shadows from filename (usually "/etc/shadow").
func ReadShadowFile(filename string) ([]Shadow, error) {
	var shadows []Shadow

	if err := common.ForEachLineOfFilepath(filename, func(line string) error {
		line = strings.TrimSpace(line)

		// skip empty lines
		if line == "" {
			return nil
		}

		var shadow Shadow
		if err := UnmarshalShadow(line, &shadow); err != nil {
			return fmt.Errorf("failed to unmarshal shadow: %w", err)
		}

		shadows = append(shadows, shadow)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("cannot parse file %s: %w", filename, err)
	}

	return shadows, nil
}

// WriteShadows safely writes the provided shadows to
// filename (usually /etc/shadow) using a temporary file and atomic rename to
// avoid corruption.
func WriteShadows(filename string, shadows []Shadow) error {
	// Create a temporary file in the same directory.
	base := filepath.Base(filename)
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, "."+base+".tmp*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}

	tmpName := tmpFile.Name()

	// Ensure temporary file is closed and removed on error.
	defer func() {
		tmpFile.Close()
		os.Remove(tmpName)
	}()

	// Write each Shadow to temporary file.
	for _, shadow := range shadows {
		line, err := shadow.Marshal()
		if err != nil {
			return fmt.Errorf("cannot marshal shadow: %w", err)
		}
		if _, err := tmpFile.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("cannot write to temp file: %w", err)
		}
	}

	// Ensure data is flushed to disk.
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("cannot sync temp file: %w", err)
	}

	// Close the temporary file.
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("cannot close temp file: %w", err)
	}

	// Preserve original file permissions and ownership if the file exists
	if info, err := os.Stat(filename); err == nil {
		os.Chmod(tmpName, info.Mode().Perm())
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			os.Lchown(tmpName, int(stat.Uid), int(stat.Gid))
		}
	}

	// Atomically rename temp file to the target filename.
	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("cannot rename temp file %s to %s: %w", tmpName, filename, err)
	}

	return nil
}
