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
	beginning := time.Unix(0, 0)
	diff := time.Since(beginning)
	days := math.Ceil(diff.Hours() / 24)
	return int(days)
}

// NewShadow creates a Shadow with safe default values.
//   - if password is empty, will be set as "!!"
//   - if lastChanged is nil, will be set as current number of days from 1970.
func NewShadow(shadow Shadow) Shadow {
	if shadow.Name == "" {
		panic("name is required")
	}

	if shadow.Password == "" {
		shadow.Password = "!!"
	}

	if shadow.LastChanged == nil {
		lastChanged := getNumberOfDaysFrom1970()
		shadow.LastChanged = &lastChanged
	}

	return shadow
}

func intPtrToString(p *int) string {
	if p == nil {
		return ""
	}
	return strconv.Itoa(*p)
}

func (shadow Shadow) Marshal() (string, error) {
	if shadow.Name == "" {
		return "", errors.New("name is required")
	}

	fields := []string{
		shadow.Name,
		shadow.Password,
		intPtrToString(shadow.LastChanged),
		intPtrToString(shadow.Minimum),
		intPtrToString(shadow.Maximum),
		intPtrToString(shadow.Warn),
		intPtrToString(shadow.Inactive),
		intPtrToString(shadow.Expire),
		intPtrToString(shadow.Reserved),
	}

	return strings.Join(fields, ":"), nil
}

func parseIntPtr(s string) (*int, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func UnmarshalShadow(entry string, shadow *Shadow) error {
	tokens := strings.Split(entry, ":")

	const numExpectedFields = 9
	numFields := len(tokens)
	if numFields != numExpectedFields {
		return fmt.Errorf("invalid number of fields in entry: %s, got: %d, want: %d", entry, numFields, numExpectedFields)
	}

	shadow.Name = tokens[0]
	shadow.Password = tokens[1]
	for i := 2; i < numExpectedFields; i++ {
		v, err := parseIntPtr(tokens[i])
		if err != nil {
			return err
		}
		switch i {
		case 2:
			shadow.LastChanged = v
		case 3:
			shadow.Minimum = v
		case 4:
			shadow.Maximum = v
		case 5:
			shadow.Warn = v
		case 6:
			shadow.Inactive = v
		case 7:
			shadow.Expire = v
		case 8:
			shadow.Reserved = v
		}
	}

	return nil
}
