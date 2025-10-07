package systemdGenerator

import (
	"fmt"

	sr "github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
	"github.com/jlsalvador/simplek8s/internal/pkg/systemdGenerator/initrd"
	"github.com/jlsalvador/simplek8s/internal/pkg/systemdGenerator/sysroot"
	"github.com/jlsalvador/simplek8s/pkg/common"
	"github.com/jlsalvador/simplek8s/pkg/linux"
	log "github.com/sirupsen/logrus"
)

// Will creates systemd units that will mount and populate paths
// https://www.freedesktop.org/software/systemd/man/systemd.generator.html#Description
func CmdSystemdGenerator(generatorDir string, earlyDir string, lateDir string) error {
	log.WithFields(log.Fields{
		"generatorDir": generatorDir,
		"earlyDir":     earlyDir,
		"lateDir":      lateDir,
	}).Debug("start")
	defer log.Debug("end")

	// Validate args
	if !common.IsDir(generatorDir) {
		return fmt.Errorf("%q is not a directory", generatorDir)
	}

	// We could be executed by initrd or by sysroot
	if linux.IsStageInitrd() {
		if err := initrd.CmdSystemdGeneratorInitrd(generatorDir); err != nil {
			log.Error(err)
			return err
		}
	} else {
		sr := sr.Sysroot{}

		if err := sysroot.CmdSystemdGeneratorSysroot(&sr, generatorDir); err != nil {
			log.Error(err)
			return err
		}

		// Commit sysroot
		if err := sr.Write("/"); err != nil {
			log.Error(err)
			return err
		}
	}

	return nil
}
