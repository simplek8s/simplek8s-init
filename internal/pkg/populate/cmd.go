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
	STAGE_INITRD  string = "initrd"
	STAGE_SYSROOT        = "sysroot"
)

func Cmd(args []string) error {
	log.Debug("start")
	defer log.Debug("end")

	cmdPopulate := flag.NewFlagSet("populate", flag.ExitOnError)
	cmdPopulateStage := cmdPopulate.String("stage", "", "populate stage, could be initrd or sysroot")
	cmdPopulateOutput := cmdPopulate.String("output", "", "output directory")
	cmdPopulate.Parse(args)

	return CmdPopulate(*cmdPopulateStage, *cmdPopulateOutput)
}

func CmdPopulate(stage string, output string) error {
	log.WithFields(log.Fields{
		"start":  "CmdPopulate",
		"output": output,
	}).Debug()
	defer log.WithField("end", "CmdPopulate").Debug()

	// Validate args
	if !common.IsDir(output) {
		return fmt.Errorf("%q is not a directory", output)
	}

	switch stage {
	case STAGE_INITRD:
		if err := initrd.CmdPopulateInitrd(output); err != nil {
			return err
		}
	case STAGE_SYSROOT:
		if err := sysroot.CmdPopulateSysroot(output); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown stage: %v", stage)
	}
	return nil
}
