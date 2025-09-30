package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot/yaml"
	"github.com/jlsalvador/simplek8s/pkg/common"
)

func MountEssentialMountpoints(where string) error {
	mountpoints := []struct {
		source string
		target string
		fstype string
	}{
		{"proc", "/proc", "proc"},
		{"sysfs", "/sys", "sysfs"},
		{"udev", "/dev", "devtmpfs"},
	}

	for _, mp := range mountpoints {
		target := filepath.Join(where, mp.target)
		if err := syscall.Mount(mp.source, target, mp.fstype, 0, ""); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// Mount essential mountpoints.
	if err := MountEssentialMountpoints("/"); err != nil {
		log.Fatal(err)
	}

	// Wait for block devices for simplek8s.yaml.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var yml *yaml.Config
	for yml != nil {
		var err error
		yml, err = yaml.GetConfig(ctx)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Create and mount /sysroot.
	//TODO: custom mountpoint from simplek8s.yaml
	newroot := "/sysroot"
	if err := os.MkdirAll(newroot, 0755); err != nil {
		log.Fatal(err)
	}
	if err := syscall.Mount("tmpfs", newroot, "tmpfs", 0, "size=90%"); err != nil {
		log.Fatal(err)
	}

	// Mount essential mountpoints in /sysroot.
	if err := MountEssentialMountpoints(newroot); err != nil {
		log.Fatal(err)
	}

	//TODO: Create and mount special bind mounts.

	//TODO: Populate /sysroot.

	//TODO: Pivot root to /sysroot.
	oldroot := "/.oldroot"
	if err := syscall.PivotRoot(newroot, filepath.Join(newroot, oldroot)); err != nil {
		log.Fatal(err)
	}
	if err := os.Chdir("/"); err != nil {
		log.Fatal(err)
	}
	if err := syscall.Unmount(oldroot, syscall.MNT_DETACH); err != nil {
		log.Fatal(err)
	}
	if err := os.Remove(oldroot); err != nil {
		log.Fatal(err)
	}

	// Execute next init.
	inits := []string{"/init", "/sbin/init"}
	for _, init := range inits {
		if !common.CheckFileExists(init) {
			continue
		}
		if err := syscall.Exec(init, []string{init}, os.Environ()); err != nil {
			log.Fatal(err)
		}
	}

	os.Exit(0)
}
