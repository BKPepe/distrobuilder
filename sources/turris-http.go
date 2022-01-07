package sources

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	lxd "github.com/lxc/lxd/shared"

	"github.com/lxc/distrobuilder/shared"
)

type turris struct {
	common
}

// Run downloads the tarball and unpacks it.
func (s *turris) Run() error {
	var baseURL string

	release := s.definition.Image.Release
	releaseInFilename := strings.ToLower(release) + "-"

	var architecturePath string

	switch s.definition.Image.ArchitectureMapped {
	case "armv7l":
		architecturePath = "omnia"
	case "aarch64":
		architecturePath = "mox"
	case "powerpc":
		architecturePath = "turris1x"
	}

	// Figure out the branch (release)
	if release == "hbs" {
		// Build a daily snapshot.
		baseURL = fmt.Sprintf("%s/%s/medkit/",
			s.definition.Source.URL, architecturePath)
		releaseInFilename = ""
	} else {

		baseURL = fmt.Sprintf("%s/%s/%s/medkit", baseURL, s.definition.Source.URL, architecturePath)
	}

	var fname string

	if release == "hbs" {
		switch s.definition.Image.ArchitectureMapped {
		
		case "armv7l":
			fname = fmt.Sprintf("%s-medkit-latest.tar.gz", releaseInFilename,
				strings.Replace(architecturePath, "/", "-", 1))
		case "aarch64":
			fname = fmt.Sprintf("%s-medkit-latest.tar.gz", releaseInFilename,
				strings.Replace(architecturePath, "/", "-", 1))
		case "powerpc":
			fname = fmt.Sprintf("%s-medkit-latest.tar.gz", releaseInFilename,
				strings.Replace(architecturePath, "/", "-", 1))
		}
	}

	var (
		resp *http.Response
		err  error
	)

	err = shared.Retry(func() error {
		resp, err = http.Head(baseURL)
		if err != nil {
			return fmt.Errorf("Failed to HEAD %q: %w", baseURL, err)
		}

		return nil
	}, 3)
	if err != nil {
		return nil
	}

	url, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("Failed to parse %q: %w", baseURL, err)
	}

	checksumFile := "%-medkit-latest.tar.gz.sha256"
	if !s.definition.Source.SkipVerification {
		if len(s.definition.Source.Keys) != 0 {
			checksumFile = baseURL + "sha256sums"
			_, err := s.DownloadHash(s.definition.Image, checksumFile, "", nil)
			if err != nil {
				return fmt.Errorf("Failed to download %q: %w", checksumFile, err)
			}
		} else {
			// Force gpg checks when using http
			if url.Scheme != "https" {
				return errors.New("GPG keys are required if downloading from HTTP")
			}
		}
	}

	fpath, err := s.DownloadHash(s.definition.Image, baseURL+fname, checksumFile, sha256.New())
	if err != nil {
		return fmt.Errorf("Failed to download %q: %w", baseURL+fname, err)
	}

	s.logger.WithField("file", filepath.Join(fpath, fname)).Info("Unpacking image")

	// Unpack
	err = lxd.Unpack(filepath.Join(fpath, fname), s.rootfsDir, false, false, nil)
	if err != nil {
		return fmt.Errorf("Failed to unpack %q: %w", filepath.Join(fpath, fname), err)
	}

	return nil
}
