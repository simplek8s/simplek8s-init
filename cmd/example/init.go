package main

import (
	"log"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// must aborts if err != nil
func must(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}

func main() {
	newRoot := "/newroot"
	targetInit := "/sbin/init" // binary inside the new root to execute

	// Ensure newRoot exists
	must(os.MkdirAll(newRoot, 0755), "mkdir newroot")

	// Mount tmpfs as new root (for demo). In real life, you might mount ext4, squashfs, etc.
	must(unix.Mount("tmpfs", newRoot, "tmpfs", 0, "mode=755"), "mount tmpfs on newroot")

	// Create minimal dirs inside tmpfs (so /sbin/init exists later!)
	must(os.MkdirAll(newRoot+"/sbin", 0755), "mkdir sbin")
	// For demo, place a symlink to /bin/sh as init if nothing else exists
	if _, err := os.Stat(newRoot + "/sbin/init"); os.IsNotExist(err) {
		must(os.Symlink("/bin/sh", newRoot+"/sbin/init"), "symlink init -> /bin/sh")
	}

	// Move mount from /newroot to /
	must(unix.Mount(newRoot, "/", "", unix.MS_MOVE, ""), "mount --move")

	// Change root into /
	must(os.Chdir("/"), "chdir /")
	must(unix.Chroot("."), "chroot .")

	// Mount special filesystems in the new root
	must(os.MkdirAll("/proc", 0555), "mkdir proc")
	must(unix.Mount("proc", "/proc", "proc", 0, ""), "mount /proc")
	must(os.MkdirAll("/sys", 0555), "mkdir sys")
	must(unix.Mount("sysfs", "/sys", "sysfs", 0, ""), "mount /sys")
	must(os.MkdirAll("/dev", 0755), "mkdir dev")
	must(unix.Mount("devtmpfs", "/dev", "devtmpfs", 0, ""), "mount /dev")

	// Exec real init
	must(syscall.Exec(targetInit, []string{targetInit}, os.Environ()), "exec init")
}
