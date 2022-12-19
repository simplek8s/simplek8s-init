// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package init

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coreos/go-systemd/v22/unit"
	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot/yaml"
	"github.com/jlsalvador/simplek8s/internal/pkg/systemdGenerator/templates"
	"github.com/jlsalvador/simplek8s/pkg/common"
	log "github.com/sirupsen/logrus"
)

type SystemdUnitsTmpl = map[string]struct {
	tmplName string
	tmplData any
}

// Returns default systemd services to boot for a non-persistent session (tmpfs).
func getUnitsByDefault() (SystemdUnitsTmpl, error) {
	log.Debug("start")
	defer log.Debug("end")

	units := SystemdUnitsTmpl{}
	units["sysroot.mount"] = struct {
		tmplName string
		tmplData any
	}{
		// Mount "/sysroot" (tmpfs)
		"assets/run/systemd/system/systemd.mount.go.tmpl",
		templates.TmplDataSystemdUnitMount{
			DefaultDependencies: false,
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			Where:               "/sysroot/",
			What:                "tmpfs",
			Type:                "tmpfs",
			Options:             "size=90%",
		},
	}
	units["sysroot-usr.mount"] = struct {
		tmplName string
		tmplData any
	}{
		// Mount "/sysroot/usr" (tmpfs)
		"assets/run/systemd/system/systemd.mount.go.tmpl",
		templates.TmplDataSystemdUnitMount{
			DefaultDependencies: false,
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			After:               []string{"sysroot.mount"},
			Requires:            []string{"sysroot.mount"},
			Where:               "/sysroot/usr/",
			What:                "tmpfs",
			Type:                "tmpfs",
			Options:             "size=90%",
		},
	}
	units["sysroot-var.mount"] = struct {
		tmplName string
		tmplData any
	}{
		// Mount "/sysroot/var" (tmpfs)
		"assets/run/systemd/system/systemd.mount.go.tmpl",
		templates.TmplDataSystemdUnitMount{
			DefaultDependencies: false,
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			After:               []string{"sysroot.mount"},
			Requires:            []string{"sysroot.mount"},
			Where:               "/sysroot/var/",
			What:                "tmpfs",
			Type:                "tmpfs",
			Options:             "size=90%",
		},
	}
	units["sysroot-etc.mount"] = struct {
		tmplName string
		tmplData any
	}{
		// Mount "/sysroot/etc" (bind "/sysroot/var/etc")
		"assets/run/systemd/system/systemd.mount.go.tmpl",
		templates.TmplDataSystemdUnitMount{
			DefaultDependencies: false,
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			After:               []string{"sysroot-var.mount"},
			Requires:            []string{"sysroot-var.mount"},
			Where:               "/sysroot/etc/",
			What:                "/sysroot/var/etc/",
			Type:                "none",
			Options:             "bind",
		},
	}
	units["sysroot-home.mount"] = struct {
		tmplName string
		tmplData any
	}{
		// Mount "/sysroot/home" (bind "/sysroot/var/home")
		"assets/run/systemd/system/systemd.mount.go.tmpl",
		templates.TmplDataSystemdUnitMount{
			DefaultDependencies: false,
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			After:               []string{"sysroot-var.mount"},
			Requires:            []string{"sysroot-var.mount"},
			Where:               "/sysroot/home/",
			What:                "/sysroot/var/home/",
			Type:                "none",
			Options:             "bind",
		},
	}
	units["sysroot-mnt.mount"] = struct {
		tmplName string
		tmplData any
	}{
		// Mount "/sysroot/mnt" (bind "/sysroot/var/mnt")
		"assets/run/systemd/system/systemd.mount.go.tmpl",
		templates.TmplDataSystemdUnitMount{
			DefaultDependencies: false,
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			After:               []string{"sysroot-var.mount"},
			Requires:            []string{"sysroot-var.mount"},
			Where:               "/sysroot/mnt/",
			What:                "/sysroot/var/mnt/",
			Type:                "none",
			Options:             "bind",
		},
	}
	units["sysroot-opt.mount"] = struct {
		tmplName string
		tmplData any
	}{
		// Mount "/sysroot/opt" (bind "/sysroot/var/opt")
		"assets/run/systemd/system/systemd.mount.go.tmpl",
		templates.TmplDataSystemdUnitMount{
			DefaultDependencies: false,
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			After:               []string{"sysroot-var.mount"},
			Requires:            []string{"sysroot-var.mount"},
			Where:               "/sysroot/opt/",
			What:                "/sysroot/var/opt/",
			Type:                "none",
			Options:             "bind",
		},
	}
	units["sysroot-root.mount"] = struct {
		tmplName string
		tmplData any
	}{
		// Mount "/sysroot/root" (bind "/sysroot/var/root")
		"assets/run/systemd/system/systemd.mount.go.tmpl",
		templates.TmplDataSystemdUnitMount{
			DefaultDependencies: false,
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			After:               []string{"sysroot-var.mount"},
			Requires:            []string{"sysroot-var.mount"},
			Where:               "/sysroot/root/",
			What:                "/sysroot/var/root/",
			Type:                "none",
			Options:             "bind",
		},
	}
	units["sysroot-usr-libexec.mount"] = struct {
		tmplName string
		tmplData any
	}{
		// Mount "/sysroot/usr/libexec" (bind "/sysroot/var/usr/libexec")
		"assets/run/systemd/system/systemd.mount.go.tmpl",
		templates.TmplDataSystemdUnitMount{
			DefaultDependencies: false,
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			After:               []string{"sysroot-usr.mount", "sysroot-var.mount"},
			Requires:            []string{"sysroot-usr.mount", "sysroot-var.mount"},
			Where:               "/sysroot/usr/libexec/",
			What:                "/sysroot/var/usr/libexec/",
			Type:                "none",
			Options:             "bind",
		},
	}
	units["sysroot-usr-local.mount"] = struct {
		tmplName string
		tmplData any
	}{
		// Mount "/sysroot/usr/local" (bind "/sysroot/var/usr/local")
		"assets/run/systemd/system/systemd.mount.go.tmpl",
		templates.TmplDataSystemdUnitMount{
			DefaultDependencies: false,
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			After:               []string{"sysroot-usr.mount", "sysroot-var.mount"},
			Requires:            []string{"sysroot-usr.mount", "sysroot-var.mount"},
			Where:               "/sysroot/usr/local/",
			What:                "/sysroot/var/usr/local/",
			Type:                "none",
			Options:             "bind",
		},
	}
	units["simplek8s-populate-sysroot.service"] = struct {
		tmplName string
		tmplData any
	}{
		// Populate "/sysroot" (mostly "/sysroot/usr" and a few for "/sysroot/etc")
		"assets/run/systemd/system/systemd.service.go.tmpl",
		templates.TmplDataSystemdUnitService{
			DefaultDependencies: false,
			Description:         "Populate /sysroot",
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "shutdown.target"},
			After: []string{
				"sysroot-usr.mount",
				"sysroot-var.mount",
				"sysroot-etc.mount",
				"sysroot-home.mount",
				"sysroot-mnt.mount",
				"sysroot-opt.mount",
				"sysroot-root.mount",
				"sysroot-usr-libexec.mount",
				"sysroot-usr-local.mount",
			},
			Requires: []string{
				"sysroot-usr.mount",
				"sysroot-var.mount",
				"sysroot-etc.mount",
				"sysroot-home.mount",
				"sysroot-mnt.mount",
				"sysroot-opt.mount",
				"sysroot-root.mount",
				"sysroot-usr-libexec.mount",
				"sysroot-usr-local.mount",
			},
			Type:            "oneshot",
			Restart:         "no",
			RemainAfterExit: true,
			ExecStart:       []string{"/usr/lib/simplek8s/init populate -stage=initrd -output=/sysroot"},
		},
	}
	units["simplek8s-remount-ro-sysroot-usr.service"] = struct {
		tmplName string
		tmplData any
	}{
		// Remount "/sysroot/usr" as Read-Only
		"assets/run/systemd/system/systemd.service.go.tmpl",
		templates.TmplDataSystemdUnitService{
			DefaultDependencies: false,
			Description:         "Remount /sysroot/usr as RO",
			Requires:            []string{"simplek8s-populate-sysroot.service"},
			Conflicts:           []string{"shutdown.target"},
			Before:              []string{"initrd-root-fs.target", "initrd-switch-root.service", "shutdown.target"},
			After:               []string{"simplek8s-populate-sysroot.service"},
			Type:                "oneshot",
			Restart:             "no",
			RemainAfterExit:     true,
			ExecStart:           []string{"/usr/bin/mount -o remount,ro /sysroot/usr"},
		},
	}
	return units, nil
}

