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

// isUpdateCheckDisabled reports whether the update check should be skipped
// entirely, with no network call: opted out via env var, running in CI, or
// stderr isn't a TTY (e.g. piped output, non-interactive scripts).
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

// CheckForUpdateAsync starts a background goroutine that checks for a newer
// GitHub release. Returns a channel that receives at most one notice string
// then closes. The caller should drain the channel after the command finishes.
func CheckForUpdateAsync(currentVersion string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		if notice := checkForUpdate(currentVersion); notice != "" {
			ch <- notice
		}
	}()
	return ch
}

func checkForUpdate(currentVersion string) string {
	if currentVersion == "dev" || currentVersion == "unknown" {
		return ""
	}
	if isUpdateCheckDisabled() {
		return ""
	}

	cfg, err := config.Load()
	if err != nil {
		return ""
	}

	now := time.Now().Unix()
	if now-cfg.LastUpdateCheck < int64(updateCheckInterval.Seconds()) {
		return ""
	}

	client := &http.Client{Timeout: updateCheckTimeout}
	resp, err := client.Get(updateCheckURL)
	if err != nil {
		saveUpdateCheckTimestamp(now)
		return ""
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var release struct {
		TagName string `json:"tag_name"`
	}
	// Persist the throttle timestamp regardless of decode success so a
	// malformed or unexpected feed doesn't force a check on every invocation.
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil || release.TagName == "" {
		saveUpdateCheckTimestamp(now)
		return ""
	}
	saveUpdateCheckTimestamp(now)

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
