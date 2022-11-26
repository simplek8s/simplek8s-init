package update

import (
	"errors"
	"flag"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	log "github.com/sirupsen/logrus"
)

const (
	// Common default flags

	defaultFlagArchitecture          = "x86-64"
	defaultFlagCheckSignature        = true
	defaultFlagComponent             = "kernel"
	defaultFlagDistribution          = "simplek8s"
	defaultFlagFilenameSha256sums    = "SHA256SUMS"
	defaultFlagFilenameSha256sumsGpg = "SHA256SUMS.gpg"
	defaultFlagProvider              = "https://simplek8s.jlsalvador.online/simplek8s/stable"
	defaultFlagPubring               = "/usr/lib/systemd/import-pubring.gpg"

	// Update default flags

	defaultFlagBootloader             = "auto"                   // could be: "syslinux", "rpi", or "auto"
	defaultFlagBootDevice             = "/dev/disk/by-label/EFI" //TODO On rpi4, the default value must be "/dev/disk/by-label/boot"
	defaultFlagComponentForUpdate     = "kernel"
	defaultFlagDryrun                 = false
	defaultFlagOverwrite              = false
	defaultFlagRelativeOutput         = "/simplek8s/"
	defaultFlagRelativeRpiConfig      = "/config.txt"
	defaultFlagRelativeSyslinuxConfig = "/syslinux/syslinux.cfg"
	defaultFlagRelativeUCode          = "/"
	defaultFlagVersion                = ""

	// Enums

	bootloaderAuto     = "auto"
	bootloaderSyslinux = "syslinux"
	bootloaderRpi      = "rpi"
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

	Bootloader             string
	BootDevice             string
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
		Distribution:          defaultFlagDistribution,
		Component:             defaultFlagComponent,
		CheckSignature:        defaultFlagCheckSignature,
		FilenameSha256sums:    defaultFlagFilenameSha256sums,
		FilenameSha256sumsGpg: defaultFlagFilenameSha256sumsGpg,
		Provider:              defaultFlagProvider,
		Pubring:               defaultFlagPubring,

		// Update flags
		Bootloader:             defaultFlagBootloader,
		BootDevice:             defaultFlagBootDevice,
		DryRun:                 defaultFlagDryrun,
		Overwrite:              defaultFlagOverwrite,
		RelativeOutput:         defaultFlagRelativeOutput,
		RelativeRpiConfig:      defaultFlagRelativeRpiConfig,
		RelativeSyslinuxConfig: defaultFlagRelativeSyslinuxConfig,
		RelativeUCode:          defaultFlagRelativeUCode,
		Version:                defaultFlagVersion,
	}
}

func flagSetCommon(flagSet *flag.FlagSet, fl *Flags) {
	flagSet.StringVar(&fl.Provider, "provider", fl.Provider, "Use a custom provider for updates")
	flagSet.StringVar(&fl.FilenameSha256sums, "filename-sha256sums", fl.FilenameSha256sums, "Filename with the checksums")
	flagSet.StringVar(&fl.FilenameSha256sumsGpg, "filename-sha256sums-gpg", fl.FilenameSha256sumsGpg, "Filename with the checksums PGP signature")
	flagSet.StringVar(&fl.Architecture, "architecture", fl.Architecture, "Platform architecture")
	flagSet.StringVar(&fl.Distribution, "distribution", fl.Distribution, "Distribution")
	flagSet.StringVar(&fl.Component, "component", fl.Component, "Component")
	flagSet.BoolVar(&fl.CheckSignature, "check-signature", fl.CheckSignature, "Check SHA256SUMS PGP signature")
	flagSet.StringVar(&fl.Pubring, "pubring", fl.Pubring, "Pubring to validate SHA256SUMS PGP signature")
}

func FlagParseList(args []string) (*Flags, error) {
	fl := NewFlags()

	// Common flags
	flagList := flag.NewFlagSet("list", flag.ExitOnError)
	flagSetCommon(flagList, fl)

	if err := flagList.Parse(args); err != nil {
		log.Error(err)
		return nil, err
	}

	return fl, nil
}

func FlagParseUpdate(args []string) (*Flags, error) {
	fl := NewFlags()
	fl.Component = defaultFlagComponentForUpdate

	// Common flags
	flagUpdate := flag.NewFlagSet("update", flag.ExitOnError)
	flagSetCommon(flagUpdate, fl)

	// Update flags
	flagUpdate.StringVar(&fl.Bootloader, "bootloader", fl.Bootloader, `Bootloader type to configure. Could be: "syslinux", "rpi", or "auto"`)
	flagUpdate.StringVar(&fl.BootDevice, "bootDevice", fl.BootDevice, "Device that contents the necessary to boot")
	flagUpdate.BoolVar(&fl.DryRun, "dry-run", fl.DryRun, "Do not write anything on disk")
	flagUpdate.StringVar(&fl.RelativeOutput, "output", fl.RelativeOutput, "Relative directory to boot device where to install the release")
	flagUpdate.BoolVar(&fl.Overwrite, "overwrite", fl.Overwrite, "Overwrite release filename")
	flagUpdate.StringVar(&fl.RelativeSyslinuxConfig, "syslinuxConfig", fl.RelativeSyslinuxConfig, "Relative filepath to boot device where is the syslinux.cfg")
	flagUpdate.StringVar(&fl.RelativeRpiConfig, "rpiConfig", fl.RelativeRpiConfig, "Relative filepath to boot device where is the config.cfg")
	flagUpdate.StringVar(&fl.RelativeUCode, "ucode", fl.RelativeUCode, "Relative directory to boot device where is the CPU microcode")
	flagUpdate.StringVar(&fl.Version, "version", fl.Version, "Version to download")

	if err := flagUpdate.Parse(args); err != nil {
		log.Error(err)
		return nil, err
	}

	// Validate flags
	if !common.IsStringInList(fl.Bootloader, []string{bootloaderAuto, bootloaderRpi, bootloaderSyslinux}) {
		return nil, errors.New("unknown bootloader type")
	}
	if fl.Distribution == "" {
		return nil, errors.New("distribution can not be empty")
	}
	if fl.Architecture == "" {
		return nil, errors.New("architecture can not be empty")
	}
	if fl.Component == "" {
		return nil, errors.New("component can not be empty")
	}

	return fl, nil
}

func FlagParseCurrent(args []string) (*Flags, error) {
	fl := NewFlags()
	return fl, nil
}
