package common

import (
	"fmt"
	"os"
)

/*
 * unused
 *
func IsStringInList(value string, list []string) bool {
	for _, v := range list {
		if value == v {
			return true
		}
	}
	return false
}
*/

func IsDir(path string) error {
	if len(path) == 0 {
		err := fmt.Errorf("path %q is empty", path)
		return err
	}

	if stat, err := os.Stat(path); err != nil {
		return err
	} else if !stat.IsDir() {
		err := fmt.Errorf("path %q is not a directory", path)
		return err
	}
	return nil
}
