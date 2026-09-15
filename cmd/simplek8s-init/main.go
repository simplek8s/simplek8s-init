// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/simplek8s/simplek8s-init/internal/pkg/cmd/mkpasswd"
	"github.com/simplek8s/simplek8s-init/internal/pkg/cmd/simpleinit"
	"github.com/simplek8s/simplek8s-init/internal/pkg/cmd/switchroot"
	"github.com/simplek8s/simplek8s-init/internal/pkg/cmd/wizard"

	"github.com/simplek8s/simplek8s-init/pkg/log"
	"github.com/simplek8s/simplek8s-init/pkg/simplek8s"

	l "github.com/sirupsen/logrus"
)

var Version = "0.0.1610038522"

var cmds = []struct {
	name    string
	fn      func() error
	help    string
	options []string
}{
	{"init", simpleinit.CmdFn, simpleinit.CmdHelp, nil},
	{"wizard", wizard.CmdFn, wizard.CmdHelp, nil},
	{"mkpasswd", mkpasswd.CmdFn, mkpasswd.CmdHelp, mkpasswd.CmdOptions},
	{"switchroot", switchroot.CmdFn, switchroot.CmdHelp, nil},
}

func help() error {
	log.Trace("start")
	defer log.Trace("end")

	_, err := fmt.Printf(`SimpleK8s v%s multi-call binary.

Usage: %s [function [arguments]...]
   or: function [arguments]...

Currently defined functions:
`, Version, filepath.Base(os.Args[0]))

	for _, cmd := range cmds {
		fmt.Printf("  %s\n", cmd.name)
		if len(cmd.help) > 0 {
			fmt.Printf("    %s\n", cmd.help)
		}
		if len(cmd.options) > 0 {
			fmt.Println("\n    Options:")
			for _, o := range cmd.options {
				fmt.Printf("      %s\n", o)
			}
		}
		fmt.Println()
	}

	return err
}

func fetchCmd() string {
	log.Trace("start")
	defer log.Trace("end")

	if len(os.Args) == 1 {
		return filepath.Base(os.Args[0])
	} else if len(os.Args) > 1 {
		// Linux kernel sends to us all kernel arguments.
		if filepath.Base(os.Args[0]) == "init" {
			return "init"
		}
		// From a shell, just the first argument is the command name.
		return os.Args[1]
	}
	return ""
}

func initLog() {
	if simplek8s.IsDebug() {
		log.Level = log.LevelDebug

		l.SetLevel(l.DebugLevel)
		l.SetReportCaller(true)
		l.SetFormatter(&l.TextFormatter{
			DisableTimestamp: true,
		})
	}
}

func main() {
	initLog()

	log.Trace("start")
	defer log.Trace("end")

	found := false
	var err error
	basename := fetchCmd()
	for _, cmd := range cmds {
		if basename == cmd.name {
			found = true
			err = cmd.fn()
			break
		}
	}

	if !found {
		err = help()
	}

	if err != nil {
		// Print callback.
		log.DebugFn(func() string {
			return "Callback:\n" + string(debug.Stack())
		})
		log.Error(err.Error())

		os.Exit(1)
	}

	os.Exit(0)
}
