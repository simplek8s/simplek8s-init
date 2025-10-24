package procfs

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"simplek8s/pkg/common"
	"simplek8s/pkg/linux/mount"
)

// getCustomCmdlineValue gets the value of a cmdline property from a custom location.
// It is used to override the cmdline file for testing purposes.
func getCustomCmdlineValue[T any](cmdlinePath string, key string, defaultValue T) (T, error) {
	cmdline, err := os.ReadFile(cmdlinePath)
	if err != nil {
		var zero T
		return zero, err
	}

	properties := strings.FieldsSeq(string(cmdline))
	for property := range properties {
		fields := strings.SplitN(property, "=", 2)
		if len(fields) == 2 && fields[0] == key {
			var result T
			switch any(result).(type) {
			case string:
				return any(fields[1]).(T), nil
			case int:
				v, err := strconv.Atoi(fields[1])
				if err != nil {
					return defaultValue, errors.New("invalid int value for key " + key)
				}
				return any(v).(T), nil
			case float64:
				v, err := strconv.ParseFloat(fields[1], 64)
				if err != nil {
					return defaultValue, errors.New("invalid float value for key " + key)
				}
				return any(v).(T), nil
			case bool:
				v, err := strconv.ParseBool(fields[1])
				if err != nil {
					return defaultValue, errors.New("invalid bool value for key " + key)
				}
				return any(v).(T), nil
			default:
				return defaultValue, errors.New("unsupported type")
			}
		} else if len(fields) == 1 && fields[0] == key {
			var result T
			switch any(result).(type) {
			case bool:
				return any(true).(T), nil
			default:
				return defaultValue, nil
			}
		}
	}

	return defaultValue, nil
}

// GetCmdlineValue returns the value of a cmdline property.
//
//   - If no cmdline file is found, it returns an error.
//   - If the key is not found, it returns the default value.
//   - If the key is found and its value is not empty, it returns the value.
//   - If the key is found, defaultValue type is boolean, and its value is
//     empty, it returns true.
//   - If the key is found, defaultValue type is not boolean, and its value is
//     empty, it returns the default value.
//   - If the key is found and its value is not a valid type, it returns an error.
//
// T could be one of the following types: string, int, float64 and bool.
func GetCmdlineValue[T any](key string, defaultValue T) (T, error) {
	if !common.IsPathExists("/proc/cmdline") {
		// Mount /proc to read "/proc/cmdline".
		if err := mount.Mount(mount.Mountpoints.Proc); err != nil {
			return defaultValue, err
		}
		defer mount.Unmount(mount.Mountpoints.Proc.Target, 0)
	}

	return getCustomCmdlineValue("/proc/cmdline", key, defaultValue)
}
