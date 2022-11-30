package update

import (
	"errors"
	"flag"
	"fmt"
	"runtime"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	log "github.com/sirupsen/logrus"
)

const (
	// Common default flags

	defaultFlagArchitecture          = "auto" // could be: "auto", "x86-64" or "arm64"
	defaultFlagCheckSignature        = true
	defaultFlagComponent             = "kernel"
	defaultFlagDistribution          = "simplek8s"
	defaultFlagFilenameSha256sums    = "SHA256SUMS"
	defaultFlagFilenameSha256sumsGpg = "SHA256SUMS.gpg"
	defaultFlagProvider              = "https://simplek8s.jlsalvador.online/simplek8s/stable"
	defaultFlagPubring               = "/usr/lib/systemd/import-pubring.gpg"

	// Update default flags

	defaultFlagBootDevice             = "auto" // ex: "/dev/disk/by-label/EFI"
	defaultFlagBootloader             = "auto" // could be: "syslinux", "rpi", or "auto"
	defaultFlagComponentForUpdate     = "kernel"
	defaultFlagDryrun                 = false
	defaultFlagOverwrite              = false
	defaultFlagRelativeOutput         = "/simplek8s/"
	defaultFlagRelativeRpiConfig      = "/config.txt"
	defaultFlagRelativeSyslinuxConfig = "/syslinux/syslinux.cfg"
	defaultFlagRelativeUCode          = "/"
	defaultFlagVersion                = "" // ex: "198612212230"

	// Enums

	bootloaderAuto     = "auto"
	bootloaderRpi      = "rpi"
	bootloaderSyslinux = "syslinux"
)

type Flags struct {
	// Common flags

	Architecture          string
	CheckSignature        bool
	Component             string
	Distribution          string
	FilenameSha256sums    string
	FilenameSha256sumsGpg string
	Provider              string
	Pubring               string

	// Update flags

	BootDevice             string
	Bootloader             string
	DryRun                 bool
	Overwrite              bool
	RelativeOutput         string
	RelativeRpiConfig      string
	RelativeSyslinuxConfig string
	RelativeUCode          string
	Version                string
}

// Returns `Flags` with default values
func NewFlags() *Flags {
	return &Flags{
		// Common flags

		Architecture:          defaultFlagArchitecture,
		CheckSignature:        defaultFlagCheckSignature,
		Component:             defaultFlagComponent,
		Distribution:          defaultFlagDistribution,
		FilenameSha256sums:    defaultFlagFilenameSha256sums,
		FilenameSha256sumsGpg: defaultFlagFilenameSha256sumsGpg,
		Provider:              defaultFlagProvider,
		Pubring:               defaultFlagPubring,

		// Update flags

		BootDevice:             defaultFlagBootDevice,
		Bootloader:             defaultFlagBootloader,
		DryRun:                 defaultFlagDryrun,
		Overwrite:              defaultFlagOverwrite,
		RelativeOutput:         defaultFlagRelativeOutput,
		RelativeRpiConfig:      defaultFlagRelativeRpiConfig,
		RelativeSyslinuxConfig: defaultFlagRelativeSyslinuxConfig,
		RelativeUCode:          defaultFlagRelativeUCode,
		Version:                defaultFlagVersion,
	}
}

func flagSetCommon(fl *Flags) error {
	flag.CommandLine.StringVar(&fl.Provider, "provider", fl.Provider, "Use a custom provider for updates")
	flag.CommandLine.StringVar(&fl.FilenameSha256sums, "filename-sha256sums", fl.FilenameSha256sums, "Filename with the checksums")
	flag.CommandLine.StringVar(&fl.FilenameSha256sumsGpg, "filename-sha256sums-gpg", fl.FilenameSha256sumsGpg, "Filename with the checksums PGP signature")
	flag.CommandLine.StringVar(&fl.Architecture, "architecture", fl.Architecture, `Platform architecture. Could be: "auto", "x86-64" or "arm64"`)
	flag.CommandLine.StringVar(&fl.Distribution, "distribution", fl.Distribution, "Distribution")
	flag.CommandLine.StringVar(&fl.Component, "component", fl.Component, "Component")
	flag.CommandLine.BoolVar(&fl.CheckSignature, "check-signature", fl.CheckSignature, "Check SHA256SUMS PGP signature")
	flag.CommandLine.StringVar(&fl.Pubring, "pubring", fl.Pubring, "Pubring to validate SHA256SUMS PGP signature")

	// Compute flags values
	if fl.Architecture == "auto" {
		switch runtime.GOARCH {
		case "amd64":
			fl.Architecture = "x86-64"
		case "arm64":
			fl.Architecture = "arm64"
		default:
			err := fmt.Errorf("unknown architecture %q", runtime.GOARCH)
			log.Error(err)
			return err
		}
	}

	return nil
}

