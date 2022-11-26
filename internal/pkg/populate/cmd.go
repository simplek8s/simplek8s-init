package populate

import (
	"flag"
	"fmt"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	"github.com/jlsalvador/simplek8s/internal/pkg/populate/initrd"
	"github.com/jlsalvador/simplek8s/internal/pkg/populate/sysroot"
	log "github.com/sirupsen/logrus"
)

const (
	STAGE_INITRD  = "initrd"
	STAGE_SYSROOT = "sysroot"
)

func Cmd(args []string) error {
	log.WithFields(log.Fields{
		"start": "Cmd",
		"args":  args,
	}).Debug()
	defer log.WithField("end", "Cmd").Debug()

	cmdPopulate := flag.NewFlagSet("populate", flag.ExitOnError)
	cmdPopulateStage := cmdPopulate.String("stage", "", "populate stage, could be initrd or sysroot")
	cmdPopulateOutput := cmdPopulate.String("output", "", "output directory")
	if err := cmdPopulate.Parse(args); err != nil {
		log.Error(err)
		return err
	}

	return CmdPopulate(*cmdPopulateStage, *cmdPopulateOutput)
}

func CmdPopulate(stage string, output string) error {
	log.WithFields(log.Fields{
		"start":  "CmdPopulate",
		"stage":  stage,
		"output": output,
	}).Debug()
	defer log.WithField("end", "CmdPopulate").Debug()

	// Validate args
	if !common.IsDir(output) {
		err := fmt.Errorf("%q is not a directory", output)
		log.Error(err)
		return err
	}

	switch stage {
	case STAGE_INITRD:
		if err := initrd.CmdPopulateInitrd(output); err != nil {
			log.Error(err)
			return err
		}
	case STAGE_SYSROOT:
		if err := sysroot.CmdPopulateSysroot(output); err != nil {
			log.Error(err)
			return err
		}
	default:
		err := fmt.Errorf("unknown stage: %v", stage)
		log.Error(err)
		return err
	}
	return nil
}
