package populate

import (
	"flag"
	"fmt"

	"github.com/jlsalvador/simplek8s/internal/pkg/populate/initrd"
	"github.com/jlsalvador/simplek8s/internal/pkg/populate/sysroot"
	"github.com/jlsalvador/simplek8s/pkg/common"
	log "github.com/sirupsen/logrus"
)

const (
	STAGE_INITRD  = "initrd"
	STAGE_SYSROOT = "sysroot"
)

func Cmd(args []string) error {
	log.WithFields(log.Fields{
		"args": args,
	}).Debug("start")
	defer log.Debug("end")

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
		"stage":  stage,
		"output": output,
	}).Debug("start")
	defer log.Debug("end")

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
