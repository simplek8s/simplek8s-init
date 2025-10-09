package feeder

import (
	"github.com/jlsalvador/simplek8s/pkg/common"
	"github.com/jlsalvador/simplek8s/pkg/linux/passwd"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/sysroot"
)

func feedSysrootByDefaultUsers(sr *sysroot.Sysroot) error {
	sameUser := func(a, b passwd.User) bool {
		return a.Name == b.Name
	}
	for _, u := range []passwd.User{
		passwd.NewUser(passwd.User{
			Name:  "root",
			Uid:   0,
			Gid:   0,
			Home:  "/root",
			Shell: "/usr/bin/sh",
		}),
		passwd.NewUser(passwd.User{
			Name:  "systemd-timesync",
			Uid:   978,
			Gid:   978,
			Gecos: []string{"systemd Time Synchronization"},
		}),
		passwd.NewUser(passwd.User{
			Name:  "systemd-resolve",
			Uid:   979,
			Gid:   979,
			Gecos: []string{"systemd Resolver"},
		}),
		passwd.NewUser(passwd.User{
			Name:  "systemd-network",
			Uid:   980,
			Gid:   980,
			Gecos: []string{"systemd Network Management"},
		}),
		passwd.NewUser(passwd.User{
			Name:  "sshd",
			Uid:   981,
			Gid:   981,
			Gecos: []string{"SSH drop priv user"},
		}),
		passwd.NewUser(passwd.User{
			Name:  "dbus",
			Uid:   982,
			Gid:   982,
			Gecos: []string{"System Message Bus"},
		}),
	} {
		sr.Users = common.UpdateOrAppend(sr.Users, u, sameUser)
	}
	return nil
}

func feedSysrootByDefaultGroups(sr *sysroot.Sysroot) error {
	sameGroup := func(a, b passwd.Group) bool {
		return a.Name == b.Name
	}
	for _, g := range []passwd.Group{
		passwd.NewGroup(passwd.Group{
			Name: "root",
			Gid:  0,
		}),
		passwd.NewGroup(passwd.Group{
			Name: "systemd-timesync",
			Gid:  978,
		}),
		passwd.NewGroup(passwd.Group{
			Name: "systemd-resolve",
			Gid:  979,
		}),
		passwd.NewGroup(passwd.Group{
			Name: "systemd-network",
			Gid:  980,
		}),
		passwd.NewGroup(passwd.Group{
			Name: "sshd",
			Gid:  981,
		}),
		passwd.NewGroup(passwd.Group{
			Name: "dbus",
			Gid:  982,
		}),
	} {
		sr.Groups = common.UpdateOrAppend(sr.Groups, g, sameGroup)
	}
	return nil
}

func FeedSysrootByDefault(sr *sysroot.Sysroot) error {
	if err := feedSysrootByDefaultUsers(sr); err != nil {
		return err
	}
	if err := feedSysrootByDefaultGroups(sr); err != nil {
		return err
	}
	return nil
}
