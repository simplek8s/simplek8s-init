// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

package procfs

import (
	"bufio"
	"strconv"
	"strings"
)

// MountEntry represents a line in /proc/mounts
type MountEntry struct {
	Source  string   // device or source
	Target  string   // mount point
	FSType  string   // filesystem type
	Options []string // comma-separated options
	Dump    int      // dump frequency
	Pass    int      // fsck order
}

// Unescape escapes the \040 (space), \011 (tab), and \012 (newline) characters.
func unescape(s string) string {
	s = strings.ReplaceAll(s, "\\040", " ")
	s = strings.ReplaceAll(s, "\\011", "\t")
	s = strings.ReplaceAll(s, "\\012", "\n")
	return s
}

// ParseMounts parses the mountpoints information from /proc/mounts.
//
//	var mounts string
//
// should be a multi-line string where each line should contain lines in the format:
//
//	source target fstype options dump pass
func ParseMounts(mounts string) ([]MountEntry, error) {
	var ms []MountEntry
	reader := strings.NewReader(mounts)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		source := unescape(fields[0])
		target := unescape(fields[1])
		options := strings.Split(fields[3], ",")
		dump, _ := strconv.Atoi(fields[4])
		pass, _ := strconv.Atoi(fields[5])

		entry := MountEntry{
			Source:  source,
			Target:  target,
			FSType:  fields[2],
			Options: options,
			Dump:    dump,
			Pass:    pass,
		}
		ms = append(ms, entry)
	}
	return ms, nil
}

const MountsFilepath = "/proc/mounts"
