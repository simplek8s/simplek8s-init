package initrd

import (
	"path/filepath"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
	"github.com/jlsalvador/simplek8s/internal/pkg/systemdGenerator/templates"
	log "github.com/sirupsen/logrus"
)

// Generate systemd units
func initrdSystemdUnits(sr *sysroot.Sysroot, generatorDir string) error {
	log.WithField("start", "initrdSystemdUnits").Debug()
	defer log.WithField("stop", "initrdSystemdUnits").Debug()

	for _, unit := range []struct {
		name     string
		tmplName string
		tmplData any
	}{
		{
			// Mount /sysroot (tmpfs)
			name:     "sysroot.mount",
			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitMount{
				DefaultDependencies: false,
				Before:              []string{"initrd-root-fs.target"},
				After:               []string{},
				Where:               "/sysroot/",
				What:                "tmpfs",
				Type:                "tmpfs",
				Options:             "size=90%",
			},
		},
		{
			// Mount /sysroot/usr (tmpfs)
			name:     "sysroot-usr.mount",
			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitMount{
				DefaultDependencies: false,
				Before:              []string{"initrd-root-fs.target"},
				After:               []string{"sysroot.mount"},
				Where:               "/sysroot/usr/",
				What:                "tmpfs",
				Type:                "tmpfs",
				Options:             "size=90%",
			},
		},
		{
			// Populate /sysroot (mostly /sysroot/usr)
			name:     "simplek8s-populate-sysroot.service",
			tmplName: "assets/run/systemd/system/systemd.service.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitService{
				Description: "Populate /sysroot",
				Before:      []string{"initrd-root-fs.target"},
				After:       []string{"sysroot-usr.mount"},
				Type:        "oneshot",
				ExecStart:   []string{"/usr/lib/simplek8s/init populate --stage initrd --output /sysroot"},
			},
		},
		{
			// Remount /sysroot as Read-Only
			name:     "sysroot-usr-remount-ro.service",
			tmplName: "assets/run/systemd/system/systemd.service.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitService{
				Description: "Remount /sysroot/usr as RO",
				Before:      []string{"initrd-root-fs.target"},
				After:       []string{"simplek8s-populate-sysroot.service"},
				Type:        "oneshot",
				ExecStart:   []string{"/usr/bin/mount -o remount,ro /sysroot/usr"},
			},
		},
	} {
		// Generate unit content
		content, err := common.RenderTemplate(
			templates.Templates,
			unit.tmplName,
			unit.tmplData,
		)
		if err != nil {
			log.WithFields(log.Fields{
				"sr":           sr,
				"generatorDir": generatorDir,
				"tmplName":     unit.tmplName,
				"tmplData":     unit.tmplData,
			}).Error(err)
			return err
		}

		// Append systemd units
		sr.Files = append(sr.Files, sysroot.File{
			Overwrite: true,
			Filename:  filepath.Join(generatorDir, unit.name),
			Content:   content,
			Mode:      0644,
			Uid:       0,
			Gid:       0,
		})

		// Enable unit
		sr.Links = append(sr.Links, sysroot.Link{
			Overwrite: true,
			Path:      filepath.Join(generatorDir, "initrd-root-fs.target.requires", unit.name),
			Target:    filepath.Join(generatorDir, unit.name),
			Uid:       0,
			Gid:       0,
			Hard:      false,
		})
	}
	return nil
}

func CmdSystemdGeneratorInitrd(sr *sysroot.Sysroot, generatorDir string) error {
	log.WithFields(log.Fields{
		"start":        "CmdSystemdGeneratorInitrd",
		"generatorDir": generatorDir,
	}).Debug()
	defer log.WithField("end", "CmdSystemdGeneratorInitrd").Debug()

	if err := initrdSystemdUnits(sr, generatorDir); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
