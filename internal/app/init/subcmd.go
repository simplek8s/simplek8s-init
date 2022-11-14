package init

import (
	"fmt"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	"github.com/jlsalvador/simplek8s/internal/pkg/populate"
	"github.com/jlsalvador/simplek8s/internal/pkg/update"
)

const (
	SUBCOMMAND_POPULATE string = "populate"
	SUBCOMMAND_UPDATE   string = "update"
)

var SUBCOMMANDS = []string{
	SUBCOMMAND_POPULATE,
	SUBCOMMAND_UPDATE,
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
		if err := populate.Cmd(args[1:]); err != nil {
			panic(err)
		}
	case SUBCOMMAND_UPDATE:
		if err := update.Cmd(args[1:]); err != nil {
			panic(err)
		}
	}
	return nil
}
