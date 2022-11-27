package update

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	sdbus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/coreos/go-systemd/v22/unit"
	"github.com/godbus/dbus/v5"
	"github.com/jlsalvador/simplek8s/internal/pkg/checksum"
	"github.com/jlsalvador/simplek8s/internal/pkg/common"
	log "github.com/sirupsen/logrus"
	"github.com/ulikunitz/xz"
)

const (
	SUBCMD_LIST    = "list"
	SUBCMD_UPDATE  = "update"
	SUBCMD_CURRENT = "current"
)

type Release struct {
	Distribution string
	Version      string
	Checksum     string
	Architecture string
	Component    string
	Compression  string
}

// TODO
func cmdCurrent(fl Flags) error {
	panic("unimplemented")
}

// Fetch a HTTP resource and returns its ReadCloser interface.
func httpGetFile(u ...string) (io.ReadCloser, error) {
	log.WithFields(log.Fields{
		"start": "httpGetFile",
		"u":     u,
	}).Debug()
	defer log.WithField("end", "httpGetFile").Debug()

	var endpoint string
	var err error
	if len(u) > 1 {
		if endpoint, err = url.JoinPath(u[0], u[1:]...); err != nil {
			log.Error(err)
			return nil, err
		}
	} else if len(u) == 1 {
		if endpoint, err = url.JoinPath(u[0]); err != nil {
			log.Error(err)
			return nil, err
		}
	} else {
		err = fmt.Errorf("invalid arguments")
		log.Error(err)
		return nil, err
	}

	if resp, err := http.Get(endpoint); err != nil {
		log.Error(err)
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("http status for %q is not ok, got: %v", u, resp.StatusCode)
		log.Error(err)
		return nil, err
	} else {
		return resp.Body, nil
	}
}

// Fetch the SHA256SUMS from the provider, check its GPG signature using
// entities from the pubring and returns its []byte content.
//
// Example:
//
//   - provider = "http://simplek8s.jlsalvador.online/simplek8s/dev/"
//   - checkSignature = true
//   - pubring = "/usr/lib/systemd/import-pubring.gpg"
func getSha256Sums(provider string, checkSignature bool, pubring string) ([]byte, error) {
	log.WithFields(log.Fields{
		"start":          "getSha256Sums",
		"provider":       provider,
		"checkSignature": checkSignature,
		"pubring":        pubring,
	}).Debug()
	defer log.WithField("end", "getSha256Sums").Debug()

	Sha256sums, err := httpGetFile(provider, "SHA256SUMS")
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer Sha256sums.Close()

	bSha256Sums, err := io.ReadAll(Sha256sums)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Validate SHA256SUMS PGP signature
	if !checkSignature {
		fmt.Printf("SHA256SUMS.gpg ignored!\n\n")
	} else {
		// Get SHA256SUMS.gpg
		Sha256SumsGpg, err := httpGetFile(provider, "SHA256SUMS.gpg")
		if err != nil {
			log.Error(err)
			return nil, err
		}
		defer Sha256SumsGpg.Close()

		bSha256SumsGpg, err := io.ReadAll(Sha256SumsGpg)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		// Load PGP public keyring
		var el openpgp.EntityList
		if f, err := os.Open(pubring); err != nil {
			log.Error(err)
			return nil, err
		} else {
			defer f.Close()

			if el, err = openpgp.ReadArmoredKeyRing(f); err != nil {
				f.Seek(0, 0)
				if el, err = openpgp.ReadKeyRing(f); err != nil {
					log.Error(err)
					return nil, err
				}
			}
		}

		// Check SHA256SUMS PGP signature
		if err != nil {
			log.Error(err)
			return nil, err
		}
		if _, err := openpgp.CheckArmoredDetachedSignature(
			el,
			bytes.NewReader(bSha256Sums),
			bytes.NewReader(bSha256SumsGpg),
			nil,
		); err != nil {
			if _, err = openpgp.CheckDetachedSignature(
				el,
				bytes.NewReader(bSha256Sums),
				bytes.NewReader(bSha256SumsGpg),
				nil,
			); err != nil {
				log.Error(err)
				return nil, err
			}
		}
	}

	return bSha256Sums, nil
}

