package main

import (
	"io/fs"
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
		pathnameNormal := os.Args[1]
		pathnameEarly := os.Args[2]
		pathnameLate := os.Args[3]
		var statNormal fs.FileInfo
		var statEarly fs.FileInfo
		var statLate fs.FileInfo
		var err error
		statNormal, err = os.Stat(pathnameNormal)
		if err != nil {
			log.WithFields(log.Fields{
				"pathnameNormal": pathnameNormal,
				"pathnameEarly":  pathnameEarly,
				"pathnameLate":   pathnameLate,
			}).Panic(err)
		}
		statEarly, err = os.Stat(pathnameEarly)
		if err != nil {
			log.WithFields(log.Fields{
				"pathnameNormal": pathnameNormal,
				"pathnameEarly":  pathnameEarly,
				"pathnameLate":   pathnameLate,
			}).Panic(err)
		}
		statLate, err = os.Stat(pathnameLate)
		if err != nil {
			log.WithFields(log.Fields{
				"pathnameNormal": pathnameNormal,
				"pathnameEarly":  pathnameEarly,
				"pathnameLate":   pathnameLate,
			}).Panic(err)
		}
		isDirNormal := statNormal.IsDir()
		isDirEarly := statEarly.IsDir()
		isDirLate := statLate.IsDir()
		if !isDirNormal || !isDirEarly || !isDirLate {
			log.WithFields(log.Fields{
				"pathnameNormal": pathnameNormal,
				"pathnameEarly":  pathnameEarly,
				"pathnameLate":   pathnameLate,
				"isDirNormal":    isDirNormal,
				"isDirEarly":     isDirEarly,
				"isDirLate":      isDirLate,
			}).Panic("must be called as systemd generator")
		}
		systemd.SystemdGenerator(pathnameNormal)
	} else {
		sysroot.YamlParser()
	}

	log.Debug("done")
}
