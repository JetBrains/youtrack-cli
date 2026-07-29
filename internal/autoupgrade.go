package internal

import (
	"context"
	"io"
	"os/exec"
	"time"
)

const upgradeTimeout = 10 * time.Minute

// GHInstallScriptCommand is the channel-independent upgrade path: re-running the
// install script through gh, which works even while the repo is private. Shown as
// an alternative whenever the detected install method has no automated upgrade.
const GHInstallScriptCommand = "gh release download --repo JetBrains/youtrack-cli --pattern install.sh -O - | bash"

// AutoUpgradeCommand returns the argv for a scoped, non-interactive upgrade command
// for install methods whose package manager owns just this one package. ok is false
// for methods with no safe single-package upgrade path — apt/rpm ship one-off files
// with no in-place upgrade, and the AUR helper command syncs the whole system — and
// for InstallScript/InstallUnknown. In particular, executing a remotely fetched
// install script would cross a trust boundary without authenticating its contents.
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
	cmd.Stdout = out
	cmd.Stderr = errOut
	return true, cmd.Run()
}

// HasGH reports whether the gh CLI is available on PATH.
func HasGH() bool {
	return cmdExists("gh")
}
