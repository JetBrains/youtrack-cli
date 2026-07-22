package api_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/creack/pty"
	"github.com/stretchr/testify/require"
)

// These tests drive the compiled binary through a real pty so DetectInstallMethod's
// TTY check (see cmd/update.go) sees a genuine terminal and actually attempts the
// auto-upgrade — the same command dispatch as a real interactive "youtrack update"
// run, just against a stub brew/go on PATH instead of the real package manager.

// copyBinaryToFakeInstallPath copies the already-built test binary into
// t.TempDir()/relDir/youtrack, so DetectInstallMethod's path-based detection sees
// it living somewhere a real install method would put it.
func copyBinaryToFakeInstallPath(t *testing.T, relDir string) string {
	t.Helper()
	data, err := os.ReadFile(binaryPath)
	require.NoError(t, err)

	dir := filepath.Join(t.TempDir(), relDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	fakeBinary := filepath.Join(dir, "youtrack")
	require.NoError(t, os.WriteFile(fakeBinary, data, 0o755))
	return fakeBinary
}

// writeE2EStubExecutable writes a shell script named name in dir that records its
// arguments to recordPath, standing in for a real package-manager binary on PATH.
func writeE2EStubExecutable(t *testing.T, dir, name, recordPath string) {
	t.Helper()
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\" > %q\n", recordPath)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
}

// runViaPTY runs binary with args attached to a pty as stdin/stdout/stderr (merged),
// so term.IsTerminal is true inside the child — exercising the code path that
// requires a real interactive terminal before attempting an auto-upgrade.
func runViaPTY(t *testing.T, binary string, extraEnv []string, args ...string) (output string, err error) {
	t.Helper()
	ptmx, tty, openErr := pty.Open()
	require.NoError(t, openErr)
	defer func() { _ = ptmx.Close() }()

	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty

	if startErr := cmd.Start(); startErr != nil {
		_ = tty.Close()
		return "", startErr
	}
	_ = tty.Close() // the child holds its own copy; the parent's is no longer needed

	var buf bytes.Buffer
	copyDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, ptmx)
		close(copyDone)
	}()

	waitErr := cmd.Wait()
	<-copyDone
	return buf.String(), waitErr
}

func TestCmd_Update_AutoUpgrade_Homebrew(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty-based TTY simulation is POSIX-only")
	}
	fakeBinary := copyBinaryToFakeInstallPath(t, "Cellar/youtrack/1.0.0/bin")

	stubDir := t.TempDir()
	record := filepath.Join(stubDir, "record.txt")
	writeE2EStubExecutable(t, stubDir, "brew", record)

	output, err := runViaPTY(t, fakeBinary,
		[]string{"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH")},
		"update", "--format", "table")
	skipIfUpdateCheckUnreachable(t, output, output, err)
	require.NoError(t, err, "output: %s", output)

	got, readErr := os.ReadFile(record)
	require.NoError(t, readErr, "stub brew was never invoked; output: %s", output)
	require.Equal(t, "upgrade youtrack\n", string(got))
}

func TestCmd_Update_AutoUpgrade_GoInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty-based TTY simulation is POSIX-only")
	}
	fakeBinary := copyBinaryToFakeInstallPath(t, "go/bin")

	stubDir := t.TempDir()
	record := filepath.Join(stubDir, "record.txt")
	writeE2EStubExecutable(t, stubDir, "go", record)

	output, err := runViaPTY(t, fakeBinary,
		[]string{"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH")},
		"update", "--format", "table")
	skipIfUpdateCheckUnreachable(t, output, output, err)
	require.NoError(t, err, "output: %s", output)

	got, readErr := os.ReadFile(record)
	require.NoError(t, readErr, "stub go was never invoked; output: %s", output)
	require.Equal(t, "install github.com/JetBrains/youtrack-cli@latest\n", string(got))
}
