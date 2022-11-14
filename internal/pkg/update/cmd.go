package update

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"text/tabwriter"

	"github.com/ProtonMail/go-crypto/openpgp"
	log "github.com/sirupsen/logrus"
)

const (
	SUBCMD_LIST    = "list"
	SUBCMD_UPDATE  = "update"
	SUBCMD_CURRENT = "current"
)

// TODO
func cmdCurrent(fl Flags) error {
	panic("unimplemented")
}

func httpGetFile(u ...string) (io.ReadCloser, error) {
	var endpoint string
	var err error
	if len(u) > 1 {
		if endpoint, err = url.JoinPath(u[0], u[1:]...); err != nil {
			return nil, err
		}
	} else if len(u) == 1 {
		if endpoint, err = url.JoinPath(u[0]); err != nil {
			return nil, err
		}
	} else {
		err = fmt.Errorf("invalid arguments")
		return nil, err
	}

	if resp, err := http.Get(endpoint); err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status for %q is not ok, got: %v", u, resp.StatusCode)
	} else {
		return resp.Body, nil
	}
}

func getSHA256Sums(provider string, checkSignature bool, pubring string) ([]byte, error) {
	Sha256sums, err := httpGetFile(provider, "SHA256SUMS")
	if err != nil {
		return nil, err
	}
	defer Sha256sums.Close()

	bSha256Sums, err := io.ReadAll(Sha256sums)
	if err != nil {
		return nil, err
	}

	// Validate SHA256SUMS PGP signature
	if !checkSignature {
		fmt.Printf("SHA256SUMS.gpg ignored!\n\n")
	} else {
		// Get SHA256SUMS.gpg
		Sha256SumsGpg, err := httpGetFile(provider, "SHA256SUMS.gpg")
		if err != nil {
			return nil, err
		}
		defer Sha256SumsGpg.Close()

		bSha256SumsGpg, err := io.ReadAll(Sha256SumsGpg)
		if err != nil {
			return nil, err
		}

		// Load PGP public keyring
		var el openpgp.EntityList
		if f, err := os.Open(pubring); err != nil {
			return nil, err
		} else {
			defer f.Close()

			if el, err = openpgp.ReadArmoredKeyRing(f); err != nil {
				f.Seek(0, 0)
				if el, err = openpgp.ReadKeyRing(f); err != nil {
					return nil, err
				}
			}
		}

		// Check SHA256SUMS PGP signature
		if err != nil {
			return nil, err
		}
		if _, err := openpgp.CheckArmoredDetachedSignature(el, bytes.NewReader(bSha256Sums), bytes.NewReader(bSha256SumsGpg), nil); err != nil {
			if _, err = openpgp.CheckDetachedSignature(el, bytes.NewReader(bSha256Sums), bytes.NewReader(bSha256SumsGpg), nil); err != nil {
				return nil, err
			}
		}
	}

	return bSha256Sums, nil
}

// TODO
func cmdList(fl Flags) error {
	bSha256Sums, err := getSHA256Sums(fl.Provider, fl.CheckSignature, fl.Pubring)
	if err != nil {
		return err
	}

	type Release struct {
		Distribution string
		Version      string
		Checksum     string
		Architecture string
		Component    string
		Compression  string
	}
	releases := []Release{}

	// Parse "component", "version", "installed", "available" from SHA256SUMS
	bs := bufio.NewScanner(bytes.NewReader(bSha256Sums))
	re := regexp.MustCompile(`(?P<checksum>\w+)\s+\*?(?P<distribution>[-_\w]+)\.(?P<version>[-_\w]+)\.(?P<architecture>[-_\w]+)(\.(?P<component>\w+))?(\.(?P<compression>\w+))?`)
	for bs.Scan() {
		line := bs.Text()
		match := re.FindStringSubmatch(line)

		// Check for "index out of range"
		if len(match) != len(re.SubexpNames()) {
			fmt.Printf("Can regexp the line %q!\n", line)
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

	// Print each release
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(tw, "Distribution\tArchitecture\tComponent\tVersion\tCompression")
	fmt.Fprintln(tw, "------------\t------------\t---------\t-------\t-----------")
	for _, r := range releases {

		// Skip another architectures
		if len(fl.Architecture) > 0 && fl.Architecture != r.Architecture {
			continue
		}

		// Skip another distros
		if len(fl.Distribution) > 0 && fl.Distribution != r.Distribution {
			continue
		}

		// Skip another components
		if len(fl.Component) > 0 && fl.Component != r.Component {
			continue
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", r.Distribution, r.Architecture, r.Component, r.Version, r.Compression)
	}
	tw.Flush()

	//TODO panic("unimplemented")
	return nil
}

// TODO
func cmdUpdate(fl Flags) error {
	//TODO: bSha256Sums
	_, err := getSHA256Sums(fl.Provider, fl.CheckSignature, fl.Pubring)
	if err != nil {
		return err
	}

	panic("unimplemented")
}

// TODO
func Cmd(args []string) error {
	log.Debug("start")
	defer log.Debug("end")

	//TODO: Show help
	if len(args) > 1 {
		switch args[1] {
		case SUBCMD_LIST:
			fl, _ := FlagParseList(args[2:])
			if err := cmdList(*fl); err != nil {
				return err
			}
		case SUBCMD_UPDATE:
			fl, _ := FlagParseUpdate(args[2:])
			if err := cmdUpdate(*fl); err != nil {
				return err
			}
		case SUBCMD_CURRENT:
			fl, _ := FlagParseCurrent(args[2:])
			if err := cmdCurrent(*fl); err != nil {
				return err
			}
		}
	}
	return nil
}
