package init

import (
	"errors"
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
	if len(args) < 2 {
		return false
	}
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

func isCmdAliasUpdateCtl(args []string) bool {
	fullPathFilename := args[0]
	got := filepath.Base(fullPathFilename)
	want := CMDALIAS_UPDATECTL
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

	if err := systemd.SystemdGenerator(normalDir, earlyDir, lateDir); err != nil {
		return err
	}
	return nil
}

func cmdAliasUpdateCtl(args []string) error {
	//TODO: unimplemented
	return errors.New("unimplemented")
}

func RunCmdAlias(args []string) error {
	switch {
	case isCmdAliasSystemdGenerator(args):
		return cmdAliasSystemdGenerator(args)
	case isCmdAliasUpdateCtl(args):
		return cmdAliasUpdateCtl(args)
	}
	return nil
}
