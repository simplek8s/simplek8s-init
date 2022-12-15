package passwd

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type Shadow struct {
	// It is your login name.
	Name string

	// It is your encrypted password. The password should be minimum
	// 8-12 characters long including special characters, digits,
	// lower case alphabetic and more. Usually password format is set
	// to `$id$salt$hashed`.
	Password string

	// Days since Jan 1, 1970 that password was last changed.
	LastChanged *int

	// The minimum number of days required between password changes
	// i.e. the number of days left before the user is allowed to
	// change his/her password.
	Minimum *int

	// The maximum number of days the password is valid (after that
	// user is forced to change his/her password).
	Maximum *int

	// The number of days before password is to expire that user is
	// warned.
	Warn *int

	// The number of days after password expires that account is
	// disabled.
	Inactive *int

	// Days since Jan 1, 1970 that account is disabled i.e. an absolute
	// date specifying when the login may no longer be used.
	Expire *int

	// Reserved for future use
	Reserved *int
}

// Calculate how many days there are between now and 1970.
func getNumberOfDaysFrom1970() int {
	begining := time.Unix(0, 0)
	diff := time.Since(begining)
	days := math.Ceil(diff.Hours() / 24)
	return int(days)
}

func NewShadow(shadow Shadow) Shadow {
	if len(shadow.Name) == 0 {
		panic("name is required")
	}

	if len(shadow.Password) == 0 {
		shadow.Password = "!!"
	}

	if shadow.LastChanged == nil {
		lastChanged := getNumberOfDaysFrom1970()
		shadow.LastChanged = &lastChanged
	}

	return shadow
}

func (shadow Shadow) Marshal() (string, error) {
	name := shadow.Name
	if len(shadow.Name) == 0 {
		return "", errors.New("name is required")
	}

	password := shadow.Password
	if len(shadow.Password) == 0 {
		return "", errors.New(`password is required (ex: could be "!!")`)
	}

	var lastChanged string
	if shadow.LastChanged == nil {
		return "", errors.New(`lastChanged is required`)
	} else {
		lastChanged = fmt.Sprint(*shadow.LastChanged)
	}

	minimum := ""
	if shadow.Minimum != nil {
		minimum = fmt.Sprint(*shadow.Minimum)
	}

	maximum := ""
	if shadow.Maximum != nil {
		maximum = fmt.Sprint(*shadow.Maximum)
	}

	warn := ""
	if shadow.Warn != nil {
		warn = fmt.Sprint(*shadow.Warn)
	}

	inactive := ""
	if shadow.Inactive != nil {
		inactive = fmt.Sprint(*shadow.Inactive)
	}

	expire := ""
	if shadow.Expire != nil {
		expire = fmt.Sprint(*shadow.Expire)
	}

	reserved := ""
	if shadow.Reserved != nil {
		reserved = fmt.Sprint(*shadow.Reserved)
	}

	return strings.Join([]string{
		name, password, lastChanged, minimum, maximum, warn, inactive, expire, reserved,
	}, ":"), nil
}

func UnmarshalShadow(entry string, shadow *Shadow) error {
	for index, value := range strings.Split(entry, ":") {
		switch index {
		case 0:
			shadow.Name = value
		case 1:
			shadow.Password = value
		case 2:
			if len(value) > 0 {
				valueAsInt, err := strconv.Atoi(value)
				if err != nil {
					return err
				}
				shadow.LastChanged = &valueAsInt
			}
		case 3:
			if len(value) > 0 {
				valueAsInt, err := strconv.Atoi(value)
				if err != nil {
					return err
				}
				shadow.Minimum = &valueAsInt
			}
		case 4:
			if len(value) > 0 {
				valueAsInt, err := strconv.Atoi(value)
				if err != nil {
					return err
				}
				shadow.Maximum = &valueAsInt
			}
		case 5:
			if len(value) > 0 {
				valueAsInt, err := strconv.Atoi(value)
				if err != nil {
					return err
				}
				shadow.Warn = &valueAsInt
			}
		case 6:
			if len(value) > 0 {
				valueAsInt, err := strconv.Atoi(value)
				if err != nil {
					return err
				}
				shadow.Inactive = &valueAsInt
			}
		case 7:
			if len(value) > 0 {
				valueAsInt, err := strconv.Atoi(value)
				if err != nil {
					return err
				}
				shadow.Expire = &valueAsInt
			}
		case 8:
			if len(value) > 0 {
				valueAsInt, err := strconv.Atoi(value)
				if err != nil {
					return err
				}
				shadow.Reserved = &valueAsInt
			}
		}
	}
	return nil
}
