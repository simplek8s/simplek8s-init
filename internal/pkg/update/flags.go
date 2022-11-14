package update

import (
	"errors"
	"flag"
)

const (
	// Common default flags

	defaultFlagProvider              = "https://simplek8s.jlsalvador.online/simplek8s/stable"
	defaultFlagFilenameSha256sums    = "SHA256SUMS"
	defaultFlagFilenameSha256sumsGpg = "SHA256SUMS.gpg"
	defaultFlagArchitecture          = "x86-64"
	defaultFlagDistribution          = "simplek8s"
	defaultFlagComponent             = "kernel"
	defaultFlagCheckSignature        = true
	defaultFlagPubring               = "/usr/lib/systemd/import-pubring.gpg"

	// Update default flags

	defaultFlagDryrun         = false
	defaultFlagOutput         = "/boot/simplek8s/"
	defaultFlagSyslinuxConfig = "/boot/syslinux/syslinux.cfg"
	defaultFlagVersion        = "latest"
)

type Flags struct {
	// Common flags

	Provider              string
	Architecture          string
	Distribution          string
	Component             string
	FilenameSha256sums    string
	FilenameSha256sumsGpg string
	CheckSignature        bool
	Pubring               string

	// Update flags

	DryRun         bool
	Output         string
	SyslinuxConfig string
	Version        string
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
		DryRun:         defaultFlagDryrun,
		Output:         defaultFlagOutput,
		SyslinuxConfig: defaultFlagSyslinuxConfig,
		Version:        defaultFlagVersion,
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
		return nil, err
	}

	return fl, nil
}

func FlagParseUpdate(args []string) (*Flags, error) {
	fl := NewFlags()

	// Common flags
	flagUpdate := flag.NewFlagSet("update", flag.ExitOnError)
	flagSetCommon(flagUpdate, fl)

	// Update flags
	flagUpdate.BoolVar(&fl.DryRun, "dry-run", fl.DryRun, "Do not write anything on disk")
	flagUpdate.StringVar(&fl.Output, "output", fl.Output, "Directory to install the release")
	flagUpdate.StringVar(&fl.SyslinuxConfig, "syslinuxConfig", fl.SyslinuxConfig, "Filepath to syslinux.cfg")
	flagUpdate.StringVar(&fl.Version, "version", fl.Version, "Version to download")

	if err := flagUpdate.Parse(args); err != nil {
		return nil, err
	}

	// Validate flags
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
