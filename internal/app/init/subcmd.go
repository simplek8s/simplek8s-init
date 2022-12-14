package init

import (
	"fmt"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	"github.com/jlsalvador/simplek8s/internal/pkg/populate"
	log "github.com/sirupsen/logrus"
)

const (
	SUBCOMMAND_POPULATE string = "populate"
)

var SUBCOMMANDS = []string{
	SUBCOMMAND_POPULATE,
}

func IsSubcmd(args []string) bool {
	log.WithFields(log.Fields{
		"args": args,
	}).Debug("start")
	defer log.Debug("end")

	if len(args) < 2 {
		return false
	}
	return common.IsStringInList(args[1], SUBCOMMANDS)
}

func RunSubCommands(args []string) error {
	log.WithFields(log.Fields{
		"args": args,
	}).Debug("start")
	defer log.Debug("end")

	if len(args) <= 1 {
		err := fmt.Errorf("expected one of these subcommands: %v", SUBCOMMANDS)
		log.Error(err)
		return err
	}

	switch args[1] {
	case SUBCOMMAND_POPULATE:
		if err := populate.Cmd(args[2:]); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}
