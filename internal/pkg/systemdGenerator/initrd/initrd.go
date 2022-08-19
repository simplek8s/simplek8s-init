package initrd

import (
	"fmt"
	"path/filepath"

	"github.com/coreos/go-systemd/unit"
	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot/yaml"
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
			// Mount "/sysroot" (tmpfs)
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
			// Mount "/sysroot/usr" (tmpfs)
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
			// Mount "/sysroot/var" (tmpfs)
			name:     "sysroot-var.mount",
			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitMount{
				DefaultDependencies: false,
				Before:              []string{"initrd-root-fs.target"},
				After:               []string{"sysroot.mount"},
				Where:               "/sysroot/var/",
				What:                "tmpfs",
				Type:                "tmpfs",
				Options:             "size=90%",
			},
		},
		{
			// Mount "/sysroot/etc" (bind "/sysroot/var/etc")
			name:     "sysroot-etc.mount",
			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitMount{
				DefaultDependencies: true,
				Before:              []string{"initrd-root-fs.target"},
				After:               []string{"sysroot-var.mount"},
				Where:               "/sysroot/etc/",
				What:                "/sysroot/var/etc/",
				Type:                "none",
				Options:             "bind",
			},
		},
		{
			// Mount "/sysroot/home" (bind "/sysroot/var/home")
			name:     "sysroot-home.mount",
			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitMount{
				DefaultDependencies: true,
				Before:              []string{"initrd-root-fs.target"},
				After:               []string{"sysroot-var.mount"},
				Where:               "/sysroot/home/",
				What:                "/sysroot/var/home/",
				Type:                "none",
				Options:             "bind",
			},
		},
		{
			// Mount "/sysroot/mnt" (bind "/sysroot/var/mnt")
			name:     "sysroot-mnt.mount",
			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitMount{
				DefaultDependencies: true,
				Before:              []string{"initrd-root-fs.target"},
				After:               []string{"sysroot-var.mount"},
				Where:               "/sysroot/mnt/",
				What:                "/sysroot/var/mnt/",
				Type:                "none",
				Options:             "bind",
			},
		},
		{
			// Mount "/sysroot/opt" (bind "/sysroot/var/opt")
			name:     "sysroot-opt.mount",
			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitMount{
				DefaultDependencies: true,
				Before:              []string{"initrd-root-fs.target"},
				After:               []string{"sysroot-var.mount"},
				Where:               "/sysroot/opt/",
				What:                "/sysroot/var/opt/",
				Type:                "none",
				Options:             "bind",
			},
		},
		{
			// Mount "/sysroot/root" (bind "/sysroot/var/root")
			name:     "sysroot-root.mount",
			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitMount{
				DefaultDependencies: true,
				Before:              []string{"initrd-root-fs.target"},
				After:               []string{"sysroot-var.mount"},
				Where:               "/sysroot/root/",
				What:                "/sysroot/var/root/",
				Type:                "none",
				Options:             "bind",
			},
		},
		{
			// Mount "/sysroot/usr/libexec" (bind "/sysroot/var/usr/libexec")
			name:     "sysroot-usr-libexec.mount",
			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitMount{
				DefaultDependencies: true,
				Before:              []string{"initrd-root-fs.target"},
				After:               []string{"sysroot-var.mount"},
				Where:               "/sysroot/usr/libexec/",
				What:                "/sysroot/var/usr/libexec/",
				Type:                "none",
				Options:             "bind",
			},
		},
		{
			// Mount "/sysroot/usr/local" (bind "/sysroot/var/usr/local")
			name:     "sysroot-usr-local.mount",
			tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
			tmplData: templates.TmplDataSystemdUnitMount{
				DefaultDependencies: true,
				Before:              []string{"initrd-root-fs.target"},
				After:               []string{"var.mount"},
				Where:               "/sysroot/usr/local/",
				What:                "/sysroot/var/usr/local/",
				Type:                "none",
				Options:             "bind",
			},
		},
		{
			// Populate "/sysroot" (mostly "/sysroot/usr")
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
			// Remount "/sysroot" as Read-Only
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

func initrdYamlMounts(sr *sysroot.Sysroot, generatorDir string) error {
	log.WithField("start", "initrdYamlMounts").Debug()
	defer log.WithField("end", "initrdYamlMounts").Debug()

	// Just write systemd mount units from yaml
	if yamlSimpleK8s, err := yaml.GetYamlSimpleK8s(); err != nil {
		log.Error(err)
		return err
	} else if yamlSimpleK8s != nil {
		for _, mount := range yamlSimpleK8s.Storage.Mounts {
			log.Debug(mount)

			// patch `where` because switch root to sysroot
			where := mount.Where
			if string([]rune(where)[0:1]) == "/" {
				where = filepath.Join("/sysroot", where)
			}

			mType := "auto"
			if mount.Type != nil {
				mType = *mount.Type
			}

			options := "defaults"
			if mount.Options != nil {
				options = *mount.Options
			}

			// content
			data := templates.TmplDataSystemdUnitMount{
				What:    mount.What,
				Where:   where,
				Type:    mType,
				Options: options,
			}
			content, err := common.RenderTemplate(templates.Templates, "assets/run/systemd/system/systemd.mount.go.tmpl", data)
			if err != nil {
				log.Error(err)
				return err
			}

			escaped := unit.UnitNamePathEscape(mount.Where)
			filename := fmt.Sprintf("sysroot-%s.mount", escaped)
			path := filepath.Join(generatorDir, filename)
			srFile := sysroot.File{
				Overwrite: true,
				Filename:  path,
				Content:   content,
				Mode:      0664,
				Uid:       0,
				Gid:       0,
			}
			log.Debug(srFile)

			// Replace or append
			found := false
			for i, file := range sr.Files {
				if file.Filename == path {
					found = true
					sr.Files[i] = srFile
					break
				}
			}
			if !found {
				sr.Files = append(sr.Files, srFile)
			}
		}
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

	if err := initrdYamlMounts(sr, generatorDir); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