func getYamlUnits(defaultUnits SystemdUnitsTmpl) (SystemdUnitsTmpl, error) {
	log.Debug("start")
	defer log.Debug("end")

	units := defaultUnits

	if yamlSimpleK8s, err := yaml.GetYamlSimpleK8s(); err != nil {
		log.Error(err)
		return nil, err
	} else if yamlSimpleK8s != nil && yamlSimpleK8s.Storage != nil && yamlSimpleK8s.Storage.Mounts != nil {
		log.WithField("mounts", yamlSimpleK8s.Storage.Mounts).Debug()
		for _, mount := range yamlSimpleK8s.Storage.Mounts {

			// patch `where` because switch root to sysroot
			where := mount.Where
			if string([]rune(where)[0:1]) == "/" {
				where = filepath.Join("/sysroot", where)
			}

			escapedWhat := unit.UnitNamePathEscape(mount.What)
			escapedWhere := unit.UnitNamePathEscape(where)
			unitName := fmt.Sprintf("%s.mount", escapedWhere)

			// if `what` is a device, binds mount unit to this device
			bindsTo := []string{}
			if string([]rune(mount.What)[0:1]) == "/" {
				bindsTo = []string{fmt.Sprintf("%s.device", escapedWhat)}
			}

			mType := "auto"
			if mount.Type != nil {
				mType = *mount.Type
			}

			options := "defaults"
			if mount.Options != nil {
				options = *mount.Options
			}

			defaultDependencies := false
			before := []string{"initrd-root-fs.target", "shutdown.target"}
			after := []string{fmt.Sprintf("blockdev@%s.target", escapedWhat)}
			requires := []string{}
			if v, ok := defaultUnits[unitName]; ok {
				if v, ok := v.tmplData.(templates.TmplDataSystemdUnitMount); ok {
					defaultDependencies = v.DefaultDependencies
					before = v.Before
					after = v.After
					requires = v.Requires
				}
			}

			units[unitName] = struct {
				tmplName string
				tmplData any
			}{
				tmplName: "assets/run/systemd/system/systemd.mount.go.tmpl",
				tmplData: templates.TmplDataSystemdUnitMount{
					DefaultDependencies: defaultDependencies,
					BindsTo:             bindsTo,
					Conflicts:           []string{"shutdown.target"},
					Before:              before,
					After:               after,
					Requires:            requires,
					Where:               where,
					What:                mount.What,
					Type:                mType,
					Options:             options,
				},
			}
		}
	}
	return units, nil
}

