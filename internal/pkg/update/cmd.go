package update

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/ProtonMail/go-crypto/openpgp"
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
		if _, err := openpgp.CheckArmoredDetachedSignature(el, bytes.NewReader(bSha256Sums), bytes.NewReader(bSha256SumsGpg), nil); err != nil {
			if _, err = openpgp.CheckDetachedSignature(el, bytes.NewReader(bSha256Sums), bytes.NewReader(bSha256SumsGpg), nil); err != nil {
				log.Error(err)
				return nil, err
			}
		}
	}

	return bSha256Sums, nil
}

func parseReleasesFromSha256Sums(bSha256Sums []byte) []Release {
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

func cmdList(fl Flags) error {
	bSha256Sums, err := getSha256Sums(fl.Provider, fl.CheckSignature, fl.Pubring)
	if err != nil {
		log.Error(err)
		return err
	}

	releases := parseReleasesFromSha256Sums(bSha256Sums)
	releases = filterReleases(releases, fl)

	// Print each release
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(tw, "Distribution\tArchitecture\tComponent\tVersion\tCompression")
	fmt.Fprintln(tw, "------------\t------------\t---------\t-------\t-----------")
	for _, r := range releases {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", r.Distribution, r.Architecture, r.Component, r.Version, r.Compression)
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

// TODO
func cmdUpdate(fl Flags) error {
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

	//TODO Validate SHA256

	//TODO Write release into output
	if lastVersion.Compression == "xz" {
		r, err := xz.NewReader(bRelease)
		if err != nil {
			log.Error(err)
			return err
		}
		releaseNameWithoutCompressionExtension := strings.TrimRight(releaseName, ".xz")
		output := filepath.Join(fl.Output, releaseNameWithoutCompressionExtension)

		fmt.Printf("Writing into %q ...\n", output)

		dst, err := os.Create(output)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, r); err != nil {
			return err
		}
	} else {
		err := fmt.Errorf("unknown compresssion %q", lastVersion.Compression)
		log.Error(err)
		return err
	}

	//TODO Bootloader

	return nil
}

func Cmd(args []string) error {
	log.Debug("start")
	defer log.Debug("end")

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
