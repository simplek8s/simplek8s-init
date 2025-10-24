// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package procfs

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Partitions struct {
	Major  int64  // Major number of the device with this partition.
	Minor  int64  // Minor number of the device with this partition.
	Blocks int64  // Number of physical disk blocks contained.
	Name   string // Name of the partition.
}

// Parse partitions from a file (normally /proc/partitions).
// The file should contain lines in the format:
//
// major minor  #blocks  name
//
//	259        0 1953514584 nvme0n1
//	259        4 1953513472 nvme0n1p1
func parsePartitionsFromFile(file *os.File) ([]Partitions, error) {
	// Parse each line from file.
	partitions := []Partitions{}
	s := bufio.NewScanner(file)
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
			return nil, fmt.Errorf("cannot parse line %q, from %q", line, file.Name())
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
		partitions = append(partitions, Partitions{
			Major:  major,
			Minor:  minor,
			Blocks: blocks,
			Name:   name,
		})
	}

	return partitions, nil
}
