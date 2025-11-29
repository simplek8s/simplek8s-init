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

package procfs

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

type Partitions struct {
	Major  int64  // Major number of the device with this partition.
	Minor  int64  // Minor number of the device with this partition.
	Blocks int64  // Number of physical disk blocks contained.
	Name   string // Name of the partition.
}

// ParsePartitions parses the partitions information from /proc/partitions.
//
//	var partitions string
//
// should be a multi-line string where each line should contain lines in the
// format:
//
//	 major minor  #blocks  name
//
//		259        0 1953514584 nvme0n1
//		259        4 1953513472 nvme0n1p1
func ParsePartitions(partitions string) ([]Partitions, error) {
	// Parse each line from file.
	parts := []Partitions{}

	reader := strings.NewReader(partitions)
	s := bufio.NewScanner(reader)
	for i := 0; s.Scan(); i++ {
		line := s.Text()

		if i == 0 { // Skip header.
			if line != "major minor  #blocks  name" {
				return nil, fmt.Errorf("unexpected header, got %q", line)
			}
			continue
		} else if i == 1 { // Skip "\n" header separator.
			if line != "" {
				return nil, fmt.Errorf("unexpected new line header separator, got %q", line)
			}
			continue
		}

		// Parse each fields.
		fields := strings.Fields(line)
		if len(fields) != 4 {
			return nil, fmt.Errorf("cannot parse line %d: %q", i+1, line)
		}
		major, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			return nil, err
		}
		minor, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return nil, err
		}
		blocks, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			return nil, err
		}
		name := fields[3]

		// Append the parsed line as new struct.
		parts = append(parts, Partitions{
			Major:  major,
			Minor:  minor,
			Blocks: blocks,
			Name:   name,
		})
	}

	return parts, nil
}
