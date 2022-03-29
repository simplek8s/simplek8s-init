package systemd

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"strings"
	"text/template"

	unit "github.com/coreos/go-systemd/v22/unit"
	"github.com/jlsalvador/simplek8s/common"
	"github.com/jlsalvador/simplek8s/sysroot/yaml"
	log "github.com/sirupsen/logrus"
)

//go:embed templates/*
var templates embed.FS

var initrdRootFsTargetRequiresPath string = "initrd-root-fs.target.requires"

func createPaths(generatorDir string) error {
	path := generatorDir + initrdRootFsTargetRequiresPath
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

func writeSysrootTemplate(generatorDir string, templateFilename string, outputFilename string, data any) error {
	tmpl, err := template.ParseFS(templates, templateFilename)
	if err != nil {
		log.WithFields(log.Fields{
			"templates": templates,
			"tmpl":      tmpl,
		}).Error(err)
		return err
	}
	log.Debug(tmpl)

	content := new(bytes.Buffer)
	if err := tmpl.Execute(content, data); err != nil {
		log.WithFields(log.Fields{
			"tmpl": tmpl,
			"data": data,
		}).Error(err)
		return err
	}
	log.Debug(content)

	name := generatorDir + outputFilename
	mode := fs.FileMode(0644)
	if err := os.WriteFile(name, content.Bytes(), mode); err != nil {
		log.WithFields(log.Fields{
			"name":    name,
			"content": content.String(),
			"mode":    mode,
		}).Error(err)
		return err
	}

	oldname := ".." + outputFilename
	newname := generatorDir + initrdRootFsTargetRequiresPath + outputFilename
	if err := os.Symlink(oldname, newname); err != nil {
		log.WithFields(log.Fields{
			"oldname": oldname,
			"newname": newname,
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

func processSimpleK8sMount(generatorDir string, mounts []systemdUnitMount) error {
	if simpleK8s, err := yaml.GetYamlSimpleK8s(); err != nil {
		return err
	} else if simpleK8s != nil {
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

func writeSystemdUnitMounts(generatorDir string, mounts []systemdUnitMount) error {
	template := "templates/systemd.mount.go.tmpl"
	log.WithField("initMounts", mounts).Debug()
	for _, mount := range mounts {
		escapedFolder := unit.UnitNameEscape(mount.Where)
		escapedFolder = strings.TrimLeft(escapedFolder, "-")
		output := "/" + escapedFolder + ".mount"
		log.WithField("output", output).Debug()
		if err := writeSysrootTemplate(generatorDir, template, output, mount); err != nil {
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

func writeSystemdUnitServiceInit(generatorDir string) error {
	template := "templates/simplek8s-init.service"
	output := "/simplek8s-init.service"
	if err := writeSysrootTemplate(generatorDir, template, output, nil); err != nil {
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
func SystemdGenerator(generatorDir string) error {
	log.Debug("init systemd generator")

	mounts := []systemdUnitMount{
		{[]string{}, "/sysroot", "tmpfs", "tmpfs", "size=90%"},
		{[]string{"sysroot.mount"}, "/sysroot/var", "tmpfs", "tmpfs", "size=90%"},
		{[]string{"sysroot-var.mount"}, "/sysroot/etc", "/sysroot/var/etc", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/sysroot/home", "/sysroot/var/home", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/sysroot/mnt", "/sysroot/var/mnt", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/sysroot/opt", "/sysroot/var/opt", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/sysroot/root", "/sysroot/var/root", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/sysroot/usr/libexec", "/sysroot/var/usr/libexec", "none", "bind"},
		{[]string{"sysroot-var.mount"}, "/sysroot/usr/local", "/sysroot/var/usr/local", "none", "bind"},
	}

	if err := common.IsDir(generatorDir); err != nil {
		return err
	}
	generatorDir = strings.TrimRight(generatorDir, "/") + "/"

	if err := createPaths(generatorDir); err != nil {
		return err
	}

	if err := processSimpleK8sMount(generatorDir, mounts); err != nil {
		return err
	}

	if err := writeSystemdUnitMounts(generatorDir, mounts); err != nil {
		return err
	}

	if err := writeSystemdUnitServiceInit(generatorDir); err != nil {
		return err
	}

	return nil
}