func parseReleasesFromSha256Sums(bSha256Sums []byte) []Release {
	log.WithFields(log.Fields{
		"start":       "parseReleasesFromSha256Sums",
		"bSha256Sums": bSha256Sums,
	}).Debug()
	defer log.WithField("end", "parseReleasesFromSha256Sums").Debug()

	releases := []Release{}
	re := regexp.MustCompile(`(?P<checksum>\w+)\s+\*?(?P<distribution>[-_\w]+)\.(?P<version>[-_\w]+)\.(?P<architecture>[-_\w]+)(\.(?P<component>\w+))?(\.(?P<compression>\w+))?`)
	bs := bufio.NewScanner(bytes.NewReader(bSha256Sums))
	for bs.Scan() {
		line := bs.Text()
		match := re.FindStringSubmatch(line)

		// Check for "index out of range"
		if len(match) != len(re.SubexpNames()) {
			log.Warnf("Can regexp the line %q!\n", line)
			continue
		}

		releases = append(releases, Release{
			Distribution: match[re.SubexpIndex("distribution")],
			Component:    match[re.SubexpIndex("component")],
			Version:      match[re.SubexpIndex("version")],
			Checksum:     match[re.SubexpIndex("checksum")],
			Architecture: match[re.SubexpIndex("architecture")],
			Compression:  match[re.SubexpIndex("compression")],
		})
	}

	// Sort releases
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Version < releases[j].Version
	})

	return releases
}

func filterReleases(releases []Release, fl Flags) []Release {
	log.WithFields(log.Fields{
		"start":    "filterReleases",
		"releases": releases,
		"fl":       fl,
	}).Debug()
	defer log.WithField("end", "filterReleases").Debug()

	newReleases := []Release{}
	for i := range releases {
		if (len(fl.Architecture) > 0 && fl.Architecture != releases[i].Architecture) ||
			// Filter by distribution
			(len(fl.Distribution) > 0 && fl.Distribution != releases[i].Distribution) ||
			// Filter by component
			(len(fl.Component) > 0 && fl.Component != releases[i].Component) ||
			// Filter by version
			(len(fl.Version) > 0 && fl.Version != releases[i].Version) ||
			// Skip "latest" versions if the version filter is not "latest", because is an alias
			(fl.Version != "latest" && releases[i].Version == "latest") {
			continue
		}
		newReleases = append(newReleases, releases[i])
	}
	return newReleases
}

// TODO Mark current in the list
// TODO Hide some headers if flags filter are set
func cmdList(fl Flags) error {
	log.WithFields(log.Fields{
		"start": "cmdList",
		"fl":    fl,
	}).Debug()
	defer log.WithField("end", "cmdList").Debug()

	bSha256Sums, err := getSha256Sums(fl.Provider, fl.CheckSignature, fl.Pubring)
	if err != nil {
		log.Error(err)
		return err
	}

	releases := parseReleasesFromSha256Sums(bSha256Sums)
	releases = filterReleases(releases, fl)

	// Print each release
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(tw, "DISTRIBUTION\tARCHITECTURE\tCOMPONENT\tVERSION")
	for _, r := range releases {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.Distribution, r.Architecture, r.Component, r.Version)
	}
	tw.Flush()

	return nil
}

func getReleaseFilename(release Release) string {
	compression := ""
	if release.Compression != "" {
		compression = "." + release.Compression
	}
	return fmt.Sprintf(
		"%s.%s.%s.%s%s",
		release.Distribution,
		release.Version,
		release.Architecture,
		release.Component,
		compression,
	)
}

func mountBootPartition(device string) (mountPath string, err error) {
	log.WithFields(log.Fields{
		"start":  "mountBootPartition",
		"device": device,
	}).Debug()
	defer log.WithField("end", "mountBootPartition").Debug()

	timeout := time.Duration(time.Second * 30)

	// Generate unique mount name
	timestamp := time.Now().Unix()
	mountPath = "/run/media/root/boot" + strconv.FormatInt(timestamp, 10)
	unitName := fmt.Sprintf("%s.mount", unit.UnitNamePathEscape(mountPath))

	// Connect to Systemd DBUS
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := sdbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	defer conn.Close()

	ch := make(chan string)

	deviceEscaped := unit.UnitNamePathEscape(device)
	deviceUnitName := fmt.Sprintf("%s.device", deviceEscaped)
	blockdevUnitName := fmt.Sprintf("blockdev@%s.target", deviceEscaped)

	_, err = conn.StartTransientUnitContext(ctx, unitName, "replace", []sdbus.Property{
		sdbus.PropDescription(mountPath),
		sdbus.PropRequires("system.slice", "-.mount", deviceUnitName),
		sdbus.PropAfter("system.slice", "systemd-journald.socket", "-.mount", deviceUnitName, "local-fs-pre.target", blockdevUnitName),
		sdbus.PropBefore("local-fs.target", "umount.target"),
		sdbus.PropConflicts("umount.target"),
		{Name: "What", Value: dbus.MakeVariant(device)},
	}, ch)
	if err != nil {
		log.Error(err)
		return
	}
	result := <-ch
	if result != "done" {
		err = fmt.Errorf("starting systemd unit %q got %q", unitName, result)
		log.Error(err)
		return
	}

	return
}

