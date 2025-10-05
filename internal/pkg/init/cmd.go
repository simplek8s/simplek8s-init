// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package init

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	//"strings"

	"github.com/coreos/go-systemd/v22/unit"
	//"github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
	"github.com/jlsalvador/simplek8s/internal/pkg/systemdGenerator/templates"
	"github.com/jlsalvador/simplek8s/pkg/common"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/bootstrap"
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

// var reIpV4 = regexp.MustCompile(`^((?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.?){4})`)
// var reIpV6 = regexp.MustCompile(`(?m)^([[:xdigit:]]{1,4}(?::[[:xdigit:]]{1,4}){7}|::|:(?::[[:xdigit:]]{1,4}){1,6}|[[:xdigit:]]{1,4}:(?::[[:xdigit:]]{1,4}){1,5}|(?:[[:xdigit:]]{1,4}:){2}(?::[[:xdigit:]]{1,4}){1,4}|(?:[[:xdigit:]]{1,4}:){3}(?::[[:xdigit:]]{1,4}){1,3}|(?:[[:xdigit:]]{1,4}:){4}(?::[[:xdigit:]]{1,4}){1,2}|(?:[[:xdigit:]]{1,4}:){5}:[[:xdigit:]]{1,4}|(?:[[:xdigit:]]{1,4}:){1,6}:)`)

func getUnitsFromBootstrapConfig(config *bootstrap.Config, defaultUnits SystemdUnitsTmpl) (SystemdUnitsTmpl, error) {
	log.Debug("start")
	defer log.Debug("end")

	units := defaultUnits

	if config != nil && config.Storage != nil && config.Storage.Mounts != nil {
		log.WithField("mounts", config.Storage.Mounts).Debug()
		for _, mount := range config.Storage.Mounts {

			where := mount.Where
			if string([]rune(mount.Where)[0:1]) == "/" {
				// Patch `where` because switch root to sysroot
				where = filepath.Join("/sysroot", mount.Where)
			}

			escapedWhere := unit.UnitNamePathEscape(where)
			unitName := fmt.Sprintf("%s.mount", escapedWhere)
			escapedWhat := unit.UnitNamePathEscape(mount.What)
			defaultDependencies := false
			requires := []string{}
			bindsTo := []string{}
			before := []string{"initrd-root-fs.target", "shutdown.target"}
			after := []string{fmt.Sprintf("blockdev@%s.target", escapedWhat)}
			mType := "auto"
			options := "defaults"

			// Check `mount.What` in order to patch other fields
			if string([]rune(mount.What)[0:1]) == "/" {
				// If `what` is a device, binds mount unit to this device
				bindsTo = append(bindsTo, fmt.Sprintf("%s.device", escapedWhat))
				/*
					} else if reIpV4.MatchString(mount.What) || reIpV6.MatchString(mount.What) {
						// // Requires network if `mount.What` is an address
						// requires = append(requires, "systemd-networkd.service")
						// after = append(after, "systemd-networkd.service")

						// In order to boot from a network device, we need to setup the
						// interfaces, so we'll start `systemd-networkd.service` before
						// `sysroot.mount` and we'll stop it before switch to `/sysroot`.
						//
						// - Start systemd-networkd before "sysroot.mount".
						// - Stop systemd-networkd with the same requirements that starts `simplek8s-populate-sysroot.service`
						units["simplek8s-temporal-systemd-networkd.service"] = struct {
							tmplName string
							tmplData any
						}{
							"assets/run/systemd/system/systemd.service.go.tmpl",
							templates.TmplDataSystemdUnitService{
								DefaultDependencies: false,
								Description:         "Temporal systemd-networkd",
								Conflicts:           []string{"shutdown.target"},
								Before:              []string{"initrd-root-fs.target", "shutdown.target", "sysroot.mount"},
								Type:                "oneshot",
								Restart:             "on-failure",
								RemainAfterExit:     true,
								ExecStart: []string{
									"/usr/bin/systemctl restart systemd-networkd",
									"/usr/lib/systemd/systemd-networkd-wait-online --timeout=60",
									"/usr/bin/systemctl stop systemd-networkd.service systemd-networkd.socket",
								},
							},
						}
				*/
			}

			if mount.Type != nil {
				mType = *mount.Type
			}

			if mount.Options != nil {
				options = *mount.Options
			}

			if v, ok := defaultUnits[unitName]; ok {
				if v, ok := v.tmplData.(templates.TmplDataSystemdUnitMount); ok {
					defaultDependencies = v.DefaultDependencies
					before = append(before, v.Before...)
					after = append(after, v.After...)
					requires = append(requires, v.Requires...)
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

/*
func createSystemdNetworkFilesFromYaml(yamlSimpleK8s *yaml.SimpleK8s, units SystemdUnitsTmpl) (SystemdUnitsTmpl, error) {
	log.Debug("start")
	defer log.Debug("end")

	if yamlSimpleK8s != nil && yamlSimpleK8s.Storage != nil && yamlSimpleK8s.Storage.Files != nil {
		p := "/etc/systemd/network/"
		if err := os.MkdirAll(p, 0644); err != nil {
			log.Error(err)
			return units, err
		}
		for _, file := range yamlSimpleK8s.Storage.Files {
			if strings.HasPrefix(file.Path, p) {
				b := sysroot.GetBytesFromEncoding(file.Encoding, file.Content)
				log.WithFields(log.Fields{
					"path":     file.Path,
					"encoding": file.Encoding,
					"content":  file.Content,
					"bytes":    b,
				}).Debug("write file")
				if err := os.WriteFile(file.Path, b, 0644); err != nil {
					log.Error(err)
					return units, err
				}
			}
		}
	}
	return units, nil
}
*/

func Cmd(args []string) error {
	log.WithFields(log.Fields{
		"args": args,
	}).Debug("start")
	defer log.Debug("end")

	var units SystemdUnitsTmpl
	var err error

	if units, err = getUnitsByDefault(); err != nil {
		log.Error(err)
		return err
	} else if yamlSimpleK8s, err := bootstrap.GetConfig(); err != nil {
		log.Error(err)
		return err
	} else if units, err = getUnitsFromBootstrapConfig(yamlSimpleK8s, units); err != nil {
		log.Error(err)
		return err
		//} else if units, err = createSystemdNetworkFilesFromYaml(yamlSimpleK8s, units); err != nil {
		//	log.Error(err)
		//	return err
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
	if err := exec.CommandContext(ctx, "/usr/bin/systemctl", "daemon-reload").Run(); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
