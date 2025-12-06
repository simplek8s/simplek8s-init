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

// Package simplek8s provides common functions for the simplek8s distro.
package simplek8s

import (
	"strconv"
	"sync"

	"github.com/simplek8s/simplek8s-init/pkg/common"
	"github.com/simplek8s/simplek8s-init/pkg/linux/mount"
	"github.com/simplek8s/simplek8s-init/pkg/linux/procfs"
)

var (
	isDebug     bool
	isDebugOnce sync.Once
)

func IsDebug() bool {
	isDebugOnce.Do(func() {
		// Get debug from environment.
		v, _ := strconv.ParseBool(common.GetEnv("DEBUG", "false"))
		if v {
			isDebug = true
			return
		}

		// Get debug value from "/proc/cmdline".
		if !common.IsPathExists(procfs.CmdlineFilepath) {
			if err := mount.Mount(mount.Mountpoints.Proc); err != nil {
				isDebug = false
				return
			}
		}
		cmdline, err := common.ReadFileAsString(procfs.CmdlineFilepath)
		if err != nil {
			isDebug = false
			return
		}
		v, _ = procfs.GetCmdlineValue(string(cmdline), "debug", false)
		isDebug = v
	})
	return isDebug
}
