package systemd

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"text/template"

	unit "github.com/coreos/go-systemd/v22/unit"
	log "github.com/sirupsen/logrus"
)

//go:embed templates/*
var templates embed.FS

var initrdRootFsTargetRequiresPath string = "/initrd-root-fs.target.requires"

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

func writeSysrootTemplate(generatorDir string, templateFilename string, outputFilename string, data interface{}) error {
	var content []byte
	if data != nil {
		tmpl, err := template.ParseFS(templates, templateFilename)
		if err != nil {
			log.WithFields(log.Fields{
				"templates": templates,
				"tmpl":      tmpl,
			}).Error(err)
			return err
		}

		content := new(bytes.Buffer)
		if err := tmpl.Execute(content, data); err != nil {
			log.WithFields(log.Fields{
				"tmpl": tmpl,
				"data": data,
			}).Error(err)
			return err
		}
	} else {
		var err error
		content, err = templates.ReadFile(templateFilename)
		if err != nil {
			log.WithFields(log.Fields{
				"templates": templates,
			}).Error(err)
			return err
		}
	}

	name := generatorDir + outputFilename
	mode := fs.FileMode(0644)
	if err := os.WriteFile(name, content, mode); err != nil {
		log.WithFields(log.Fields{
			"name":    name,
			"content": string(content),
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

func SystemdGenerator(generatorDir string) error {
	log.Debug("init systemd generator")

	if err := createPaths(generatorDir); err != nil {
		log.WithField("generatorDir", generatorDir).Error(err)
		return err
	}

	template := "templates/sysroot.mount"
	output := "/sysroot.mount"
	if err := writeSysrootTemplate(generatorDir, template, output, nil); err != nil {
		log.WithFields(log.Fields{
			"generatorDir": generatorDir,
			"template":     template,
			"output":       output,
		}).Error(err)
		return err
	}

	template = "templates/sysroot-var.mount.go.tmpl"
	output = "/sysroot-var.mount"
	data := struct {
		What    string
		Type    string
		Options string
	}{
		What:    "tmpfs",
		Type:    "tmpfs",
		Options: "size=90%",
	}
	//TODO: Find `/var` values from somewhere (CMDLINE || YAML)
	// data.What = "/dev/disk/by-partlabel/var"
	// data.Type = "auto"
	// data.Options = "defaults"
	if err := writeSysrootTemplate(generatorDir, template, output, data); err != nil {
		log.WithFields(log.Fields{
			"generatorDir": generatorDir,
			"template":     template,
			"output":       output,
			"data":         data,
		}).Error(err)
		return err
	}

	for _, folder := range []string{
		"etc",
		"home",
		"mnt",
		"opt",
		"root",
		"usr/libexec",
		"usr/local",
	} {
		escapedFolder := unit.UnitNameEscape(folder)
		template := "templates/sysroot-PATH.mount.go.tmpl"
		output := "/sysroot-" + escapedFolder + ".mount"
		data := struct {
			Folder string
		}{
			Folder: folder,
		}
		if err := writeSysrootTemplate(generatorDir, template, output, data); err != nil {
			log.WithFields(log.Fields{
				"generatorDir": generatorDir,
				"template":     template,
				"output":       output,
				"data":         data,
			}).Error(err)
			return err
		}
	}

	template = "templates/sysroot-populate.service"
	output = "/sysroot-populate.service"
	if err := writeSysrootTemplate(generatorDir, template, output, nil); err != nil {
		log.WithFields(log.Fields{
			"generatorDir": generatorDir,
			"template":     template,
			"output":       output,
		}).Error(err)
		return err
	}

	template = "templates/sysroot-configurator.service"
	output = "/sysroot-configurator.service"
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
