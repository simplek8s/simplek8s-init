package procfs

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Partitions struct {
	// The Major number of the device with this partition.
	Major int64
	// The Minor number of the device with this partition.
	Minor int64
	// Lists the number of physical disk Blocks contained in a particular partition.
	Blocks int64
	// The Name of the partition.
	Name string
}

func ParsePartitions() ([]Partitions, error) {
	fname := "/proc/partitions"
	return parsePartitions(fname)
}

func parsePartitions(procPartitionsFilePath string) ([]Partitions, error) {

	// open partitions file
	f, err := os.Open(procPartitionsFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if fs, err := f.Stat(); err != nil {
		return nil, err
	} else if fs.IsDir() {
		return nil, fmt.Errorf("%q is not a file", procPartitionsFilePath)
	}

	// for each line from file
	partitions := []Partitions{}
	s := bufio.NewScanner(f)
	for i := 0; s.Scan(); i++ {
		line := s.Text()

		if i == 0 { // skip header
			if line != "major minor  #blocks  name" {
				return nil, fmt.Errorf("unexpected header, got %q", line)
			}
			continue
		} else if i == 1 { // skip new line header separator
			if line != "" {
				return nil, fmt.Errorf("unexpected new line header separator, got %q", line)
			}
			continue

		}

		// parse fields
		fields := strings.Fields(line)
		if len(fields) != 4 {
			return nil, fmt.Errorf("can not parse line %q, from %q", line, procPartitionsFilePath)
		}
		major, _ := strconv.ParseInt(fields[0], 10, 64)
		minor, _ := strconv.ParseInt(fields[1], 10, 64)
		blocks, _ := strconv.ParseInt(fields[2], 10, 64)
		name := fields[3]

		// append new struct
		partitions = append(partitions, Partitions{
			Major:  major,
			Minor:  minor,
			Blocks: blocks,
			Name:   name,
		})
	}

	return partitions, nil
}
