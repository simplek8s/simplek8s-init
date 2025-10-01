package initrd

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/jlsalvador/simplek8s/pkg/cp"
	log "github.com/sirupsen/logrus"
)

// TODO: Replace this func by systemd tmpfiles.d
func populateInitrdWithFiles(output string) error {
	log.WithFields(log.Fields{
		"output": output,
	}).Debug("start")
	defer log.Debug("end")

	// Copy each src into dst
	for _, tbc := range []struct {
		src string
		dst string
		opt *cp.CopyOptions
	}{
		{
			src: "/usr",
			dst: filepath.Join(output, "/usr"),
			opt: &cp.CopyOptions{
				Exclude: []*regexp.Regexp{
					regexp.MustCompile(`/usr/share/factory/sysroot`),
				},
				PreserveAll: true,
				Overwrite:   true,
			},
		},
		{
			src: "/etc/ssl/certs",
			dst: filepath.Join(output, "/usr/share/factory/etc/ssl/certs"),
			opt: &cp.CopyOptions{
				PreserveAll: true,
				Overwrite:   true,
			},
		},
		{
			src: "/usr/share/factory/sysroot",
			dst: output,
			opt: &cp.CopyOptions{
				PreserveAll: true,
				Overwrite:   true,
			},
		},
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

	// Creates the `usr/local` directory because the future mount point `/usr`
	// will be Read-Only, so `/usr/local` must be created before.
	if err := os.MkdirAll(filepath.Join(output, "/usr/local"), 0755); err != nil {
		log.Error(err)
		return err
	}
	// Creates the `tmp` directory because systemd >=253 does not create this
	// directory before hand.
	if err := os.MkdirAll(filepath.Join(output, "/tmp"), 1777); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func CmdPopulateInitrd(output string) error {
	log.WithFields(log.Fields{
		"output": output,
	}).Debug("start")
	defer log.WithField("end", "CmdPopulateInitrd").Debug("end")

	if err := populateInitrdWithFiles(output); err != nil {
		log.WithField("output", output).Error(err)
		return err
	}

	return nil
}
