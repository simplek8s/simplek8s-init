package init

import (
	"fmt"
	"path/filepath"
	"slices"

	systemdGenerator "github.com/jlsalvador/simplek8s/internal/pkg/systemdGenerator"
	"github.com/jlsalvador/simplek8s/pkg/common"
	log "github.com/sirupsen/logrus"
)

const (
	CMDALIAS_GENERATOR string = "simplek8s-generator"
)

func IsCmdAlias(args []string) bool {
	if len(args) < 1 {
		return false
	}

	fullPathFilename := args[0]
	got := filepath.Base(fullPathFilename)
	return slices.Contains([]string{
		CMDALIAS_GENERATOR,
	}, got)
}

func isCmdAliasSystemdGenerator(args []string) bool {
	// args[0] own cmd
	// args[1] normal dirpath
	// args[2] early dirpath
	// args[3] late dirpath
	if len(args) != 4 {
		return false
	}

	fullPathFilename := args[0]
	got := filepath.Base(fullPathFilename)
	want := CMDALIAS_GENERATOR

	return got == want && // Validate cmdname at args[0]
		common.IsDir(args[1]) && common.IsDir(args[2]) && common.IsDir(args[3]) // Validate directories
}

func cmdAliasSystemdGenerator(args []string) error {
	log.WithFields(log.Fields{
		"args": args,
	}).Debug("start")
	defer log.Debug("end")

	normalDir := args[0]
	earlyDir := args[1]
	lateDir := args[2]

	if err := systemdGenerator.CmdSystemdGenerator(normalDir, earlyDir, lateDir); err != nil {
		return err
	}
	return nil
}

func RunCmdAlias(args []string) error {
	log.WithFields(log.Fields{
		"args": args,
	}).Debug("start")
	defer log.Debug("end")

	switch {
	case isCmdAliasSystemdGenerator(args):
		return cmdAliasSystemdGenerator(args[1:])
	default:
		err := fmt.Errorf("unknown alias: %q", args)
		log.Error(err)
		return err
	}
}
