package internal

import "testing"

func TestDetectFromPath(t *testing.T) {
	cases := []struct {
		name string
		exe  string
		want InstallMethod
	}{
		{"homebrew cellar", "/usr/local/Cellar/youtrack/1.0.0/bin/youtrack", InstallHomebrew},
		{"homebrew opt dir", "/opt/homebrew/bin/youtrack", InstallHomebrew},
		{"scoop shim", `C:\Users\alice\scoop\shims\youtrack.exe`, InstallScoop},
		{"chocolatey bin", `C:\ProgramData\chocolatey\bin\youtrack.exe`, InstallChocolatey},
		{"winget links", `C:\Users\alice\AppData\Local\Microsoft\WindowsApps\youtrack.exe`, InstallWinGet},
		{"npm global", "/usr/local/lib/node_modules/@jetbrains/youtrack-cli/bin/youtrack", InstallNPM},
		{"install script default dir", "/usr/local/bin/youtrack", InstallScript},
		{"install script user dir", "/home/alice/.local/bin/youtrack", InstallScript},
		{"unrecognized path", "/opt/random/youtrack", InstallUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := detectFromPathContext(t.Context(), c.exe); got != c.want {
				t.Errorf("detectFromPath(%q) = %v, want %v", c.exe, got, c.want)
			}
		})
	}
}

func TestAutoUpgradeCommand(t *testing.T) {
	cases := []struct {
		name   string
		method InstallMethod
		ok     bool
	}{
		{"homebrew is scoped to this package", InstallHomebrew, true},
		{"scoop is scoped to this package", InstallScoop, true},
		{"chocolatey is scoped to this package", InstallChocolatey, true},
		{"winget is scoped to this package", InstallWinGet, true},
		{"npm is scoped to this package", InstallNPM, true},
		{"go install is scoped to this package", InstallGoInstall, true},
		{"apt has no in-place upgrade", InstallApt, false},
		{"rpm has no in-place upgrade", InstallRPM, false},
		{"AUR helper would sync the whole system", InstallArch, false},
		{"script install re-runs the installer", InstallScript, true},
		{"unknown install has no upgrade path", InstallUnknown, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			name, args, ok := c.method.AutoUpgradeCommand()
			if ok != c.ok {
				t.Fatalf("AutoUpgradeCommand() ok = %v, want %v", ok, c.ok)
			}
			if ok && name == "" {
				t.Errorf("AutoUpgradeCommand() returned ok=true with an empty command name")
			}
			if !ok && (name != "" || args != nil) {
				t.Errorf("AutoUpgradeCommand() returned ok=false but non-empty command: %q %v", name, args)
			}
		})
	}
}

func TestIsGoBin(t *testing.T) {
	cases := []struct {
		name string
		exe  string
		want bool
	}{
		{"go bin unix", "/home/alice/go/bin/youtrack", true},
		{"go bin windows", `C:\Users\alice\go\bin\youtrack.exe`, true},
		{"not a go bin", "/usr/local/bin/youtrack", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isGoBin(c.exe); got != c.want {
				t.Errorf("isGoBin(%q) = %v, want %v", c.exe, got, c.want)
			}
		})
	}
}

func TestInstallerEnvDropsStaleInstallSettings(t *testing.T) {
	t.Setenv("YOUTRACK_INSTALL_RELEASE", "v0.5.1")
	t.Setenv("YOUTRACK_INSTALL_DIR", "/somewhere/else")
	t.Setenv("YOUTRACK_INSTALL_UPDATE", "")
	t.Setenv("YOUTRACK_UNRELATED", "kept")

	env := map[string]int{}
	for _, kv := range installerEnv() {
		env[kv]++
	}
	for _, stale := range []string{"YOUTRACK_INSTALL_RELEASE=v0.5.1", "YOUTRACK_INSTALL_DIR=/somewhere/else", "YOUTRACK_INSTALL_UPDATE="} {
		if env[stale] != 0 {
			t.Errorf("installerEnv() kept %q", stale)
		}
	}
	if env["YOUTRACK_INSTALL_UPDATE=1"] != 1 {
		t.Errorf("installerEnv() must set YOUTRACK_INSTALL_UPDATE=1 exactly once")
	}
	if env["YOUTRACK_UNRELATED=kept"] != 1 {
		t.Errorf("installerEnv() dropped an unrelated variable")
	}
}

func TestShellQuote(t *testing.T) {
	for in, want := range map[string]string{
		"/home/alice/.local/bin": `'/home/alice/.local/bin'`,
		"/opt/my tools":          `'/opt/my tools'`,
		"/tmp/it's":              `'/tmp/it'\''s'`,
		"/tmp/$(rm -rf ~)":       `'/tmp/$(rm -rf ~)'`,
	} {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}
