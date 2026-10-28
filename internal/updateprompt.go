package internal

import (
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/JetBrains/youtrack-cli/internal/config"
)

// PendingUpdate reports whether the startup prompt should offer latest: a release
// build, a newer cached release, and not the release the user chose to skip.
func PendingUpdate(current, latest, dismissed string) bool {
	if current == "dev" || current == "unknown" || latest == "" || latest == dismissed {
		return false
	}
	return IsNewerVersion(strings.TrimPrefix(current, "v"), latest)
}

// UpdatePromptEnabled reports whether a person can answer the startup prompt: update
// checks are not disabled, and stdin, stdout, and stderr are all terminals. Agents,
// pipes, and CI never see the prompt.
func UpdatePromptEnabled() bool {
	return !isUpdateCheckDisabled() &&
		term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// DismissUpdate records version as skipped, so the startup prompt stays quiet until a newer release.
func DismissUpdate(version string) {
	_, _ = config.Update(func(c *config.Config) error {
		c.DismissedVersion = version
		return nil
	})
}
