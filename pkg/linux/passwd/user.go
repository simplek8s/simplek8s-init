// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

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
	//  - `*`: The user can not login, but its account can run process.
	//  - `!`: The user is locked.
	//  - `x`: The user password is stored in /etc/shadow
	Password string
	Uid      int
	Gid      int
	Gecos    []string
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

	uid := fmt.Sprint(user.Uid)
	gid := fmt.Sprint(user.Gid)
	gecos := strings.Join(user.Gecos, ",")

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
			user.Uid = valueAsInt
		case 3:
			valueAsInt, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			user.Gid = valueAsInt
		case 4:
			user.Gecos = strings.Split(value, ",")
		case 5:
			user.Home = value
		case 6:
			user.Shell = value
		}
	}
	return nil
}
