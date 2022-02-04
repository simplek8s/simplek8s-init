package main

import (
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

func main() {
	if debug, _ := strconv.ParseBool(getEnv("DEBUG", "true")); debug {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	}

	log.WithFields(log.Fields{
		"args": os.Args,
	}).Debug()

	if len(os.Args) == 4 {
		// https://www.freedesktop.org/software/systemd/man/systemd.generator.html#Description
		if stat, err := os.Stat(os.Args[1]); err != nil {
			log.WithField("path", os.Args[1]).Panic(err)
		} else if isDir := stat.IsDir(); !isDir {
			log.WithFields(log.Fields{
				"path":  os.Args[1],
				"isDir": isDir,
			}).Panic("must be called from systemd")
		}
		if err := systemd.SystemdGenerator(os.Args[1]); err != nil {
			panic(err)
		}
	} else if len(os.Args) == 3 && (os.Args[1] == "populate" || os.Args[1] == "configurator") {
		var sr *sysroot.Sysroot
		var err error
		if stat, err := os.Stat(os.Args[2]); err != nil || !stat.IsDir() {
			log.WithFields(log.Fields{
				"path":  os.Args[2],
				"isDir": stat.IsDir(),
			}).Panic(err)
		}
		if sr, err = sysroot.New(os.Args[2]); err != nil {
			panic(err)
		}
		switch os.Args[1] {
		case "populate":
			if err := sr.Populate(); err != nil {
				panic(err)
			}
		case "configurator":
			if err := sr.YamlParser(); err != nil {
				panic(err)
			}
		}
	}

	log.Debug("done")
}
