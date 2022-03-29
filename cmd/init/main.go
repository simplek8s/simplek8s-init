package main

import (
	"fmt"
	"os"
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

const (
	OPERATION_NONE int = iota
	OPERATION_SYSTEMD
	OPERATION_POPULATE
	OPERATION_CONFIGURATOR
)

func getOperation(args []string) int {
	log.WithFields(log.Fields{
		"args": args,
	}).Debug()

	operation := OPERATION_NONE
	switch len(args) {
	case 4:
		operation = OPERATION_SYSTEMD
	case 3:
		switch args[1] {
		case "populate":
			operation = OPERATION_POPULATE
		case "configurator":
			operation = OPERATION_CONFIGURATOR
		}
	}
	return operation
}

func printUsage() {
	fmt.Println(`SimpleK8s Init
This program do not support to be executed by the user.`)
}

func main() {
	if debug, _ := strconv.ParseBool(getEnv("DEBUG", "true")); debug {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	}

	switch getOperation(os.Args) {
	case OPERATION_SYSTEMD:
		if err := systemd.SystemdGenerator(os.Args[1]); err != nil {
			panic(err)
		}
	case OPERATION_POPULATE:
		if err := sysroot.Populate(os.Args[2]); err != nil {
			panic(err)
		}
	case OPERATION_CONFIGURATOR:
		if err := sysroot.Configure(os.Args[2]); err != nil {
			panic(err)
		}
	default:
		printUsage()
	}

	log.Debug("done")
}
