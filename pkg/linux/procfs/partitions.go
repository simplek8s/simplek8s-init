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
	// Open the partitions file.
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

	// Parse each line from file.
	partitions := []Partitions{}
	s := bufio.NewScanner(f)
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
			return nil, fmt.Errorf("can not parse line %q, from %q", line, procPartitionsFilePath)
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
