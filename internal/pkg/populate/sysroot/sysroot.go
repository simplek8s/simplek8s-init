// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package sysroot

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/jlsalvador/simplek8s/internal/pkg/populate/templates"
	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
	"github.com/jlsalvador/simplek8s/pkg/common"
	"github.com/jlsalvador/simplek8s/pkg/linux/passwd"
	"github.com/jlsalvador/simplek8s/pkg/simplek8s/bootstrap"
	log "github.com/sirupsen/logrus"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
)

// The users that any OS with systemd must have.
var defaultUsers = []passwd.User{
	passwd.NewUser(passwd.User{
		Name:  "root",
		Uid:   0,
		Gid:   0,
		Gecos: []string{"Super User"},
		Home:  "/root",
		Shell: "/usr/bin/sh",
	}),
	passwd.NewUser(passwd.User{
		Name:  "nobody",
		Uid:   65534,
		Gid:   65534,
		Gecos: []string{"Nobody"},
	}),
	passwd.NewUser(passwd.User{
		Name:  "dbus",
		Uid:   982,
		Gid:   982,
		Gecos: []string{"System Message Bus"},
	}),
	passwd.NewUser(passwd.User{
		Name:  "sshd",
		Uid:   981,
		Gid:   981,
		Gecos: []string{"SSH drop priv user"},
		Home:  "/var/empty",
	}),
	passwd.NewUser(passwd.User{
		Name:  "systemd-network",
		Uid:   980,
		Gid:   980,
		Gecos: []string{"systemd Network Management"},
	}),
	passwd.NewUser(passwd.User{
		Name:  "systemd-resolve",
		Uid:   979,
		Gid:   979,
		Gecos: []string{"systemd Resolver"},
	}),
	passwd.NewUser(passwd.User{
		Name:  "systemd-timesync",
		Uid:   978,
		Gid:   978,
		Gecos: []string{"systemd Time Synchronization"},
	}),
}

var defaultGroups = []passwd.Group{
	passwd.NewGroup(passwd.Group{Name: "root", Gid: 0}),
	passwd.NewGroup(passwd.Group{Name: "nobody", Gid: 65534}),
	passwd.NewGroup(passwd.Group{Name: "adm", Gid: 999}),
	passwd.NewGroup(passwd.Group{Name: "wheel", Gid: 998}),
	passwd.NewGroup(passwd.Group{Name: "utmp", Gid: 997}),
	passwd.NewGroup(passwd.Group{Name: "audio", Gid: 996}),
	passwd.NewGroup(passwd.Group{Name: "cdrom", Gid: 995}),
	passwd.NewGroup(passwd.Group{Name: "dialout", Gid: 994}),
	passwd.NewGroup(passwd.Group{Name: "disk", Gid: 993}),
	passwd.NewGroup(passwd.Group{Name: "input", Gid: 992}),
	passwd.NewGroup(passwd.Group{Name: "kmem", Gid: 991}),
	passwd.NewGroup(passwd.Group{Name: "kvm", Gid: 990}),
	passwd.NewGroup(passwd.Group{Name: "lp", Gid: 989}),
	passwd.NewGroup(passwd.Group{Name: "render", Gid: 988}),
	passwd.NewGroup(passwd.Group{Name: "sgx", Gid: 987}),
	passwd.NewGroup(passwd.Group{Name: "tape", Gid: 986}),
	passwd.NewGroup(passwd.Group{Name: "tty", Gid: 5}),
	passwd.NewGroup(passwd.Group{Name: "video", Gid: 985}),
	passwd.NewGroup(passwd.Group{Name: "users", Gid: 984}),
	passwd.NewGroup(passwd.Group{Name: "systemd-journal", Gid: 983}),
	passwd.NewGroup(passwd.Group{Name: "dbus", Gid: 982}),
	passwd.NewGroup(passwd.Group{Name: "sshd", Gid: 981}),
	passwd.NewGroup(passwd.Group{Name: "systemd-network", Gid: 980}),
	passwd.NewGroup(passwd.Group{Name: "systemd-resolve", Gid: 979}),
	passwd.NewGroup(passwd.Group{Name: "systemd-timesync", Gid: 978}),
}

// Configure "/usr/share/factory/etc/ssh/sshd_config"
func populateSysrootSshd(output string, sr *sysroot.Sysroot, allowRootPassword bool) error {
	log.WithFields(log.Fields{
		"output":            output,
		"sr":                sr,
		"allowRootPassword": allowRootPassword,
	}).Debug("start")
	defer log.Debug("end")

	// Generate content
	tmplName := "assets/templates/usr/share/factory/etc/ssh/sshd_config.go.tmpl"
	tmplData := struct{ AllowRootPassword bool }{allowRootPassword}
	content, err := common.RenderTemplate(templates.Templates, tmplName, tmplData)
	if err != nil {
		log.WithFields(log.Fields{
			"tmplName": tmplName,
			"tmplData": tmplData,
		}).Error(err)
		return err
	}

	// Write file
	sr.Files = append(sr.Files, sysroot.File{
		Overwrite: false,
		Filename:  "/etc/ssh/sshd_config",
		Content:   content,
		Mode:      0644,
		Uid:       0,
		Gid:       0,
	})
	return nil
}

