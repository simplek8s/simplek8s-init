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

// Package feeder will fill a sysroot.Sysroot{} with values from files.
package feeder

import (
	sr "simplek8s/pkg/simplek8s/sysroot"
)

func feedSysrootByDefaultUsers(sysroot *sr.Sysroot) error {
	// sameUser := func(a, b passwd.User) bool {
	// 	return a.Name == b.Name
	// }
	// for _, u := range []passwd.User{
	// 	passwd.NewUser(passwd.User{
	// 		Name:  "root",
	// 		Uid:   0,
	// 		Gid:   0,
	// 		Home:  "/root",
	// 		Shell: "/usr/bin/sh",
	// 	}),
	// 	passwd.NewUser(passwd.User{
	// 		Name:  "systemd-timesync",
	// 		Uid:   978,
	// 		Gid:   978,
	// 		Gecos: []string{"systemd Time Synchronization"},
	// 	}),
	// 	passwd.NewUser(passwd.User{
	// 		Name:  "systemd-resolve",
	// 		Uid:   979,
	// 		Gid:   979,
	// 		Gecos: []string{"systemd Resolver"},
	// 	}),
	// 	passwd.NewUser(passwd.User{
	// 		Name:  "systemd-network",
	// 		Uid:   980,
	// 		Gid:   980,
	// 		Gecos: []string{"systemd Network Management"},
	// 	}),
	// 	passwd.NewUser(passwd.User{
	// 		Name:  "sshd",
	// 		Uid:   981,
	// 		Gid:   981,
	// 		Gecos: []string{"SSH drop priv user"},
	// 	}),
	// 	passwd.NewUser(passwd.User{
	// 		Name:  "dbus",
	// 		Uid:   982,
	// 		Gid:   982,
	// 		Gecos: []string{"System Message Bus"},
	// 	}),
	// } {
	// 	sysroot.Users = common.UpdateOrAppend(sysroot.Users, u, sameUser)
	// }
	return nil
}

func feedSysrootByDefaultGroups(sysroot *sr.Sysroot) error {
	// sameGroup := func(a, b passwd.Group) bool {
	// 	return a.Name == b.Name
	// }
	// for _, g := range []passwd.Group{
	// 	passwd.NewGroup(passwd.Group{
	// 		Name: "root",
	// 		Gid:  0,
	// 	}),
	// 	passwd.NewGroup(passwd.Group{
	// 		Name: "systemd-timesync",
	// 		Gid:  978,
	// 	}),
	// 	passwd.NewGroup(passwd.Group{
	// 		Name: "systemd-resolve",
	// 		Gid:  979,
	// 	}),
	// 	passwd.NewGroup(passwd.Group{
	// 		Name: "systemd-network",
	// 		Gid:  980,
	// 	}),
	// 	passwd.NewGroup(passwd.Group{
	// 		Name: "sshd",
	// 		Gid:  981,
	// 	}),
	// 	passwd.NewGroup(passwd.Group{
	// 		Name: "dbus",
	// 		Gid:  982,
	// 	}),
	// } {
	// 	sysroot.Groups = common.UpdateOrAppend(sysroot.Groups, g, sameGroup)
	// }
	return nil
}

func FeedSysrootByDefault(sysroot *sr.Sysroot) error {
	for _, fn := range []func(sysroot *sr.Sysroot) error{
		feedSysrootByDefaultUsers,
		feedSysrootByDefaultGroups,
	} {
		if err := fn(sysroot); err != nil {
			return err
		}
	}
	return nil
}