func FlagParseList(args []string) (*Flags, error) {
	log.WithFields(log.Fields{
		"start": "FlagParseList",
		"args":  args,
	}).Debug()
	defer log.WithField("end", "FlagParseList").Debug()

	fl := NewFlags()

	// Common flags
	if err := flagSetCommon(fl); err != nil {
		log.Error(err)
		return nil, err
	}

	// List flags
	if err := flag.CommandLine.Parse(args); err != nil {
		log.Error(err)
		return nil, err
	}

	return fl, nil
}

func FlagParseUpdate(args []string) (*Flags, error) {
	log.WithFields(log.Fields{
		"start": "FlagParseUpdate",
		"args":  args,
	}).Debug()
	defer log.WithField("end", "FlagParseUpdate").Debug()

	fl := NewFlags()
	fl.Component = defaultFlagComponentForUpdate

	// Common flags
	if err := flagSetCommon(fl); err != nil {
		log.Error(err)
		return nil, err
	}

	// Update flags
	flag.CommandLine.StringVar(&fl.Bootloader, "bootloader", fl.Bootloader, `Bootloader type to configure. Could be: "syslinux", "rpi", or "auto"`)
	flag.CommandLine.StringVar(&fl.BootDevice, "bootDevice", fl.BootDevice, "Device that contents the necessary to boot")
	flag.CommandLine.BoolVar(&fl.DryRun, "dry-run", fl.DryRun, "Do not write anything on disk")
	flag.CommandLine.StringVar(&fl.RelativeOutput, "output", fl.RelativeOutput, "Relative directory to boot device where to install the release")
	flag.CommandLine.BoolVar(&fl.Overwrite, "overwrite", fl.Overwrite, "Overwrite release filename")
	flag.CommandLine.StringVar(&fl.RelativeSyslinuxConfig, "syslinuxConfig", fl.RelativeSyslinuxConfig, "Relative filepath to boot device where is the syslinux.cfg")
	flag.CommandLine.StringVar(&fl.RelativeRpiConfig, "rpiConfig", fl.RelativeRpiConfig, "Relative filepath to boot device where is the config.cfg")
	flag.CommandLine.StringVar(&fl.RelativeUCode, "ucode", fl.RelativeUCode, "Relative directory to boot device where is the CPU microcode")
	flag.CommandLine.StringVar(&fl.Version, "version", fl.Version, "Version to download")
	if err := flag.CommandLine.Parse(args); err != nil {
		log.Error(err)
		return nil, err
	}

	// Compute flags values
	if fl.BootDevice == "auto" {
		if common.CheckFileExists("/dev/disk/by-label/EFI") {
			fl.BootDevice = "/dev/disk/by-label/EFI"
		} else if common.CheckFileExists("/dev/disk/by-label/boot") {
			fl.BootDevice = "/dev/disk/by-label/boot"
		} else {
			err := errors.New("can not detect the boot device")
			log.Error(err)
			return nil, err
		}
	}

	// Validate flags
	if !common.IsStringInList(fl.Bootloader, []string{bootloaderAuto, bootloaderRpi, bootloaderSyslinux}) {
		err := errors.New("unknown bootloader type")
		log.Error(err)
		return nil, err
	}
	if fl.Distribution == "" {
		err := errors.New("distribution can not be empty")
		log.Error(err)
		return nil, err
	}
	if fl.Architecture == "" {
		err := errors.New("architecture can not be empty")
		log.Error(err)
		return nil, err
	}
	if fl.Component == "" {
		err := errors.New("component can not be empty")
		log.Error(err)
		return nil, err
	}

	return fl, nil
}

func FlagParseCurrent(args []string) (*Flags, error) {
	log.WithFields(log.Fields{
		"start": "FlagParseCurrent",
		"args":  args,
	}).Debug()
	defer log.WithField("end", "FlagParseCurrent").Debug()

	fl := NewFlags()
	return fl, nil
}
