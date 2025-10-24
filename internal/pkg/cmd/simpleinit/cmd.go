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
	"simplek8s/pkg/log"
)

const CmdHelp = "Bootstrap from (initrd) root to the next (systemd) root."

// CmdFn will:
//   - Verifies that the process is PID 1.
//   - Prepares the next root filesystem and switches to it.
//   - Exits cleanly.
func CmdFn() error {
	log.Trace("start")
	defer log.Trace("end")

	// Check if we are PID 1.
	pid := os.Getpid()
	if pid != 1 {
		return fmt.Errorf("not PID 1: %d", pid)
	}

	where := "/sysroot"

	if err := createSysroot(where); err != nil {
		return fmt.Errorf("cannot populate next root %s: %w", where, err)
	}

	if err := switchRoot(where); err != nil {
		return fmt.Errorf("cannot chroot to %s: %w", where, err)
	}

	return nil
}
