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

func writeSysrootMount(generatorDir string) error {
	filename := "/sysroot.mount"

	content, err := templates.ReadFile("templates" + filename)
	if err != nil {
		log.WithFields(log.Fields{
			"templates": templates,
		}).Error(err)
		return err
	}

	name := generatorDir + filename
	mode := fs.FileMode(0644)
	if err := os.WriteFile(name, content, mode); err != nil {
		log.WithFields(log.Fields{
			"name":    name,
			"content": content,
			"mode":    mode,
		}).Error(err)
		return err
	}

	oldname := ".." + filename
	newname := generatorDir + initrdRootFsTargetRequiresPath + filename
	if err := os.Symlink(oldname, newname); err != nil {
		log.WithFields(log.Fields{
			"oldname": oldname,
			"newname": newname,
		}).Error(err)
		return err
	}

	return nil
}

func writeSysrootVarMount(generatorDir string) error {
	filename := "/sysroot-var.mount"

	tmpl, err := template.ParseFS(templates, "templates"+filename+".go.tmpl")
	if err != nil {
		log.WithFields(log.Fields{
			"templates": templates,
			"tmpl":      tmpl,
		}).Error(err)
		return err
	}

	payload := struct {
		What    string
		Type    string
		Options string
	}{
		What:    "tmpfs",
		Type:    "tmpfs",
		Options: "size=90%",
	}

	//TODO: Find `/var` values from somewhere (CMDLINE || YAML)
	// payload.What = "/dev/disk/by-partlabel/var"
	// payload.Type = "auto"
	// payload.Options = "defaults"

	content := new(bytes.Buffer)
	if err := tmpl.Execute(content, payload); err != nil {
		log.WithFields(log.Fields{
			"tmpl":    tmpl,
			"payload": payload,
		}).Error(err)
		return err
	}

	name := generatorDir + filename
	mode := fs.FileMode(0644)
	if err := os.WriteFile(name, content.Bytes(), mode); err != nil {
		log.WithFields(log.Fields{
			"name":    name,
			"content": content,
			"mode":    mode,
		}).Error(err)
		return err
	}

	oldname := ".." + filename
	newname := generatorDir + initrdRootFsTargetRequiresPath + filename
	if err := os.Symlink(oldname, newname); err != nil {
		log.WithFields(log.Fields{
			"oldname": oldname,
			"newname": newname,
		}).Error(err)
		return err
	}

	return nil
}

func writeSysrootPATHMount(generatorDir string, folder string) error {
	tmpl, err := template.ParseFS(templates, "templates/sysroot-PATH.mount.go.tmpl")
	if err != nil {
		log.WithFields(log.Fields{
			"templates": templates,
			"tmpl":      tmpl,
		}).Error(err)
		return err
	}

	payload := struct {
		Folder string
	}{
		Folder: folder,
	}

	content := new(bytes.Buffer)
	if err := tmpl.Execute(content, payload); err != nil {
		log.WithFields(log.Fields{
			"tmpl":    tmpl,
			"payload": payload,
		}).Error(err)
		return err
	}

	escapedFolder := unit.UnitNameEscape(folder)
	name := generatorDir + "/sysroot-" + escapedFolder + ".mount"
	mode := fs.FileMode(0644)
	if err := os.WriteFile(name, content.Bytes(), mode); err != nil {
		log.WithFields(log.Fields{
			"name":    name,
			"content": content,
			"mode":    mode,
		}).Error(err)
		return err
	}

	oldname := ".." + "/sysroot-" + escapedFolder + ".mount"
	newname := generatorDir + initrdRootFsTargetRequiresPath + "/sysroot-" + escapedFolder + ".mount"
	if err := os.Symlink(oldname, newname); err != nil {
		log.WithFields(log.Fields{
			"oldname": oldname,
			"newname": newname,
		}).Error(err)
		return err
	}

	return nil
}

func writeSysrootPopulate(generatorDir string) error {
	filename := "/sysroot-populate.service"

	content, err := templates.ReadFile("templates" + filename)
	if err != nil {
		log.WithFields(log.Fields{
			"templates": templates,
		}).Error(err)
		return err
	}
	name := generatorDir + filename
	mode := fs.FileMode(0644)
	if err := os.WriteFile(name, content, mode); err != nil {
		log.WithFields(log.Fields{
			"name":    name,
			"content": content,
			"mode":    mode,
		}).Error(err)
		return err
	}

	oldname := ".." + filename
	newname := generatorDir + initrdRootFsTargetRequiresPath + filename
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
		return err
	}
	if err := writeSysrootMount(generatorDir); err != nil {
		return err
	}
	if err := writeSysrootVarMount(generatorDir); err != nil {
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
		if err := writeSysrootPATHMount(generatorDir, folder); err != nil {
			return err
		}
	}
	if err := writeSysrootPopulate(generatorDir); err != nil {
		return err
	}

	return nil
}

//TODO: Refactor this whole file