func setBootloaderVersionRpi(
	pathBoot string,
	relativePathKernel string,
	relativePathRpiConfig string,
) error {
	log.WithFields(log.Fields{
		"start":                 "setBootloaderVersionRpi",
		"pathBoot":              pathBoot,
		"relativePathKernel":    relativePathKernel,
		"relativePathRpiConfig": relativePathRpiConfig,
	}).Debug()
	defer log.WithField("end", "setBootloaderVersionRpi").Debug()

	pathRpiConfig := filepath.Join(pathBoot, relativePathRpiConfig)

	fo, err := os.CreateTemp("", "tmp-rpiconfig-*")
	if err != nil {
		log.Error(err)
		return err
	}
	defer func() {
		// Remove temporal
		if err := fo.Close(); err != nil {
			log.Error(err)
			//TODO return err
		}
		if err := os.Remove(fo.Name()); err != nil {
			log.Error(err)
			//TODO return err
		}
	}()

	reKernel := regexp.MustCompile(`(?i)^kernel=`)

	fi, err := os.OpenFile(pathRpiConfig, os.O_RDONLY, 0644)
	if err != nil {
		log.Error(err)
		return err
	}
	s := bufio.NewScanner(fi)
	for i := 0; s.Scan(); i++ {
		line := s.Text()

		if reKernel.MatchString(line) {
			if _, err := fo.WriteString(fmt.Sprintf("kernel=%s\n", relativePathKernel)); err != nil {
				log.Error(err)
				return err
			}
			continue
		}

		if _, err := fo.WriteString(line + "\n"); err != nil {
			log.Error(err)
			return err
		}
	}
	if err := fi.Close(); err != nil {
		log.Error(err)
		return err
	}

	// Copy from temporal to final
	if _, err := fo.Seek(0, 0); err != nil {
		log.Error(err)
		return err
	}
	if f, err := os.OpenFile(
		pathRpiConfig,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_SYNC,
		0644,
	); err != nil {
		log.Error(err)
		return err
	} else if _, err := io.Copy(f, fo); err != nil {
		log.Error(err)
		return err
	} else if err := f.Close(); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func setBootloaderVersionSyslinux(
	pathBoot string,
	relativePathKernel string,
	relativePathMicrocode string,
	relativePathSyslinuxConfig string,
) error {
	log.WithFields(log.Fields{
		"start":                      "setBootloaderVersionSyslinux",
		"pathBoot":                   pathBoot,
		"relativePathKernel":         relativePathKernel,
		"relativePathMicrocode":      relativePathMicrocode,
		"relativePathSyslinuxConfig": relativePathSyslinuxConfig,
	}).Debug()
	defer log.WithField("end", "setBootloaderVersionSyslinux").Debug()

	pathSyslinuxConfig := filepath.Join(pathBoot, relativePathSyslinuxConfig)

	fo, err := os.CreateTemp("", "tmp-syslinux-*")
	if err != nil {
		log.Error(err)
		return err
	}
	defer func() {
		// Remove temporal
		if err := fo.Close(); err != nil {
			log.Error(err)
			//TODO return err
		}
		if err := os.Remove(fo.Name()); err != nil {
			log.Error(err)
			//TODO return err
		}
	}()

	reDefault := regexp.MustCompile(`(?i)^DEFAULT `)
	kernelName := strings.TrimSuffix(path.Base(relativePathKernel), ".kernel")
	reLabel := regexp.MustCompile(`(?i)^LABEL ` + strings.ReplaceAll(kernelName, `.`, `\.`))

	fi, err := os.OpenFile(pathSyslinuxConfig, os.O_RDONLY, 0644)
	if err != nil {
		log.Error(err)
		return err
	}
	foundLabel := false
	s := bufio.NewScanner(fi)
	for i := 0; s.Scan(); i++ {
		line := s.Text()

		if reDefault.MatchString(line) {
			if _, err := fo.WriteString(fmt.Sprintf("DEFAULT %s\n", kernelName)); err != nil {
				log.Error(err)
				return err
			}
			continue
		}

		if reLabel.MatchString(line) {
			foundLabel = true
		}

		if _, err := fo.WriteString(line + "\n"); err != nil {
			log.Error(err)
			return err
		}
	}
	if err := fi.Close(); err != nil {
		log.Error(err)
		return err
	}

	if !foundLabel {
		if _, err := fo.WriteString(fmt.Sprintf("LABEL %s\n KERNEL %s\n", kernelName, relativePathKernel)); err != nil {
			log.Error(err)
			return err
		}
		for _, microcodeFilename := range []string{"intel-ucode.img", "amd-ucode.img"} {
			if common.CheckFileExists(filepath.Join(pathBoot, relativePathMicrocode, microcodeFilename)) {
				if _, err := fo.WriteString(fmt.Sprintf(" INITRD %s\n", filepath.Join(relativePathMicrocode, microcodeFilename))); err != nil {
					log.Error(err)
					return err
				}
			}
		}
		if _, err := fo.WriteString("\n"); err != nil {
			log.Error(err)
			return err
		}
	}

	// Copy from temporal to final
	if _, err := fo.Seek(0, 0); err != nil {
		log.Error(err)
		return err
	}
	if f, err := os.OpenFile(
		pathSyslinuxConfig,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_SYNC,
		0644,
	); err != nil {
		log.Error(err)
		return err
	} else if _, err := io.Copy(f, fo); err != nil {
		log.Error(err)
		return err
	} else if err := f.Close(); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// Detect bootloaders and set version to boot
func setBootloaderVersion(
	pathBoot string,
	relativePathKernel string,
	relativePathMicrocode string,
	bootloader string,
) error {
	log.WithFields(log.Fields{
		"start":                 "setBootloaderVersion",
		"pathBoot":              pathBoot,
		"relativePathKernel":    relativePathKernel,
		"relativePathMicrocode": relativePathMicrocode,
		"bootloader":            bootloader,
	}).Debug()
	defer log.WithField("end", "setBootloaderVersion").Debug()

	relativePathRpiConfig := "/config.txt"
	relativePathSyslinuxConfig := "/syslinux/syslinux.cfg"

	switch bootloader {
	case bootloaderSyslinux:
		return setBootloaderVersionSyslinux(
			pathBoot,
			relativePathKernel,
			relativePathMicrocode,
			relativePathSyslinuxConfig,
		)
	case bootloaderRpi:
		return setBootloaderVersionRpi(
			pathBoot,
			relativePathKernel,
			relativePathRpiConfig,
		)
	case bootloaderAuto:
		if common.CheckFileExists(filepath.Join(pathBoot, relativePathRpiConfig)) {
			return setBootloaderVersion(
				pathBoot,
				relativePathKernel,
				relativePathMicrocode,
				bootloaderRpi,
			)
		} else if common.CheckFileExists(filepath.Join(pathBoot, relativePathSyslinuxConfig)) {
			return setBootloaderVersion(
				pathBoot,
				relativePathKernel,
				relativePathMicrocode,
				bootloaderSyslinux,
			)
		}
		fallthrough
	default:
		err := errors.New("unknown bootloader")
		return err
	}
}

func umountBootPartition(mountPath string) error {
	log.WithFields(log.Fields{
		"start":     "umountBootPartition",
		"mountPath": mountPath,
	}).Debug()
	defer log.WithField("end", "umountBootPartition").Debug()

	timeout := time.Duration(time.Second * 30)
	unitName := fmt.Sprintf("%s.mount", unit.UnitNamePathEscape(mountPath))

	// Connect to Systemd DBUS
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := sdbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		log.Error(err)
		return err
	}
	defer conn.Close()

	ch := make(chan string)
	if _, err := conn.StopUnitContext(ctx, unitName, "fail", ch); err != nil {
		log.Error(err)
		return err
	}

	result := <-ch
	if result != "done" {
		err = fmt.Errorf("stopping systemd unit %q got %q", unitName, result)
		log.Error(err)
		return err
	}

	return nil
}

func cmdUpdate(fl Flags) error {
	log.WithFields(log.Fields{
		"start": "cmdUpdate",
		"fl":    fl,
	}).Debug()
	defer log.WithField("end", "cmdUpdate").Debug()

	//TODO DryRun
	//TODO Overwrite

	bSha256Sums, err := getSha256Sums(fl.Provider, fl.CheckSignature, fl.Pubring)
	if err != nil {
		log.Error(err)
		return err
	}

	releases := parseReleasesFromSha256Sums(bSha256Sums)
	releases = filterReleases(releases, fl)
	lastVersion := releases[len(releases)-1]

	releaseName := getReleaseFilename(lastVersion)
	bRelease, err := httpGetFile(fl.Provider, releaseName)
	if err != nil {
		log.Error(err)
		return err
	}
	defer bRelease.Close()

	// Validate SHA256
	buf := bytes.Buffer{}
	r := io.TeeReader(bRelease, &buf)
	fmt.Println("Downloading and verifying ...")
	hash, err := checksum.Sha256sum(r)
	if err != nil {
		log.Error(err)
		return err
	}
	if hash != lastVersion.Checksum {
		err := fmt.Errorf("checksum invalid. got: %q, want: %q", hash, lastVersion.Checksum)
		log.Error(err)
		return err
	}
	fmt.Println("Checksum is valid.")

	// Mount the "boot" partition as RW
	fmt.Println("Mounting boot device ...")
	pathMount, err := mountBootPartition(fl.BootDevice)
	if err != nil {
		log.Error(err)
		return err
	}
	defer func() {
		// Unmount "boot" partition if it is necessary
		fmt.Println("Unmounting boot device ...")
		if err := umountBootPartition(pathMount); err != nil {
			log.Error(err)
			//TODO return err
		}
		if err := os.Remove(pathMount); err != nil {
			log.Error(err)
			//TODO return err
		}
	}()

	// Detect compression
	var releaseNameWithoutCompressionExtension string
	var lastVersionReader io.Reader
	switch lastVersion.Compression {
	case "xz": // XZ
		lastVersionReader, err = xz.NewReader(&buf)
		if err != nil {
			log.Error(err)
			return err
		}
		releaseNameWithoutCompressionExtension = strings.TrimRight(releaseName, ".xz")
	case "": // None
		lastVersionReader = &buf
		releaseNameWithoutCompressionExtension = releaseName
	default:
		err := fmt.Errorf("unknown compression %q", lastVersion.Compression)
		log.Error(err)
		return err
	}

	// Write release

	pathKernel := filepath.Join(
		pathMount,
		fl.RelativeOutput,
		releaseNameWithoutCompressionExtension,
	)

	// Create the destination directory
	dir := filepath.Join(pathMount, fl.RelativeOutput)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Error(err)
		return err
	}

	// Copy the temporal file to the final destination
	fmt.Printf("Writing into %q ...\n", pathKernel)
	if dst, err := os.OpenFile(
		pathKernel,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_SYNC,
		0644,
	); err != nil {
		log.Error(err)
		return err
	} else if _, err := io.Copy(dst, lastVersionReader); err != nil {
		log.Error(err)
		return err
	} else if err := dst.Close(); err != nil {
		log.Error(err)
		return err
	}

	// Bootloader
	fmt.Println("Configuring bootloader ...")
	if err := setBootloaderVersion(
		pathMount,
		filepath.Join(fl.RelativeOutput, releaseNameWithoutCompressionExtension),
		fl.RelativeUCode,
		fl.Bootloader,
	); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func Cmd(args []string) error {
	log.WithFields(log.Fields{
		"start": "Cmd",
		"args":  args,
	}).Debug()
	defer log.WithField("end", "Cmd").Debug()

	//TODO: Show help
	if len(args) > 1 {
		switch args[1] {
		case SUBCMD_LIST:
			fl, err := FlagParseList(args[2:])
			if err != nil {
				log.Error(err)
				return err
			}
			if err := cmdList(*fl); err != nil {
				log.Error(err)
				return err
			}
		case SUBCMD_UPDATE:
			fl, err := FlagParseUpdate(args[2:])
			if err != nil {
				log.Error(err)
				return err
			}
			if err := cmdUpdate(*fl); err != nil {
				log.Error(err)
				return err
			}
		case SUBCMD_CURRENT:
			fl, err := FlagParseCurrent(args[2:])
			if err != nil {
				log.Error(err)
				return err
			}
			if err := cmdCurrent(*fl); err != nil {
				log.Error(err)
				return err
			}
		}
	}
	return nil
}
