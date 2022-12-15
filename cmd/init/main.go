package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	cmd "github.com/jlsalvador/simplek8s/internal/app/init"
	"github.com/jlsalvador/simplek8s/pkg/common"
	log "github.com/sirupsen/logrus"
)

var Version string

func showHelp() {
	fmt.Println(Version)
}

func main() {
	if debug, _ := strconv.ParseBool(common.GetEnv("DEBUG", "false")); debug || common.IsCmdlineDebug() {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)

		logFileName := fmt.Sprintf("simplek8s-init.%d.log", time.Now().Unix())
		logFileName = filepath.Join(os.TempDir(), logFileName)
		log.Infof("logfile: %s", logFileName)
		if file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err != nil {
			log.Error(err)
			panic(err)
		} else {
			defer file.Close()
			log.SetOutput(io.MultiWriter(file, os.Stdout)) // Write into file and stdout
		}
	}
	defer log.Debug("done")

	if cmd.IsCmdAlias(os.Args) {
		if err := cmd.RunCmdAlias(os.Args); err != nil {
			log.Error(err)
			panic(err)
		}
	} else if cmd.IsSubcmd(os.Args) {
		if err := cmd.RunSubCommands(os.Args); err != nil {
			log.Error(err)
			panic(err)
		}
	} else {
		showHelp()
		os.Exit(1)
	}
}
