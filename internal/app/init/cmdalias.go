package init

import (
	"path/filepath"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	"github.com/jlsalvador/simplek8s/internal/pkg/systemd"
	log "github.com/sirupsen/logrus"
)

const (
	CMDALIAS_GENERATOR string = "simplek8s-generator"
	CMDALIAS_UPDATECTL string = "simplek8s-updatectl"
)

var CMDALIAS = []string{
	CMDALIAS_GENERATOR,
	CMDALIAS_UPDATECTL,
}

func IsCmdAlias(args []string) bool {
	return common.IsStringInList(args[1], CMDALIAS)
}

func isCmdAliasSystemdGenerator(args []string) bool {
	// Validate cmdname at args[0]
	fullPathFilename := args[0]
	got := filepath.Base(fullPathFilename)
	want := CMDALIAS_GENERATOR
	if got != want {
		return false
	}

	// Validate args[1:]
	// [0] own cmd
	// [1] normal dir
	// [2] early dir
	// [3] late dir
	if len(args) != 4 {
		return false
	}

	//TODO: Validate directories

	return true
}

func cmdAliasSystemdGenerator(args []string) (normalDir string, earlyDir string, lateDir string) {
	log.WithFields(log.Fields{
		"args": args,
	}).Debug("isOperationSystemdGenerator")
	return args[1], args[2], args[3]
}

func RunCmdAlias(args []string) error {
	switch {
	case isCmdAliasSystemdGenerator(args):
		normalDir, earlyDir, lateDir := cmdAliasSystemdGenerator(args)
		if err := systemd.SystemdGenerator(normalDir, earlyDir, lateDir); err != nil {
			return err
		}
	}
	return nil
}
