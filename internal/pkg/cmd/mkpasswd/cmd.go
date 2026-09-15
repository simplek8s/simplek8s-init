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
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/simplek8s/simplek8s-init/pkg/log"
	"github.com/simplek8s/simplek8s-init/pkg/simplek8s/generateshadow"

	"golang.org/x/term"
)

const CmdHelp = "Generate a hashed shadow password. Usage: mkpasswd [options]"

var CmdOptions = []string{
	"-r, --random\tGenerate a random password",
}

// CmdFn will:
//   - Generates a hashed shadow password based on user input or default settings.
//   - Exits cleanly.
func CmdFn() error {
	log.Trace("start")
	defer log.Trace("end")

	isTerm := term.IsTerminal(int(os.Stdin.Fd()))

	// Read input from stdin if available.
	var stdin []byte
	if !isTerm {
		stdin, _ = io.ReadAll(os.Stdin)
	}

	// Check for the random flag.
	isRandom := false
	if slices.Index(os.Args, "-r") >= 0 {
		isRandom = true
	} else if slices.Index(os.Args, "--random") >= 0 {
		isRandom = true
	}

	var pwd string
	if isRandom {
		pwd, err := generateshadow.GeneratePwd()
		if err != nil {
			return err
		}

		// Print plain generated password.
		fmt.Printf("Plain: %s\n", pwd)
	} else if len(stdin) > 0 {
		pwd = strings.TrimSpace(string(stdin))
	} else if isTerm {
		// Ask for user input.
		var err error
		var pwd2 string

		pwd, err = PromptSecret("Password: ")
		if err != nil {
			return err
		}

		pwd2, err = PromptSecret("Repeat Password: ")
		if err != nil {
			return err
		}

		if pwd != pwd2 {
			pwd = ""
			pwd2 = ""
			return errors.New("passwords do not match")
		}
		pwd2 = ""
	} else {
		return errors.New("cannot ask for a password because term is not available")
	}

	// Hash pwd.
	hashed, err := generateshadow.GenerateShadowPassword(pwd)
	// Best-effort: remove secrets from memory.
	clear(stdin)
	pwd = ""
	if err != nil {
		return err
	}

	// Print hashed password.
	fmt.Printf("Hashed: %s\n", hashed)

	return nil
}
