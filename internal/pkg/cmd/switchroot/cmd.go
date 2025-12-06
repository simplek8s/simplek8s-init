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

package switchroot

import (
	"fmt"
	"os"

	"github.com/simplek8s/simplek8s-init/internal/pkg/cmd/simpleinit"

	"github.com/simplek8s/simplek8s-init/pkg/log"
)

const CmdHelp = "Switch root filesystem to the next one."

// CmdFn will switch rootfs by /sysroot and execute init in it.
func CmdFn() error {
	log.Trace("start")
	defer log.Trace("end")

	// Check if we are PID 1.
	pid := os.Getpid()
	if pid != 1 {
		return fmt.Errorf("not PID 1: %d", pid)
	}

	where := "/sysroot"
	return simpleinit.SwitchRoot(where)
}
