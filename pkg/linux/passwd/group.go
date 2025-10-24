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

package passwd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Group struct {
	Name     string
	Password string // Could be ``, `*`, `!`, `x`, or hash.
	GID      int
	UserList []string
}

func NewGroup(group Group) Group {
	if group.UserList == nil {
		group.UserList = []string{}
	}
	return group
}

func (group Group) Marshal() (string, error) {
	name := group.Name
	if len(group.Name) == 0 {
		return "", errors.New("name is required")
	}

	password := group.Password
	if len(group.Password) == 0 {
		password = "x"
	}

	gid := fmt.Sprint(group.GID)
	userList := strings.Join(group.UserList, ",")

	return strings.Join([]string{
		name, password, gid, userList,
	}, ":"), nil
}

func UnmarshalGroup(entry string, group *Group) error {
	for index, value := range strings.Split(entry, ":") {
		switch index {
		case 0:
			group.Name = value
		case 1:
			group.Password = value
		case 2:
			valueAsInt, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			group.GID = valueAsInt
		case 3:
			group.UserList = strings.Split(value, ",")
		}
	}
	return nil
}
