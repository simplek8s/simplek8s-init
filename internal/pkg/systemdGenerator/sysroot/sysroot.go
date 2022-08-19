package sysroot

import (
	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
	log "github.com/sirupsen/logrus"
)

// // Create and enable mount units
// func writeSystemdUnits(sr *sysroot.Sysroot, generatorDir string) error {
// 	log.WithField("start", "sysrootPrepareMountUnits").Debug()
// 	defer log.WithField("end", "sysrootPrepareMountUnits").Debug()

// 	for _, unit := range []struct {
// 		name     string
// 		tmplName string
// 		tmplData any
// 	}{
// 		{
// 			name:     "etc.mount",
// 			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
// 			tmplData: templates.TmplDataSystemdUnitMount{
// 				DefaultDependencies: true,
// 				Before:              []string{"local-fs.target"},
// 				After:               []string{"var.mount"},
// 				Where:               "/etc/",
// 				What:                "/var/etc/",
// 				Type:                "none",
// 				Options:             "bind",
// 			},
// 		},
// 		{
// 			name:     "home.mount",
// 			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
// 			tmplData: templates.TmplDataSystemdUnitMount{
// 				DefaultDependencies: true,
// 				Before:              []string{"local-fs.target"},
// 				After:               []string{"var.mount"},
// 				Where:               "/home/",
// 				What:                "/var/home/",
// 				Type:                "none",
// 				Options:             "bind",
// 			},
// 		},
// 		{
// 			name:     "mnt.mount",
// 			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
// 			tmplData: templates.TmplDataSystemdUnitMount{
// 				DefaultDependencies: true,
// 				Before:              []string{"local-fs.target"},
// 				After:               []string{"var.mount"},
// 				Where:               "/mnt/",
// 				What:                "/var/mnt/",
// 				Type:                "none",
// 				Options:             "bind",
// 			},
// 		},
// 		{
// 			name:     "opt.mount",
// 			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
// 			tmplData: templates.TmplDataSystemdUnitMount{
// 				DefaultDependencies: true,
// 				Before:              []string{"local-fs.target"},
// 				After:               []string{"var.mount"},
// 				Where:               "/opt/",
// 				What:                "/var/opt/",
// 				Type:                "none",
// 				Options:             "bind",
// 			},
// 		},
// 		{
// 			name:     "root.mount",
// 			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
// 			tmplData: templates.TmplDataSystemdUnitMount{
// 				DefaultDependencies: true,
// 				Before:              []string{"local-fs.target"},
// 				After:               []string{"var.mount"},
// 				Where:               "/root/",
// 				What:                "/var/root/",
// 				Type:                "none",
// 				Options:             "bind",
// 			},
// 		},
// 		{
// 			name:     "usr-libexec.mount",
// 			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
// 			tmplData: templates.TmplDataSystemdUnitMount{
// 				DefaultDependencies: true,
// 				Before:              []string{"local-fs.target"},
// 				After:               []string{"var.mount"},
// 				Where:               "/usr/libexec/",
// 				What:                "/var/usr/libexec/",
// 				Type:                "none",
// 				Options:             "bind",
// 			},
// 		},
// 		{
// 			name:     "usr-local.mount",
// 			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
// 			tmplData: templates.TmplDataSystemdUnitMount{
// 				DefaultDependencies: true,
// 				Before:              []string{"local-fs.target"},
// 				After:               []string{"var.mount"},
// 				Where:               "/usr/local/",
// 				What:                "/var/usr/local/",
// 				Type:                "none",
// 				Options:             "bind",
// 			},
// 		},
// 		{
// 			name:     "simplek8s-populate-root.service",
// 			tmplName: "assets/run/systemd/system/systemd.service.go.tmpl",
// 			tmplData: templates.TmplDataSystemdUnitService{
// 				Description: "Populate /",
// 				Before:      []string{"local-fs.target"},
// 				After: []string{
// 					"var.mount",
// 					"etc.mount",
// 					"home.mount",
// 					"mnt.mount",
// 					"opt.mount",
// 					"root.mount",
// 					"usr-libexec.mount",
// 					"usr-local.mount",
// 				},
// 				Type: "oneshot",
// 				ExecStart: []string{
// 					"/usr/bin/systemd-machine-id-setup",
// 					"/usr/lib/simplek8s/init populate --stage sysroot --output /",
// 					"/usr/bin/systemctl preset-all",
// 				},
// 			},
// 		},
// 	} {
// 		// Generate unit content
// 		content, err := common.RenderTemplate(
// 			templates.Templates,
// 			unit.tmplName,
// 			unit.tmplData,
// 		)
// 		if err != nil {
// 			log.WithFields(log.Fields{
// 				"sr":           sr,
// 				"generatorDir": generatorDir,
// 				"tmplName":     unit.tmplName,
// 				"tmplData":     unit.tmplData,
// 			}).Error(err)
// 			return err
// 		}

// 		// Write unit
// 		sr.Files = append(sr.Files, sysroot.File{
// 			Overwrite: false,
// 			Filename:  filepath.Join(generatorDir, unit.name),
// 			Content:   content,
// 			Mode:      0644,
// 			Uid:       0,
// 			Gid:       0,
// 		})

// 		// Enable unit
// 		sr.Links = append(sr.Links, sysroot.Link{
// 			Overwrite: false,
// 			Path:      filepath.Join(generatorDir, "local-fs.target.requires", unit.name),
// 			Target:    filepath.Join(generatorDir, unit.name),
// 			Uid:       0,
// 			Gid:       0,
// 			Hard:      false,
// 		})
// 	}
// 	return nil
// }

// func sysrootSimpleK8sYamlSystemdUnitsMount(sr *sysroot.Sysroot, generatorDir string) error {
// 	// Just write mounts from safe paths
// 	if yamlSimpleK8s, err := yaml.GetYamlSimpleK8s(); err != nil {
// 		log.Error(err)
// 		return err
// 	} else {
// 		if err := sr.FeedByYAML(yamlSimpleK8s); err != nil {
// 			log.Error(err)
// 			return err
// 		}

// 		// Filter systemd units for the generator step
// 		filteredFiles := []sysroot.File{}
// 		for _, f := range sr.Files {
// 			// Only accepts units that will write into `/run/systemd/`
// 			if matched, err := regexp.MatchString(`^/run/systemd/.+$`, f.Filename); err != nil {
// 				log.Error(err)
// 				return err
// 			} else if matched {
// 				filteredFiles = append(filteredFiles, f)
// 			}
// 		}
// 		log.WithFields(log.Fields{
// 			"files":         sr.Files,
// 			"filteredFiles": filteredFiles,
// 		}).Debug()
// 		sr.Files = filteredFiles
// 	}
// 	return nil
// }

func CmdSystemdGeneratorSysroot(sr *sysroot.Sysroot, generatorDir string) error {
	log.WithFields(log.Fields{
		"start":        "CmdSystemdGeneratorSysroot",
		"generatorDir": generatorDir,
	}).Debug()
	defer log.WithField("end", "CmdSystemdGeneratorSysroot").Debug()

	// if err := writeSystemdUnits(sr, generatorDir); err != nil {
	// 	log.Error(err)
	// 	return err
	// }

	// if err := sysrootSimpleK8sYamlSystemdUnitsMount(sr, generatorDir); err != nil {
	// 	log.Error(err)
	// 	return err
	// }

	return nil
}
