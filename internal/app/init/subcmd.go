package init

import (
	"flag"
	"fmt"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	"github.com/jlsalvador/simplek8s/internal/pkg/k8s"
	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
)

const (
	SUBCOMMAND_POPULATE  string = "populate"
	SUBCOMMAND_CONFIGURE        = "configure"
	SUBCOMMAND_DEBUG            = "debug"
)

var SUBCOMMANDS = []string{
	SUBCOMMAND_POPULATE,
	SUBCOMMAND_CONFIGURE,
	SUBCOMMAND_DEBUG,
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
		cmdPopulateRoot := cmdPopulate.String("root", "", "sysroot directory")
		cmdPopulate.Parse(args[2:])

		if err := sysroot.Populate(*cmdPopulateRoot); err != nil {
			panic(err)
		}
	case SUBCOMMAND_CONFIGURE:
		cmdConfigure := flag.NewFlagSet(SUBCOMMAND_CONFIGURE, flag.ExitOnError)
		cmdConfigureRoot := cmdConfigure.String("root", "", "sysroot directory")
		cmdConfigureLive := cmdConfigure.Bool("live", false, "preconfigure login")
		cmdConfigure.Parse(args[2:])

		if err := sysroot.Configure(*cmdConfigureRoot, *cmdConfigureLive); err != nil {
			panic(err)
		}
	case SUBCOMMAND_DEBUG:
		cmdDebug := flag.NewFlagSet(SUBCOMMAND_DEBUG, flag.ExitOnError)
		cmdDebugRenderKubeadmConf := cmdDebug.Bool("renderKubeadmConf", false, "render kubeadm.conf")
		cmdDebug.Parse(args[2:])

		switch {
		case *cmdDebugRenderKubeadmConf:
			if err := k8s.RenderKubeadmConf(); err != nil {
				panic(err)
			}
		}
	}
	return nil
}
