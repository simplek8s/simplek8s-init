package init

import (
	"fmt"

	init2 "github.com/jlsalvador/simplek8s/internal/pkg/init"
	"github.com/jlsalvador/simplek8s/internal/pkg/populate"
	"github.com/jlsalvador/simplek8s/pkg/common"
	log "github.com/sirupsen/logrus"
)

const (
	SUBCOMMAND_POPULATE string = "populate"
	SUBCOMMAND_INIT     string = "init"
)

var SUBCOMMANDS = []string{
	SUBCOMMAND_POPULATE,
	SUBCOMMAND_INIT,
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
	case SUBCOMMAND_INIT:
		if err := init2.Cmd(args[2:]); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}
