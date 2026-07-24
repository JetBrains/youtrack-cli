package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/JetBrains/youtrack-cli/internal/config"
)

const (
	updateCheckInterval = 24 * time.Hour
	updateCheckTimeout  = 3 * time.Second
	updateCheckURL      = "https://api.github.com/repos/JetBrains/youtrack-cli/releases/latest"
)

// ReleaseInfo is the latest GitHub release, as reported by the GitHub API.
type ReleaseInfo struct {
	Version string // semver, no leading "v"
	URL     string
}

// UpdateStatus is the result of an explicit "youtrack update" check.
type UpdateStatus struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	UpgradeCommand  string `json:"upgradeCommand,omitempty"`
	ReleaseURL      string `json:"releaseUrl"`
}

// isUpdateCheckDisabled reports whether to skip the check entirely: env opt-out, CI, or non-TTY stderr.
func isUpdateCheckDisabled() bool {
	if os.Getenv("YOUTRACK_NO_UPDATE") != "" || os.Getenv("DO_NOT_TRACK") != "" || os.Getenv("CI") != "" {
		return true
	}
	return !term.IsTerminal(int(os.Stderr.Fd()))
}

// saveUpdateCheckTimestamp re-loads the config and saves the current timestamp
// to avoid race conditions with concurrent config writes (e.g., from instance add).
func saveUpdateCheckTimestamp(now int64) {
	cfg, err := config.Load()
	if err != nil {
		return
	}
	cfg.LastUpdateCheck = now
	_ = cfg.Save()
}

// CheckForUpdateAsync decides synchronously whether a check is due and, if so, persists
// the throttle timestamp up front so it survives an immediate exit (e.g. --quiet, or a
// wait shorter than the network round trip). The network fetch runs in the background.
// Returns a channel that receives at most one notice string then closes.
func CheckForUpdateAsync(currentVersion string) <-chan string {
	ch := make(chan string, 1)
	if !updateCheckDue(currentVersion) {
		close(ch)
		return ch
	}
	// Record the attempt now, before the network call, so any exit timing leaves the throttle persisted.
	saveUpdateCheckTimestamp(time.Now().Unix())
	go func() {
		defer close(ch)
		if notice := checkForUpdate(currentVersion); notice != "" {
			ch <- notice
		}
	}()
	return ch
}

// updateCheckDue reports whether a network update check should run now: a real build,
// not opted out, and past the throttle interval since the last recorded check.
func updateCheckDue(currentVersion string) bool {
	if currentVersion == "dev" || currentVersion == "unknown" {
		return false
	}
	if isUpdateCheckDisabled() {
		return false
	}
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	return time.Now().Unix()-cfg.LastUpdateCheck >= int64(updateCheckInterval.Seconds())
}

// fetchLatestRelease queries for the latest published release via the public,
// unauthenticated API first. If that fails (e.g. the repo is private/internal),
// it falls back to the gh CLI, which carries the caller's own GitHub auth.
func fetchLatestRelease() (*ReleaseInfo, error) {
	release, apiErr := fetchLatestReleaseViaAPI()
	if apiErr == nil {
		return release, nil
	}
	if !cmdExists("gh") {
		return nil, fmt.Errorf(
			"%w (install the gh CLI and run 'gh auth login' to check while the repository is private: https://cli.github.com/)",
			apiErr,
		)
	}
	return fetchLatestReleaseViaGH()
}

// fetchLatestReleaseViaGH shells out to the gh CLI. Callers must check cmdExists("gh") first.
func fetchLatestReleaseViaGH() (*ReleaseInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "api", "repos/JetBrains/youtrack-cli/releases/latest")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("gh api failed: %w: %s", err, msg)
		}
		return nil, fmt.Errorf("gh api failed: %w", err)
	}
	return parseReleaseJSON(out)
}

func fetchLatestReleaseViaAPI() (*ReleaseInfo, error) {
	req, err := http.NewRequest(http.MethodGet, updateCheckURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "youtrack-cli")

	client := &http.Client{Timeout: updateCheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseReleaseJSON(body)
}

func parseReleaseJSON(data []byte) (*ReleaseInfo, error) {
	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(data, &release); err != nil {
		return nil, err
	}
	if release.TagName == "" {
		return nil, errors.New("release response missing tag_name")
	}
	return &ReleaseInfo{
		Version: strings.TrimPrefix(release.TagName, "v"),
		URL:     release.HTMLURL,
	}, nil
}

func checkForUpdate(currentVersion string) string {
	release, err := fetchLatestRelease()
	if err != nil {
		return ""
	}
	current := strings.TrimPrefix(currentVersion, "v")
	if !IsNewerVersion(current, release.Version) {
		return ""
	}
	return fmt.Sprintf(
		"ℹ  YouTrack CLI v%s is available (you have %s) — run 'youtrack update' to update",
		release.Version, currentVersion,
	)
}

// CheckNow performs an explicit, synchronous update check that bypasses the throttle
// interval, for the "youtrack update" command. On success it persists the check
// timestamp so the passive background check doesn't immediately refire on the next run.
func CheckNow(currentVersion string) (*UpdateStatus, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		return nil, err
	}
	saveUpdateCheckTimestamp(time.Now().Unix())

	current := strings.TrimPrefix(currentVersion, "v")
	status := &UpdateStatus{
		CurrentVersion: current,
		LatestVersion:  release.Version,
		ReleaseURL:     release.URL,
	}
	if IsNewerVersion(current, release.Version) {
		status.UpdateAvailable = true
		status.UpgradeCommand = DetectInstallMethod().UpgradeCommand()
	}
	return status, nil
}

// FormatVersion prefixes v with "v" for display, except for the "dev"/"unknown"
// sentinel values set on non-release builds, which are shown as-is.
func FormatVersion(v string) string {
	if v == "dev" || v == "unknown" {
		return v
	}
	return "v" + v
}

// IsNewerVersion reports whether latest is a newer semantic version than current.
func IsNewerVersion(current, latest string) bool {
	curMajor, curMinor, curPatch := parseSemver(current)
	latMajor, latMinor, latPatch := parseSemver(latest)

	if latMajor != curMajor {
		return latMajor > curMajor
	}
	if latMinor != curMinor {
		return latMinor > curMinor
	}
	return latPatch > curPatch
}

func parseSemver(v string) (major, minor, patch int) {
	v = strings.TrimPrefix(v, "v")
	base, _, _ := strings.Cut(v, "-")
	parts := strings.SplitN(base, ".", 3)

	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patch, _ = strconv.Atoi(parts[2])
	}
	return
}
