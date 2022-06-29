package init

import (
	"flag"
	"fmt"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	"github.com/jlsalvador/simplek8s/internal/pkg/populate"
)

const (
	SUBCOMMAND_POPULATE string = "populate"
)

var SUBCOMMANDS = []string{
	SUBCOMMAND_POPULATE,
}

func IsSubcmd(args []string) bool {
	if len(args) < 2 {
		return false
	}
	return common.IsStringInList(args[1], SUBCOMMANDS)
}

func RunSubCommands(args []string) error {
	if len(args) <= 1 {
		return fmt.Errorf("expected one of these subcommands: %v", SUBCOMMANDS)
	}

	switch args[1] {
	case SUBCOMMAND_POPULATE:
		cmdPopulate := flag.NewFlagSet(SUBCOMMAND_POPULATE, flag.ExitOnError)
		cmdPopulateStage := cmdPopulate.String("stage", "", "populate stage, could be initrd or sysroot")
		cmdPopulateOutput := cmdPopulate.String("output", "", "output directory")
		cmdPopulate.Parse(args[2:])

		if err := populate.CmdPopulate(*cmdPopulateStage, *cmdPopulateOutput); err != nil {
			panic(err)
		}
	}
	return nil
}