func Cmd(args []string) error {
	log.WithFields(log.Fields{
		"args": args,
	}).Debug("start")
	defer log.Debug("end")

	var units SystemdUnitsTmpl

	if defaultUnits, err := getUnitsByDefault(); err != nil {
		log.Error(err)
		return err
	} else {
		if units, err = getYamlUnits(defaultUnits); err != nil {
			log.Error(err)
			return err
		}
	}
	log.WithField("units", units).Debug()

	uid, gid := common.GetOwnUidGid()

	// Add transient units to systemd
	for name, tmpl := range units {
		// Generate systemd unit content
		content, err := common.RenderTemplate(templates.Templates, tmpl.tmplName, tmpl.tmplData)
		if err != nil {
			log.Error(err)
			return err
		}

		// Create systemd unit file
		if err := os.MkdirAll("/run/systemd/transient", 0755); err != nil {
			log.Error(err)
			return err
		}
		filePath := filepath.Join("/run/systemd/transient", name)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			log.Error(err)
			return err
		}

		// Create dependency
		if err := os.MkdirAll("/run/systemd/transient/initrd-root-fs.target.requires", 0755); err != nil {
			log.Error(err)
			return err
		}
		linkFilePath := filepath.Join("/run/systemd/transient/initrd-root-fs.target.requires", name)
		if err := common.CreateSymlink(linkFilePath, filePath, true, uid, gid, false); err != nil {
			log.Error(err)
			return err
		}
	}

	// Reload systemd
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "/usr/bin/systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
