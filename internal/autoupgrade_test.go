package internal

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeStubExecutable creates a shell script named name in dir that writes its
// arguments to recordPath, standing in for a real package-manager binary. This
// exercises RunAutoUpgrade's actual exec.Command dispatch — no Go-level mocking,
// no network — by letting it genuinely find and run the stub via PATH, instead
// of the real brew/npm/go.
func writeStubExecutable(t *testing.T, dir, name, recordPath string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("stub executables here are POSIX shell scripts")
	}
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\" > %q\n", recordPath)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatalf("writing stub %s: %v", name, err)
	}
}

func TestRunAutoUpgrade_Homebrew(t *testing.T) {
	dir := t.TempDir()
	record := filepath.Join(dir, "record.txt")
	writeStubExecutable(t, dir, "brew", record)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var out, errOut bytes.Buffer
	attempted, err := RunAutoUpgrade(InstallHomebrew, &out, &errOut)
	if !attempted {
		t.Fatal("RunAutoUpgrade(InstallHomebrew) attempted = false, want true")
	}
	if err != nil {
		t.Fatalf("RunAutoUpgrade failed: %v (stderr: %s)", err, errOut.String())
	}

	got, err := os.ReadFile(record)
	if err != nil {
		t.Fatalf("stub brew was never invoked: %v", err)
	}
	if want := "upgrade youtrack\n"; string(got) != want {
		t.Errorf("brew invoked with %q, want %q", got, want)
	}
}

func TestRunAutoUpgrade_GoInstall(t *testing.T) {
	dir := t.TempDir()
	record := filepath.Join(dir, "record.txt")
	writeStubExecutable(t, dir, "go", record)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var out, errOut bytes.Buffer
	attempted, err := RunAutoUpgrade(InstallGoInstall, &out, &errOut)
	if !attempted {
		t.Fatal("RunAutoUpgrade(InstallGoInstall) attempted = false, want true")
	}
	if err != nil {
		t.Fatalf("RunAutoUpgrade failed: %v (stderr: %s)", err, errOut.String())
	}

	got, err := os.ReadFile(record)
	if err != nil {
		t.Fatalf("stub go was never invoked: %v", err)
	}
	if want := "install github.com/JetBrains/youtrack-cli@latest\n"; string(got) != want {
		t.Errorf("go invoked with %q, want %q", got, want)
	}
}

func TestRunAutoUpgrade_UnsupportedMethodNotAttempted(t *testing.T) {
	var out, errOut bytes.Buffer
	attempted, err := RunAutoUpgrade(InstallApt, &out, &errOut)
	if attempted {
		t.Fatal("RunAutoUpgrade(InstallApt) attempted = true, want false")
	}
	if err != nil {
		t.Errorf("RunAutoUpgrade(InstallApt) err = %v, want nil", err)
	}
}
