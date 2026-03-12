// Package upgrade provides self-update functionality for cryptx-cli.
// The two-step handover works like this:
//
//  1. The running binary (v_old) downloads the new binary (v_new) to a temp
//     file, then exec-replaces itself with v_new passing
//     --upgrade-from <v_old_path>.
//
//  2. v_new sees the flag, copies itself to v_old_path, then exec-replaces
//     itself from that permanent path — completing the upgrade in-place.
package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const githubReleasesAPI = "https://api.github.com/repos/University-Of-Sri-Jayewardenepura/cryptx-cli/releases/latest"

// Release holds the fields we need from the GitHub Releases API response.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset is a single downloadable file attached to a GitHub release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// FetchLatest queries the GitHub Releases API and returns the latest release.
func FetchLatest() (*Release, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, githubReleasesAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "cryptx-cli-selfupdate")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("parsing release JSON: %w", err)
	}
	return &rel, nil
}

// AssetName returns the expected release-asset filename for the running OS/arch.
// This matches the naming produced by `make release`.
func AssetName() string {
	name := fmt.Sprintf("cryptx-cli-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// FindAsset returns the download URL for the asset matching the current
// platform, or an error if the release does not contain such an asset.
func FindAsset(rel *Release) (string, error) {
	want := AssetName()
	for _, a := range rel.Assets {
		if a.Name == want {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("release %s has no asset for %s", rel.TagName, want)
}

// Download fetches the binary at downloadURL into a temporary file and returns
// that file's path with executable permissions set.  The caller must remove
// the temp file on failure; on success it is handed to HandoverTo.
//
// Only URLs served from github.com or *.githubusercontent.com are accepted to
// guard against SSRF if the API response were ever tampered with.
func Download(downloadURL string) (string, error) {
	if err := validateDownloadURL(downloadURL); err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "cryptx-cli-selfupdate")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "cryptx-cli-update-*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(name)
		return "", err
	}
	tmp.Close()

	if err := os.Chmod(name, 0o755); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// ApplyFrom is called at the very start of main() when the binary detects it
// was launched with --upgrade-from <oldPath>.  It copies itself to oldPath,
// sets executable permissions, and then hands execution back to the newly
// installed binary at oldPath (via exec replacement so the PID is preserved
// on POSIX, or a fresh start on Windows).  This function never returns on
// success.
func ApplyFrom(oldPath string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve self path: %w", err)
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return fmt.Errorf("resolve self symlinks: %w", err)
	}

	if err := copyFile(self, oldPath); err != nil {
		return fmt.Errorf("install new binary: %w", err)
	}
	if err := os.Chmod(oldPath, 0o755); err != nil {
		return fmt.Errorf("chmod new binary: %w", err)
	}

	// Platform-specific exec replacement (see exec_posix.go / exec_windows.go).
	return execReplace(oldPath, []string{oldPath})
}

// HandoverTo exec-replaces the current process with newBin, passing it
// --upgrade-from <currentBin> so that newBin can install itself.
// This function never returns on success.
func HandoverTo(newBin, currentBin string) error {
	return execReplace(newBin, []string{newBin, "--upgrade-from", currentBin})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func validateDownloadURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("download URL must use HTTPS")
	}
	host := strings.ToLower(u.Hostname())
	if host != "github.com" &&
		!strings.HasSuffix(host, ".github.com") &&
		!strings.HasSuffix(host, ".githubusercontent.com") {
		return fmt.Errorf("refusing download from untrusted host %q", host)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
