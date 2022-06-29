package main

import (
	"fmt"
	"os"
	"strconv"

	cmd "github.com/jlsalvador/simplek8s/internal/app/init"
	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	log "github.com/sirupsen/logrus"
)

var Version string

func showHelp() {
	fmt.Printf("SimpleK8s %s\n", Version)
}

func main() {
	if debug, _ := strconv.ParseBool(common.GetEnv("DEBUG", "false")); debug || common.IsCmdlineDebug() {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	}
	defer log.Debug("done")

	if cmd.IsCmdAlias(os.Args) {
		if err := cmd.RunCmdAlias(os.Args); err != nil {
			panic(err)
		}
	} else if cmd.IsSubcmd(os.Args) {
		if err := cmd.RunSubCommands(os.Args); err != nil {
			panic(err)
		}
	} else {
		showHelp()
		os.Exit(1)
	}
}
