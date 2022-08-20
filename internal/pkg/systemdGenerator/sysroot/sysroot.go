package sysroot

import (
	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
	log "github.com/sirupsen/logrus"
)

func CmdSystemdGeneratorSysroot(sr *sysroot.Sysroot, generatorDir string) error {
	log.WithFields(log.Fields{
		"start":        "CmdSystemdGeneratorSysroot",
		"generatorDir": generatorDir,
	}).Debug()
	defer log.WithField("end", "CmdSystemdGeneratorSysroot").Debug()

	// Nothing here, maybe in the future.

	return nil
}