func hashPassword(plainPassword string) (string, error) {
	// Generate a random string for use in the salt
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	s := make([]byte, 8)
	for i := range s {
		s[i] = charset[seededRand.Intn(len(charset))]
	}
	salt := []byte(fmt.Sprintf("$6$%s", s))
	// use salt to hash user-supplied password
	c := sha512_crypt.New()
	hash, err := c.Generate([]byte(plainPassword), salt)
	if err != nil {
		log.Errorf("error hashing user's supplied password: %s\n", err)
		return "", err
	}
	return string(hash), nil
}

func generatePasswordPlainHashed() (string, string, error) {
	log.Debug("start")
	defer log.Debug("end")

	// Kernel debug mode
	if common.IsCmdlineDebug() {
		plain := "root"
		hashed, err := hashPassword(plain)
		if err != nil {
			return "", "", err
		}
		return plain, hashed, nil
	}

	plain := strings.TrimRight(gofakeit.Sentence(5), ".")
	hashed, err := hashPassword(plain)
	if err != nil {
		return "", "", err
	}
	return plain, hashed, nil
}

// Ensure a "root" user, passwd and shadow entry
func populateSysrootUsers(output string, sr *sysroot.Sysroot, generateRootPassword bool) error {
	log.WithFields(log.Fields{
		"output":           output,
		"sr":               sr,
		"generatePassword": generateRootPassword,
	}).Debug("start")
	defer log.Debug("end")

	// Lets set the "root" password
	pwdHashed := "x"
	if generateRootPassword {
		// Generate a random "root" password
		var pwdPlain string
		var err error

		pwdPlain, pwdHashed, err = generatePasswordPlainHashed()
		if err != nil {
			log.Error(err)
			return err
		}

		// Write the "root" random plain password into "/run/issue.d/"
		tmplData := struct{ Password string }{pwdPlain}
		if content, err := common.RenderTemplate(templates.Templates, "assets/templates/run/issue.d/80-root-random-password.issue.go.tmpl", tmplData); err != nil {
			log.Error(err)
			return err
		} else {
			sr.Files = append(sr.Files, sysroot.File{
				Overwrite: false,
				Filename:  "/run/issue.d/80-root-random-password.issue",
				Content:   content,
				Mode:      0644,
				Uid:       0,
				Gid:       0,
			})
		}
	}

	// Generate shadows by `defaultUsers`, but with a `root` hashed password
	for _, user := range defaultUsers {
		var shadow passwd.Shadow
		if user.Name == "root" {
			shadow = passwd.NewShadow(passwd.Shadow{
				Name:     "root",
				Password: pwdHashed,
			})
		} else {
			shadow = passwd.NewShadow(passwd.Shadow{Name: user.Name})
		}
		sr.Shadows = append(sr.Shadows, shadow)
	}

	// Fill default groups
	sr.Groups = append(sr.Groups, defaultGroups...)

	// Fill default users
	sr.Users = append(sr.Users, defaultUsers...)

	return nil
}

func CmdPopulateSysroot(output string) error {
	log.WithFields(log.Fields{
		"output": output,
	}).Debug("start")
	defer log.Debug("end")

	config, err := bootstrap.GetConfig()
	if err != nil {
		log.WithField("output", output).Error(err)
		return err
	}
	sshAllowRootPassword := config == nil
	generateRootPassword := config == nil

	sr := sysroot.Sysroot{}

	if err := populateSysrootSshd(output, &sr, sshAllowRootPassword); err != nil {
		log.WithFields(log.Fields{
			"output":               output,
			"sr":                   sr,
			"sshAllowRootPassword": sshAllowRootPassword,
		}).Error(err)
		return err
	}

	if err := populateSysrootUsers(output, &sr, generateRootPassword); err != nil {
		log.WithFields(log.Fields{
			"output":           output,
			"sr":               sr,
			"generatePassword": generateRootPassword,
		}).Error(err)
		return err
	}

	if config != nil {
		if err := sr.FeedByCurrentFiles(); err != nil {
			log.Error(err)
			return err
		}
		if err := sr.FeedByBootstrapConfig(*config); err != nil {
			log.Error(err)
			return err
		}
	}

	// Commit sysroot
	if err := sr.Write(output); err != nil {
		log.Error(err)
		return err
	}

	//TODO: Maybe systemd daemon-reload

	return nil
}
