package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jlsalvador/simplek8s/sysroot"
	"github.com/jlsalvador/simplek8s/systemd"
	log "github.com/sirupsen/logrus"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func isSubcommandSystemdGenerator(args []string) (isGenerator bool, normalDir string, earlyDir string, lateDir string) {
	// [0] own cmd
	// [1] normal dir
	// [2] early dir
	// [3] late dir
	if len(args) != 4 {
		return false, "", "", ""
	}

	fullPathFilename := args[0]
	got := filepath.Base(fullPathFilename)
	want := "simplek8s-generator"
	log.WithFields(log.Fields{
		"args": args,
		"got":  got,
		"want": want,
	}).Debug("isOperationSystemdGenerator")
	return got == want, args[1], args[2], args[3]
}

func main() {
	if debug, _ := strconv.ParseBool(getEnv("DEBUG", "true")); debug {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	}
	defer log.Debug("done")

	if isGenerator, normalDir, earlyDir, lateDir := isSubcommandSystemdGenerator(os.Args); isGenerator {
		if err := systemd.SystemdGenerator(normalDir, earlyDir, lateDir); err != nil {
			panic(err)
		}
		return
	}

	if len(os.Args) <= 1 {
		fmt.Println("expected 'populate' or 'configure' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "populate":
		cmdPopulate := flag.NewFlagSet("populate", flag.ExitOnError)
		cmdPopulateRoot := cmdPopulate.String("root", "", "sysroot directory")

		cmdPopulate.Parse(os.Args[2:])
		if err := sysroot.Populate(*cmdPopulateRoot); err != nil {
			panic(err)
		}
	case "configure":
		cmdConfigure := flag.NewFlagSet("configure", flag.ExitOnError)
		cmdConfigureRoot := cmdConfigure.String("root", "", "sysroot directory")
		cmdConfigureLive := cmdConfigure.Bool("live", false, "preconfigure login")

		cmdConfigure.Parse(os.Args[2:])
		if err := sysroot.Configure(*cmdConfigureRoot, *cmdConfigureLive); err != nil {
			panic(err)
		}
	default:
		fmt.Printf("unknown subcommand %q, expected 'populate' or 'configure'\n", os.Args[1])
		os.Exit(1)
	}
}
