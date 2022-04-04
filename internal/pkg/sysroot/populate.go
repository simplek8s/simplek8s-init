package sysroot

import (
	"os"
	"os/exec"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	log "github.com/sirupsen/logrus"
)

func createSymlinks(source string, destination string) error {
	os.Remove(destination)
	if err := os.Symlink(source, destination); err != nil {
		log.WithFields(log.Fields{
			"source":      source,
			"destination": destination,
		}).Error(err)
		return err
	}
	return nil
}

// Preserves symlinks, owner, permissions
func copyAll(source string, destination string) error {
	if err := os.MkdirAll(destination, 0755); err != nil {
		log.WithFields(log.Fields{
			"mkdir": destination,
		}).Error(err)
		return err
	}

	//TODO: Replace `cp` command by native Golang
	cmd := exec.Command("cp", "-pan", source, destination)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.WithFields(log.Fields{
			"cmd":    "cp",
			"args":   cmd.Args,
			"output": string(output),
		}).Error(err)
		return err
	}
	return nil
}

// Will populate sysroot
func Populate(output string) error {
	if err := common.IsDir(output); err != nil {
		return err
	}

	type srcDst struct {
		source      string
		destination string
	}

	symlinks := []srcDst{
		{"usr/bin", output + "/bin"},
		{"usr/lib", output + "/lib"},
		{"lib", output + "/lib64"},
		{"usr/sbin", output + "/sbin"},
	}
	for _, toLink := range symlinks {
		if err := createSymlinks(toLink.source, toLink.destination); err != nil {
			return err
		}
	}

	toBeCopied := []srcDst{
		{"/usr", output + "/"},
		{"/etc/ssl/certs", output + "/etc/ssl/"},
		{"/usr/share/factory/etc", output + "/"},
	}
	for _, toCopy := range toBeCopied {
		if err := copyAll(toCopy.source, toCopy.destination); err != nil {
			return err
		}
	}

	return nil
}
