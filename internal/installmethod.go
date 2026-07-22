package internal

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// InstallMethod identifies how the running youtrack binary reached this machine,
// so "youtrack update" can point at the right upgrade command.
type InstallMethod int

const (
	InstallUnknown InstallMethod = iota
	InstallHomebrew
	InstallApt
	InstallRPM
	InstallArch
	InstallScoop
	InstallChocolatey
	InstallWinGet
	InstallNPM
	InstallGoInstall
	InstallScript
)

// DetectInstallMethod inspects the running executable's path (and, on Linux, the
// system package database) to guess which install method put it there.
func DetectInstallMethod() InstallMethod {
	exe, err := os.Executable()
	if err != nil {
		return InstallUnknown
	}
	return detectFromPath(exe)
}

func detectFromPath(exe string) InstallMethod {
	exeLower := strings.ToLower(exe)

	if strings.Contains(exeLower, "/cellar/") || strings.Contains(exeLower, "/homebrew/") {
		return InstallHomebrew
	}
	if strings.Contains(exeLower, `\scoop\`) {
		return InstallScoop
	}
	if strings.Contains(exeLower, `\chocolatey\`) {
		return InstallChocolatey
	}
	if strings.Contains(exeLower, `\windowsapps\`) || strings.Contains(exeLower, `\winget\`) {
		return InstallWinGet
	}
	if strings.Contains(exeLower, "node_modules") {
		return InstallNPM
	}
	if isGoBin(exeLower) {
		return InstallGoInstall
	}

	if runtime.GOOS == "linux" {
		if cmdExists("dpkg") && isInstalledVia(exe, "dpkg", "-S") {
			return InstallApt
		}
		if cmdExists("rpm") && isInstalledVia(exe, "rpm", "-qf") {
			return InstallRPM
		}
		if cmdExists("pacman") && isInstalledVia(exe, "pacman", "-Qo") {
			return InstallArch
		}
	}

	if exeLower == "/usr/local/bin/youtrack" || strings.HasSuffix(exeLower, "/.local/bin/youtrack") {
		return InstallScript
	}

	return InstallUnknown
}

func isGoBin(exe string) bool {
	if strings.Contains(exe, "/go/bin/") || strings.Contains(exe, `\go\bin\`) {
		return true
	}
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		return strings.HasPrefix(strings.ToLower(exe), strings.ToLower(gopath))
	}
	return false
}

func cmdExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func isInstalledVia(exe, cmd string, args ...string) bool {
	out, err := exec.Command(cmd, append(args, exe)...).CombinedOutput()
	if err != nil {
		return false
	}
	return len(out) > 0
}

// String names the install method for display (e.g. in verbose/debug output).
func (m InstallMethod) String() string {
	switch m {
	case InstallHomebrew:
		return "Homebrew"
	case InstallApt:
		return "apt/deb"
	case InstallRPM:
		return "rpm"
	case InstallArch:
		return "AUR"
	case InstallScoop:
		return "Scoop"
	case InstallChocolatey:
		return "Chocolatey"
	case InstallWinGet:
		return "WinGet"
	case InstallNPM:
		return "npm"
	case InstallGoInstall:
		return "go install"
	case InstallScript:
		return "install script"
	default:
		return "unknown"
	}
}

// UpgradeCommand returns the shell command a user should run to upgrade youtrack,
// matching the install method detected by DetectInstallMethod.
func (m InstallMethod) UpgradeCommand() string {
	const releasesURL = "https://github.com/JetBrains/youtrack-cli/releases/latest"
	switch m {
	case InstallHomebrew:
		return "brew upgrade youtrack"
	case InstallScoop:
		return "scoop update youtrack"
	case InstallChocolatey:
		return "choco upgrade YouTrackCLI"
	case InstallWinGet:
		return "winget upgrade JetBrains.YouTrackCLI"
	case InstallNPM:
		return "npm update -g @jetbrains/youtrack-cli"
	case InstallArch:
		return "yay -Syu youtrack-bin"
	case InstallApt:
		return "download the latest .deb from " + releasesURL + " and run: sudo dpkg -i youtrack_linux_<arch>.deb"
	case InstallRPM:
		return "sudo rpm -U " + releasesURL + "/download/youtrack_linux_<arch>.rpm"
	case InstallGoInstall:
		return "go install github.com/JetBrains/youtrack-cli@latest"
	case InstallScript:
		return "curl -fsSL " + releasesURL + "/download/install.sh | bash"
	default:
		return "visit " + releasesURL
	}
}
