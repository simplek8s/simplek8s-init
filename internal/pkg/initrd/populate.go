package initrd

import (
	"path/filepath"

	"github.com/jlsalvador/simplek8s/pkg/cp"
	log "github.com/sirupsen/logrus"
)

func PopulateRoot(where string) error {
	log.Debug("start")
	defer log.Debug("end")

	for _, tbc := range []struct {
		src string
		dst string
		opt *cp.CopyOptions
	}{
		// {
		// 	src: "/etc/passwd",
		// 	dst: filepath.Join(where, "/etc/passwd"),
		// 	opt: &cp.CopyOptions{
		// 		PreserveAll: true,
		// 		Overwrite:   true,
		// 	},
		// },
		// {
		// 	src: "/etc/shadow",
		// 	dst: filepath.Join(where, "/etc/shadow"),
		// 	opt: &cp.CopyOptions{
		// 		PreserveAll: true,
		// 		Overwrite:   true,
		// 	},
		// },
		{
			src: "/usr",
			dst: filepath.Join(where, "/usr"),
			opt: &cp.CopyOptions{
				PreserveAll: true,
				Overwrite:   true,
			},
		},
		// {
		// 	src: "/etc",
		// 	dst: filepath.Join(where, "/etc"),
		// 	opt: &cp.CopyOptions{
		// 		Exclude: []*regexp.Regexp{
		// 			regexp.MustCompile(`^/etc/initrd-release$`),
		// 		},
		// 		PreserveAll: true,
		// 		Overwrite:   true,
		// 	},
		// },
		// {
		// 	src: "/etc/ssl/certs",
		// 	dst: filepath.Join(output, "/usr/share/factory/etc/ssl/certs"),
		// 	opt: &cp.CopyOptions{
		// 		PreserveAll: true,
		// 		Overwrite:   true,
		// 	},
		// },
	} {
		if err := cp.Copy(tbc.src, tbc.dst, tbc.opt); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}
