package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jlsalvador/simplek8s/pkg/cp"
	"github.com/jlsalvador/simplek8s/pkg/linux"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/bootstrap"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func populate(output string) error {
	log.Debug("start")
	defer log.Debug("end")

	for _, tbc := range []struct {
		src string
		dst string
		opt *cp.CopyOptions
	}{
		{
			src: "/usr",
			dst: filepath.Join(output, "/usr"),
			opt: &cp.CopyOptions{
				PreserveAll: true,
				Overwrite:   true,
			},
		},
		{
			src: "/etc",
			dst: filepath.Join(output, "/etc"),
			opt: &cp.CopyOptions{
				PreserveAll: true,
				Overwrite:   true,
			},
		},
		// {
		// 	src: "/etc/ssl/certs",
		// 	dst: filepath.Join(output, "/usr/share/factory/etc/ssl/certs"),
		// 	opt: &cp.CopyOptions{
		// 		PreserveAll: true,
		// 		Overwrite:   true,
		// 	},
		// },
	} {
		if err := os.MkdirAll(tbc.dst, 0755); err != nil {
			log.Error(err)
			return err
		}
		if err := cp.CopyDir(tbc.src, tbc.dst, tbc.opt); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

func must(err error, errmsg string, msg string) {
	if err != nil {
		log.Fatal(errmsg, err)
	}
	log.Info(msg)
}

func main() {
	log.Debug("start")
	defer log.Debug("end")

	// Check if we are PID 1, warns if not.
	if os.Getpid() != 1 {
		log.Fatal("not PID 1")
	}

	// Enable debug logging.
	// log.SetLevel(log.DebugLevel)
	// log.SetReportCaller(true)

	// Retrive SimpleK8s bootstrap config.
	must(func() error {
		config, err := bootstrap.GetConfig()
		log.Info(config)
		return err
	}(), "can not fetch config", "fetch config success")

	//DEBUG: Drop to shell.
	// unix.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ())

	// Create and mount /sysroot.
	//TODO: custom mountpoint from simplek8s.yaml
	newroot := "/sysroot"
	must(os.MkdirAll(newroot, 0755), "can not mkdir "+newroot, "mkdir "+newroot+" ready")
	must(unix.Mount("tmpfs", newroot, "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, "size=90%,mode=755"), "can not mount "+newroot, "mount "+newroot+" ready")

	// Populate /sysroot.
	must(populate(newroot), "can not populate "+newroot, "populate of "+newroot+" ready")

	must(linux.MountPseudoFS(newroot), "can not mount pseudofs", "pseudofs rootfs ready")
	must(linux.CreateDeprecatedSymlinks(newroot), "can not create deprecated symlinks into "+newroot, "deprecated symlinks for "+newroot+" created")
	// must(unix.Mount(newroot, "/", "", unix.MS_MOVE, ""), "can not mount --move to "+newroot, "mount --move "+newroot+" / ready")

	must(unix.Chroot(newroot), "can not chroot into "+newroot, "chroot into "+newroot+" ready")
	// must(unix.Chdir("/"), "can not chdir into /", "chdir / ready")
	must(unix.Exec("/usr/sbin/init", []string{"/sbin/init"}, os.Environ()), "can not exec /sbin/init", "exec /sbin/init ready")
	fmt.Println("Exiting...")
	os.Exit(0)
}
