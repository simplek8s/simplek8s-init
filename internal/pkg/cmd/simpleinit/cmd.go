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

// Package simpleinit bootstrap from (initrd) root to the next (systemd) root.
package simpleinit

import (
	"fmt"
	"os"

	"github.com/simplek8s/simplek8s-init/pkg/log"
	"github.com/simplek8s/simplek8s-init/pkg/simplek8s"
)

const CmdHelp = "Bootstrap from (initrd) root to the next (systemd) root."

// CmdFn will:
//   - Verifies that the process is PID 1.
//   - Prepares the next root filesystem and switches to it.
//   - Exits cleanly.
func CmdFn() error {
	// Check if we are PID 1.
	pid := os.Getpid()
	if pid != 1 {
		return fmt.Errorf("not PID 1: %d", pid)
	}

	logToDevKmsg()

	log.Trace("start")
	defer log.Trace("end")

	where := "/sysroot"
	err := bootNextRoot(where)

	// We are PID1, so we can't crash when DEBUG is set.
	if err != nil && simplek8s.IsDebug() {
		return dropToShell(where, "Debug is true, dropping to a shell instead of crash.")
	}

	return err
}
