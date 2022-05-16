package systemd

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	unit "github.com/coreos/go-systemd/v22/unit"
	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot/yaml"
	log "github.com/sirupsen/logrus"
)

//go:embed templates/*
var templates embed.FS

var initrdRootFsTargetRequiresPath string = "initrd-root-fs.target.requires"

func createPaths(generatorDir string) error {
	path := path.Join(generatorDir, initrdRootFsTargetRequiresPath)
	mode := fs.FileMode(0755)
	if err := os.MkdirAll(path, mode); err != nil {
		log.WithFields(log.Fields{
			"path": path,
			"mode": mode,
		}).Error(err)
		return err
	}
	return nil
}

type systemdUnitMount struct {
	After   []string
	Where   string
	What    string
	Type    string
	Options string
}

func processSimpleK8sMount(mounts []systemdUnitMount) error {
	if simpleK8s, err := yaml.GetYamlSimpleK8s(); err != nil {
		return err
	} else if simpleK8s != nil && simpleK8s.Storage != nil {
		for _, sk8sMount := range simpleK8s.Storage.Mounts {
			found := false
			for index, mount := range mounts {
				if mount.Where == sk8sMount.Where {
					found = true
					if sk8sMount.Type != nil {
						mount.Type = *sk8sMount.Type
					} else {
						mount.Type = "auto"
					}
					if sk8sMount.Options != nil {
						mount.Options = *sk8sMount.Options
					} else {
						mount.Options = "defaults"
					}
					mount.What = sk8sMount.What
					mount.Where = sk8sMount.Where
					mounts[index] = mount
					break
				}
			}
			if !found {
				newMount := systemdUnitMount{
					After:   []string{"sysroot-var.mount"},
					Where:   sk8sMount.Where,
					What:    sk8sMount.What,
					Type:    "auto",
					Options: "defaults",
				}
				if sk8sMount.Options != nil {
					newMount.Options = *sk8sMount.Options
				}
				if sk8sMount.Type != nil {
					newMount.Type = *sk8sMount.Type
				}
				mounts = append(mounts, newMount)
			}
		}
	}
	return nil
}

func writeTemplate(filename string, templates embed.FS, templateFilename string, data any) error {
	if err := common.WriteTemplate(filename, templates, templateFilename, data); err != nil {
		return err
	}

	basename := filepath.Base(filename)
	dirname := filepath.Dir(filename)

	// Create systemd link dependency
	oldname := path.Join("..", basename)
	newname := path.Join(dirname, initrdRootFsTargetRequiresPath, basename)
	if err := os.Symlink(oldname, newname); err != nil {
		log.WithFields(log.Fields{
			"oldname": oldname,
			"newname": newname,
		}).Error(err)
		return err
	}

	return nil
}

func writeSystemdUnitMounts(generatorDir string, mounts []systemdUnitMount) error {
	template := "templates/systemd.mount.go.tmpl"
	log.WithField("initMounts", mounts).Debug("writeSystemdUnitMounts")

	for _, mount := range mounts {
		// Prepend `/sysroot` because chroot
		if mount.What[0:1] == "/" { // Only applicable for path
			mount.What = path.Join("/sysroot", mount.What)
		}
		if mount.Where[0:1] == "/" { // Only applicable for path
			mount.Where = path.Join("/sysroot", mount.Where)
		}

		// Calculate systemd unit name
		escapedFolder := unit.UnitNameEscape(mount.Where)
		escapedFolder = strings.TrimLeft(escapedFolder, "-")
		output := fmt.Sprintf("%s.mount", escapedFolder)

		log.WithFields(log.Fields{
			"mount":  mount,
			"output": output,
		}).Debug("writeSystemdUnitMounts")

		if err := writeTemplate(filepath.Join(generatorDir, output), templates, template, mount); err != nil {
			log.WithFields(log.Fields{
				"generatorDir": generatorDir,
				"template":     template,
				"output":       output,
				"mount":        mount,
			}).Error(err)
			return err
		}
	}
	return nil
}

func writeSystemdUnitServiceInit(generatorDir string, isLive bool) error {
	template := "templates/simplek8s-init.service"
	output := "simplek8s-init.service"
	data := struct {
		IsLive bool
	}{
		IsLive: isLive,
	}
	if err := writeTemplate(filepath.Join(generatorDir, output), templates, template, data); err != nil {
		log.WithFields(log.Fields{
			"generatorDir": generatorDir,
			"template":     template,
			"output":       output,
		}).Error(err)
		return err
	}
	return nil
}

// Will creates systemd units that will mount and populate sysroot
// https://www.freedesktop.org/software/systemd/man/systemd.generator.html#Description
func SystemdGenerator(generatorDir string, earlyDir string, lateDir string) error {
	log.Debug("init systemd generator")

	mounts := []systemdUnitMount{
		{[]string{}, "/", "tmpfs", "tmpfs", "size=90%"},
		{[]string{"sysroot.mount"}, "/dev", "devtmpfs", "devtmpfs", "defaults"},
		{[]string{"sysroot-dev.mount"}, "/var", "tmpfs", "tmpfs", "size=90%"},
		{[]string{"sysroot-var.mount"}, "/etc", "/var/etc", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/home", "/var/home", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/mnt", "/var/mnt", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/opt", "/var/opt", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/root", "/var/root", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/usr/libexec", "/var/usr/libexec", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/usr/local", "/var/usr/local", "none", "bind"},
	}

	if err := common.IsDir(generatorDir); err != nil {
		return err
	}

	if err := createPaths(generatorDir); err != nil {
		return err
	}

	if err := processSimpleK8sMount(mounts); err != nil {
		return err
	}

	isLive := false
	for _, mount := range mounts {
		if mount.Where == "/" {
			if mount.What == "tmpfs" {
				isLive = true
			}
			break
		}
	}

	if err := writeSystemdUnitMounts(generatorDir, mounts); err != nil {
		return err
	}

	if err := writeSystemdUnitServiceInit(generatorDir, isLive); err != nil {
		return err
	}

	return nil
}
