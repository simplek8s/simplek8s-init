// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

// Package passwd (un)marshal /etc/{group,shadow,passwd}
package passwd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type User struct {
	// This is the user's login name. It should not contain capital
	// letters.
	Name string
	// Could be ``, `*`, `!`, or `x`
	//  -  ``: The user can login without password.
	//  - `*`: The user cannot login, but its account can run process.
	//  - `!`: The user is locked.
	//  - `x`: The user password is stored in /etc/shadow
	Password string
	UID      int
	GID      int
	Gecos    string
	Home     string
	Shell    string
}

func NewUser(user User) User {
	if user.Home == "" {
		// A better approach could be `/var/empty`, but because legacy reasons,
		// a system user uses `/` as the default home.
		user.Home = "/"
	}
	// Not login by default
	if user.Shell == "" {
		user.Shell = "/usr/sbin/nologin"
	}
	return user
}

func (user User) Marshal() (string, error) {
	name := user.Name
	if len(user.Name) == 0 {
		return "", errors.New("name is required")
	}

	password := user.Password
	if len(user.Password) == 0 {
		password = "x"
	}

	uid := fmt.Sprint(user.UID)
	gid := fmt.Sprint(user.GID)
	gecos := user.Gecos

	home := user.Home
	if len(user.Home) == 0 {
		home = "/"
	}

	shell := user.Shell
	if len(user.Shell) == 0 {
		shell = "/usr/bin/nologin"
	}

	return strings.Join([]string{
		name, password, uid, gid, gecos, home, shell,
	}, ":"), nil
}

func UnmarshalUser(entry string, user *User) error {
	for index, value := range strings.Split(entry, ":") {
		switch index {
		case 0:
			user.Name = value
		case 1:
			user.Password = value
		case 2:
			valueAsInt, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			user.UID = valueAsInt
		case 3:
			valueAsInt, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			user.GID = valueAsInt
		case 4:
			user.Gecos = value
		case 5:
			user.Home = value
		case 6:
			user.Shell = value
		}
	}
	return nil
}
