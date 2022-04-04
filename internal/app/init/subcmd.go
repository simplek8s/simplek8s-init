package init

import (
	"flag"
	"fmt"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
)

const (
	SUBCOMMAND_POPULATE  string = "populate"
	SUBCOMMAND_CONFIGURE        = "configure"
)

var SUBCOMMANDS = []string{
	SUBCOMMAND_POPULATE,
	SUBCOMMAND_CONFIGURE,
}

func IsSubcmd(args []string) bool {
	return common.IsStringInList(args[1], SUBCOMMANDS)
}

func RunSubCommands(args []string) error {
	if len(args) <= 1 {
		return fmt.Errorf("expected one of these subcommands: %v", SUBCOMMANDS)
	}

	switch args[1] {
	case SUBCOMMAND_POPULATE:
		cmdPopulate := flag.NewFlagSet("populate", flag.ExitOnError)
		cmdPopulateRoot := cmdPopulate.String("root", "", "sysroot directory")

		cmdPopulate.Parse(args[2:])
		if err := sysroot.Populate(*cmdPopulateRoot); err != nil {
			panic(err)
		}
	case SUBCOMMAND_CONFIGURE:
		cmdConfigure := flag.NewFlagSet("configure", flag.ExitOnError)
		cmdConfigureRoot := cmdConfigure.String("root", "", "sysroot directory")
		cmdConfigureLive := cmdConfigure.Bool("live", false, "preconfigure login")

		cmdConfigure.Parse(args[2:])
		if err := sysroot.Configure(*cmdConfigureRoot, *cmdConfigureLive); err != nil {
			panic(err)
		}
	}
	return nil
}
