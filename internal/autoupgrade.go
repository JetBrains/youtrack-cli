package internal

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const upgradeTimeout = 10 * time.Minute

const (
	installScriptURL   = "https://jb.gg/youtrack-cli/install"
	installScriptPSURL = "https://jb.gg/youtrack-cli/install.ps1"
)

// AutoUpgradeCommand returns the argv for a scoped, non-interactive upgrade command
// for install methods whose package manager owns just this one package. ok is false
// for methods with no safe single-package upgrade path — apt/rpm ship one-off files
// with no in-place upgrade, and the AUR helper command syncs the whole system — and
// for InstallUnknown. An install-script install re-runs the official installer, as
// a new install does: it verifies checksums.txt (and its signature when gpg is
// installed) before it replaces the binary in the directory of the running one.
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
	case InstallScript:
		if runtime.GOOS == "windows" {
			// Detection accepts only %USERPROFILE%\.local\bin, the default directory of install.ps1.
			return "powershell", []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
				"irm " + installScriptPSURL + " | iex"}, true
		}
		exe, err := os.Executable()
		if err != nil {
			return "", nil, false
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		// $0 and $1 carry the URL and directory, so the path is never parsed as shell code.
		return "bash", []string{"-c", `curl -fsSL "$0" | bash -s -- "" "$1"`, installScriptURL, filepath.Dir(exe)}, true
	default:
		return "", nil, false
	}
}

// RunAutoUpgrade attempts an automated upgrade through the given install method,
// streaming the underlying command's output to out/errOut. attempted is false when
// the method has no safe automated path — callers should fall back to printing
// manual instructions in that case.
func RunAutoUpgrade(method InstallMethod, version string, out, errOut io.Writer) (attempted bool, err error) {
	return RunAutoUpgradeContext(context.Background(), method, version, out, errOut)
}

func RunAutoUpgradeContext(parent context.Context, method InstallMethod, _ string, out, errOut io.Writer) (attempted bool, err error) {
	name, args, ok := method.AutoUpgradeCommand()
	if !ok {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(parent, upgradeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	if method == InstallScript {
		// The installers then skip the banner and the first-run "Next steps".
		cmd.Env = append(os.Environ(), "YOUTRACK_INSTALL_UPDATE=1")
	}
	cmd.Stdout = out
	cmd.Stderr = errOut
	return true, cmd.Run()
}
