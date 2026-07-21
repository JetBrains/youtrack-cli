package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

func checkForUpdate(currentVersion string) string {
	client := &http.Client{Timeout: updateCheckTimeout}
	resp, err := client.Get(updateCheckURL)
	if err != nil {
		return ""
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil || release.TagName == "" {
		return ""
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")
	if !isNewerVersion(current, latest) {
		return ""
	}
	return fmt.Sprintf(
		"ℹ  youtrack %s is available (you have %s) — see https://github.com/JetBrains/youtrack-cli/releases",
		release.TagName, currentVersion,
	)
}

// isNewerVersion reports whether latest is a newer semantic version than current.
func isNewerVersion(current, latest string) bool {
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
