package passwd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Group struct {
	Name string
	// Could be ``, `*`, `!`, or `x`
	Password string
	Gid      int
	UserList []string
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

	gid := fmt.Sprint(group.Gid)
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
			group.Gid = valueAsInt
		case 3:
			group.UserList = strings.Split(value, ",")
		}
	}
	return nil
}
