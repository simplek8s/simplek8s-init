package initrd

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/jlsalvador/simplek8s/pkg/common/copy"
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
		opt *copy.CopyOptions
	}{
		{
			src: "/usr",
			dst: filepath.Join(output, "/usr"),
			opt: &copy.CopyOptions{
				Exclude: []*regexp.Regexp{
					regexp.MustCompile(`/usr/share/factory/sysroot`),
				},
				PreserveAll: true,
			},
		},
		{
			src: "/etc/ssl/certs",
			dst: filepath.Join(output, "/usr/share/factory/etc/ssl/certs"),
			opt: &copy.CopyOptions{
				PreserveAll: true,
			},
		},
		{
			src: "/usr/share/factory/sysroot",
			dst: output,
			opt: &copy.CopyOptions{
				PreserveAll: true,
			},
		},
	} {
		if err := os.MkdirAll(tbc.dst, 0755); err != nil {
			log.Error(err)
			return err
		}
		if err := copy.CopyDir(tbc.src, tbc.dst, tbc.opt); err != nil {
			log.Error(err)
			return err
		}
	}

	// Creates `/usr/local` directory for the future mount point
	// `/usr` will be Read-Only, so `/usr/local` must be created before.
	if err := os.MkdirAll(filepath.Join(output, "/usr/local"), 0755); err != nil {
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
