package internal

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const installScriptURL = "https://github.com/JetBrains/youtrack-cli/releases/latest/download/install.sh"

// GHInstallScriptCommand is the channel-independent upgrade path: re-running the
// install script through gh, which works even while the repo is private. Shown as
// an alternative whenever the detected install method has no automated upgrade.
const GHInstallScriptCommand = "gh release download --repo JetBrains/youtrack-cli --pattern install.sh -O - | bash"

// AutoUpgradeCommand returns the argv for a scoped, non-interactive upgrade command
// for install methods whose package manager owns just this one package. ok is false
// for methods with no safe single-package upgrade path — apt/rpm ship one-off files
// with no in-place upgrade, and the AUR helper command syncs the whole system — and
// for InstallScript/InstallUnknown, which RunAutoUpgrade handles separately.
func (m InstallMethod) AutoUpgradeCommand() (name string, args []string, ok bool) {
	switch m {
	case InstallHomebrew:
		return "brew", []string{"upgrade", "youtrack"}, true
	case InstallScoop:
		return "scoop", []string{"update", "youtrack"}, true
	case InstallChocolatey:
		return "choco", []string{"upgrade", "YouTrackCLI", "-y"}, true
	case InstallWinGet:
		return "winget", []string{
			"upgrade", "--id", "JetBrains.YouTrackCLI", "--silent",
			"--accept-package-agreements", "--accept-source-agreements",
		}, true
	case InstallNPM:
		return "npm", []string{"update", "-g", "@jetbrains/youtrack-cli"}, true
	case InstallGoInstall:
		return "go", []string{"install", "github.com/JetBrains/youtrack-cli@latest"}, true
	default:
		return "", nil, false
	}
}

// RunAutoUpgrade attempts an automated upgrade through the given install method,
// streaming the underlying command's output to out/errOut. attempted is false when
// the method has no safe automated path — callers should fall back to printing
// manual instructions in that case.
func RunAutoUpgrade(method InstallMethod, out, errOut io.Writer) (attempted bool, err error) {
	if method == InstallScript {
		return true, runInstallScriptUpgrade(out, errOut)
	}
	name, args, ok := method.AutoUpgradeCommand()
	if !ok {
		return false, nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = out
	cmd.Stderr = errOut
	return true, cmd.Run()
}

// runInstallScriptUpgrade re-runs install.sh targeting the directory that already
// holds the running binary. Without this, install.sh's own default (/usr/local/bin,
// falling back to ~/.local/bin only if that's unwritable) can silently diverge from
// wherever this binary actually lives, leaving a second, unused copy behind instead
// of replacing the one on PATH.
func runInstallScriptUpgrade(out, errOut io.Writer) error {
	script, err := downloadInstallScript()
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	outDir := filepath.Dir(exe)

	// bash -s reads the script from stdin; everything after "--" becomes $1, $2, ...
	// RELEASE ($1) is left empty so install.sh resolves the latest release itself.
	cmd := exec.Command("bash", "-s", "--", "", outDir)
	cmd.Stdin = bytes.NewReader(script)
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}

// downloadInstallScript fetches install.sh's contents, preferring the public URL
// (works once the repo is public) and falling back to gh (works while it's private).
func downloadInstallScript() ([]byte, error) {
	if body, err := downloadInstallScriptViaAPI(); err == nil {
		return body, nil
	}
	if !cmdExists("gh") {
		return nil, errors.New("install script not reachable and gh CLI not found")
	}
	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "gh", "release", "download",
		"--repo", "JetBrains/youtrack-cli", "--pattern", "install.sh", "-O", "-").Output()
}

func downloadInstallScriptViaAPI() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, installScriptURL, nil)
	if err != nil {
		return nil, err
	}
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
		return nil, errors.New("install script URL returned a non-200 status")
	}
	return io.ReadAll(resp.Body)
}

// HasGH reports whether the gh CLI is available on PATH.
func HasGH() bool {
	return cmdExists("gh")
}
