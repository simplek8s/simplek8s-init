package initrd

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jlsalvador/simplek8s/internal/pkg/systemdGenerator/templates"
	"github.com/jlsalvador/simplek8s/pkg/common"
	log "github.com/sirupsen/logrus"
)

// Create and enable "init" systemd unit
func initrdSystemdUnits(output string) error {
	log.WithFields(log.Fields{
		"output": output,
	}).Debug("start")
	defer log.Debug("end")

	unitName := "sysroot-init.service"
	filePath := filepath.Join(output, unitName)
	perm := fs.FileMode(0644)
	uid := 0
	gid := 0

	// Generate "init" systemd unit content
	tmplName := "assets/run/systemd/system/systemd.service.go.tmpl"
	tmplData := templates.TmplDataSystemdUnitService{
		Description:     "Init /sysroot",
		Conflicts:       []string{"shutdown.target"},
		Requires:        []string{"systemd-udev-settle.service"},
		Before:          []string{"initrd-root-fs.target", "shutdown.target"},
		After:           []string{"systemd-udev-settle.service"},
		Type:            "oneshot",
		Restart:         "no",
		RemainAfterExit: true,
		ExecStart:       []string{"/usr/lib/simplek8s/init init"},
	}
	content, err := common.RenderTemplate(templates.Templates, tmplName, tmplData)
	if err != nil {
		log.WithFields(log.Fields{
			"output":   output,
			"tmplName": tmplName,
			"tmplData": tmplData,
		}).Error(err)
		return err
	}

	// Write "init" systemd unit
	if err := os.WriteFile(filePath, content, perm); err != nil {
		log.WithFields(log.Fields{
			"filePath": filePath,
			"content":  content,
			"perm":     0644,
		}).Error(err)
		return err
	}
	if err := os.Chown(filePath, uid, gid); err != nil {
		log.WithFields(log.Fields{
			"filePath": filePath,
			"uid":      uid,
			"gid":      gid,
		}).Error(err)
		return err
	}

	// Enable "init" systemd unit
	filePath = filepath.Join(output, "initrd-root-fs.target.requires", unitName)
	target := filepath.Join(output, unitName)
	overwrite := true
	hard := false
	if err := common.CreateSymlink(filePath, target, overwrite, uid, gid, hard); err != nil {
		log.WithFields(log.Fields{
			"filePath":  filePath,
			"target":    target,
			"overwrite": overwrite,
			"uid":       uid,
			"gid":       gid,
			"hard":      hard,
		}).Error(err)
		return err
	}
	return nil
}

func CmdSystemdGeneratorInitrd(generatorDir string) error {
	log.WithFields(log.Fields{
		"generatorDir": generatorDir,
	}).Debug("start")
	defer log.Debug("end")

	if err := initrdSystemdUnits(generatorDir); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
