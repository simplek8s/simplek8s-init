package sysroot

import (
	"fmt"
	"path/filepath"

	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
	"github.com/jlsalvador/simplek8s/internal/pkg/systemdGenerator/templates"
	"github.com/jlsalvador/simplek8s/pkg/common"
	log "github.com/sirupsen/logrus"
)

// Create and enable systemd units
func writeSystemdUnits(sr *sysroot.Sysroot, generatorDir string) error {
	log.WithFields(log.Fields{
		"sr":           sr,
		"generatorDir": generatorDir,
	}).Debug("start")
	defer log.Debug("end")

	presetContent := ""
	for _, unit := range []struct {
		name     string
		tmplName string
		tmplData any
	}{
		{
			name:     "simplek8s-populate-root.service",
			tmplName: "assets/run/systemd/system/systemd.service.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitService{
				Description: "Populate /",
				Before:      []string{"local-fs.target"},
				After: []string{
					"var.mount",
					"etc.mount",
					"home.mount",
					"mnt.mount",
					"opt.mount",
					"root.mount",
					"usr-libexec.mount",
					"usr-local.mount",
				},
				Type: "oneshot",
				ExecStart: []string{
					"/usr/bin/systemd-machine-id-setup",
					"/usr/lib/simplek8s/init populate -stage=sysroot -output=/",
				},
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
			Path:      filepath.Join(generatorDir, "local-fs.target.requires", unit.name),
			Target:    filepath.Join(generatorDir, unit.name),
			Uid:       0,
			Gid:       0,
			Hard:      false,
		})

		// Disable systemd unit into the future preset "98-disable-transient.preset"
		presetContent += fmt.Sprintf("disable %s\n", unit.name)
	}

	// Add the transient preset to disable the above units
	sr.Files = append(sr.Files, sysroot.File{
		Overwrite: true,
		Filename:  filepath.Join(generatorDir, "../system-preset/98-disable-transient.preset"),
		Content:   []byte(presetContent),
		Mode:      0644,
		Uid:       0,
		Gid:       0,
	})

	return nil
}

func CmdSystemdGeneratorSysroot(sr *sysroot.Sysroot, generatorDir string) error {
	log.WithFields(log.Fields{
		"sr":           sr,
		"generatorDir": generatorDir,
	}).Debug("start")
	defer log.Debug("end")

	if err := writeSystemdUnits(sr, generatorDir); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
