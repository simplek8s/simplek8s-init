package procfs

import (
	"bufio"
	"os"
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

func ParseMountsFromFile(file *os.File) ([]MountEntry, error) {
	var mounts []MountEntry
	scanner := bufio.NewScanner(file)
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
		mounts = append(mounts, entry)
	}
	return mounts, nil
}
