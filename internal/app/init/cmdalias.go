package init

import (
	"path/filepath"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	systemdGenerator "github.com/jlsalvador/simplek8s/internal/pkg/systemdGenerator"
	"github.com/jlsalvador/simplek8s/internal/pkg/update"
	log "github.com/sirupsen/logrus"
)

const (
	CMDALIAS_GENERATOR string = "simplek8s-generator"
	CMDALIAS_UPDATE    string = "simplek8s-update"
)

var CMDALIAS = []string{
	CMDALIAS_GENERATOR,
	CMDALIAS_UPDATE,
}

func IsCmdAlias(args []string) bool {
	if len(args) < 1 {
		return false
	}

	fullPathFilename := args[0]
	got := filepath.Base(fullPathFilename)
	return common.IsStringInList(got, CMDALIAS)
}

func isCmdAliasSystemdGenerator(args []string) bool {
	// Validate args[1:]
	// [0] own cmd
	// [1] normal dir
	// [2] early dir
	// [3] late dir
	if len(args) != 4 {
		return false
	}

	// Validate cmdname at args[0]
	fullPathFilename := args[0]
	got := filepath.Base(fullPathFilename)
	want := CMDALIAS_GENERATOR
	if got != want {
		return false
	}

	//TODO: Validate directories

	return true
}

func isCmdAliasUpdate(args []string) bool {
	// [0] own cmd
	if len(args) < 1 {
		return false
	}

	fullPathFilename := args[0]
	got := filepath.Base(fullPathFilename)
	want := CMDALIAS_UPDATE
	if got != want {
		return false
	}

	return true
}

func cmdAliasSystemdGenerator(args []string) error {
	log.WithFields(log.Fields{
		"args": args,
	}).Debug("cmdAliasSystemdGenerator")
	normalDir := args[1]
	earlyDir := args[2]
	lateDir := args[3]

	if err := systemdGenerator.CmdSystemdGenerator(normalDir, earlyDir, lateDir); err != nil {
		return err
	}
	return nil
}

func cmdAliasUpdate(args []string) error {
	return update.Cmd(args)
}

func RunCmdAlias(args []string) error {
	switch {
	case isCmdAliasSystemdGenerator(args):
		return cmdAliasSystemdGenerator(args)
	case isCmdAliasUpdate(args):
		return cmdAliasUpdate(args)
	}
	return nil
}
