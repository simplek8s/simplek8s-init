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

package mkpasswd

import (
	"errors"
	"simplek8s/pkg/log"
)

const CmdHelp = "Generate a hashed shadow password. Usage: mkpasswd [options]"

// CmdFn will:
//   - Generates a hashed shadow password based on user input or default settings.
//   - Exits cleanly.
func CmdFn() error {
	log.Trace("start")
	defer log.Trace("end")

	//TODO: Ask for user input and print hashed password.
	//TODO: Generate random password and print plained and hashed password.

	return errors.New("unimplemented")
}
