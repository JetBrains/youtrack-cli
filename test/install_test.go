package api_test

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// archName maps Go's runtime arch names to the archive naming used by GoReleaser.
func archName() string {
	if runtime.GOARCH == "amd64" {
		return "x86_64"
	}
	return runtime.GOARCH
}

// buildFakeArchive creates a .tar.gz containing the already-built CLI binary
// alongside LICENSE and README.md, mimicking a real GoReleaser release archive.
func buildFakeArchive(t *testing.T, version string) string {
	t.Helper()

	archivePath := filepath.Join(t.TempDir(), fmt.Sprintf("youtrack_%s_%s_%s.tar.gz", version, runtime.GOOS, archName()))
	f, err := os.Create(archivePath)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	addFile := func(name, srcPath string) {
		data, err := os.ReadFile(srcPath)
		require.NoError(t, err, "reading %s", srcPath)
		hdr := &tar.Header{
			Name: name,
			Mode: 0755,
			Size: int64(len(data)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err = tw.Write(data)
		require.NoError(t, err)
	}

	addFile("youtrack", binaryPath)

	// LICENSE and README.md are optional in the archive; add stubs so the script
	// finds the binary regardless of whether the real files are present.
	repoRoot := filepath.Join(filepath.Dir(binaryPath), "..", "..")
	for _, name := range []string{"LICENSE", "README.md"} {
		src := filepath.Join(repoRoot, name)
		if _, err := os.Stat(src); err == nil {
			addFile(name, src)
		}
	}

	return archivePath
}

// TestInstallScript verifies that install.sh correctly downloads, extracts, and
// installs the youtrack binary. It uses a local httptest server to serve a fake
// release archive so no real GitHub release is required.
//
// The test patches install.sh's download URL via an env override rather than
// modifying the production script.
func TestInstallScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is not supported on Windows")
	}

	const version = "0.1.0-test"

	archivePath := buildFakeArchive(t, version)
	archiveData, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	archiveChecksum := fmt.Sprintf("%x  %s\n", sha256.Sum256(archiveData), filepath.Base(archivePath))
	servedChecksum := archiveChecksum
	gnupgHome := t.TempDir()
	gpgEnv := append(os.Environ(), "GNUPGHOME="+gnupgHome)
	generateKey := exec.Command("gpg", "--batch", "--passphrase", "", "--quick-generate-key", "Installer Test <installer@example.invalid>", "ed25519", "sign", "0")
	generateKey.Env = gpgEnv
	require.NoError(t, generateKey.Run())
	showFingerprint := exec.Command("gpg", "--batch", "--with-colons", "--fingerprint")
	showFingerprint.Env = gpgEnv
	fingerprintOutput, err := showFingerprint.Output()
	require.NoError(t, err)
	var fingerprint string
	for _, line := range strings.Split(string(fingerprintOutput), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) > 9 && fields[0] == "fpr" {
			fingerprint = fields[9]
			break
		}
	}
	require.NotEmpty(t, fingerprint)

	checksumsPath := filepath.Join(t.TempDir(), "checksums.txt")
	require.NoError(t, os.WriteFile(checksumsPath, []byte(archiveChecksum), 0600))
	signChecksums := exec.Command("gpg", "--batch", "--armor", "--detach-sign", checksumsPath)
	signChecksums.Env = gpgEnv
	require.NoError(t, signChecksums.Run())
	signatureData, err := os.ReadFile(checksumsPath + ".asc")
	require.NoError(t, err)
	// Serve the archive at the exact path install.sh constructs.
	expectedPath := fmt.Sprintf("/JetBrains/youtrack-cli/releases/download/v%s/youtrack_%s_%s_%s.tar.gz",
		version, version, runtime.GOOS, archName())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case expectedPath:
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(archiveData)
		case fmt.Sprintf("/JetBrains/youtrack-cli/releases/download/v%s/checksums.txt", version):
			_, _ = w.Write([]byte(servedChecksum))
		case fmt.Sprintf("/JetBrains/youtrack-cli/releases/download/v%s/checksums.txt.asc", version):
			_, _ = w.Write(signatureData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Create a patched copy of install.sh that uses the test server instead of GitHub.
	scriptSrc, err := os.ReadFile(filepath.Join("..", "install.sh"))
	require.NoError(t, err)

	// Replace the hardcoded GitHub base URL with the test server.
	patched := strings.ReplaceAll(
		string(scriptSrc),
		`"https://github.com/${REPO}/releases/download/${RELEASE}"`,
		`"`+srv.URL+`/JetBrains/youtrack-cli/releases/download/${RELEASE}"`,
	)
	require.NotEqual(t, string(scriptSrc), patched, "URL replacement in install.sh did not match — check the script's download URL pattern")

	// Disable the `gh` CLI branches so the patched curl URL is always used.
	// gh release download would attempt to hit the real GitHub repo, bypassing our mock server.
	patched = strings.ReplaceAll(patched,
		"if command -v gh >/dev/null 2>&1; then",
		"if false; then",
	)
	patched = strings.ReplaceAll(patched,
		`RELEASE_FINGERPRINT="B46DC71E03FEEB7F89D1F2491F7A8F87B9D8F501"`,
		`RELEASE_FINGERPRINT="`+fingerprint+`"`,
	)
	patched = strings.ReplaceAll(patched,
		"if command -v gh >/dev/null 2>&1 &&",
		"if false &&",
	)

	patchedScript := filepath.Join(t.TempDir(), "install.sh")
	require.NoError(t, os.WriteFile(patchedScript, []byte(patched), 0755))

	installDir := t.TempDir()

	cmd := exec.Command("bash", patchedScript, "v"+version, installDir)
	cmd.Env = gpgEnv
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "install.sh failed:\n%s", string(out))

	// Verify the binary was installed.
	installedBin := filepath.Join(installDir, "youtrack")
	info, err := os.Stat(installedBin)
	require.NoError(t, err, "binary not found at %s", installedBin)
	require.True(t, info.Mode()&0111 != 0, "installed binary is not executable")

	// Verify the installed binary runs correctly.
	versionOut, err := exec.Command(installedBin, "--version").Output()
	require.NoError(t, err, "--version failed on installed binary")
	require.Contains(t, string(versionOut), "youtrack ", "unexpected --version output: %s", string(versionOut))

	// Replacing both a release archive checksum entry and its manifest is not
	// enough: the manifest must still carry a valid signature from the pinned key.
	servedChecksum = fmt.Sprintf("%064x  %s\n", 0, filepath.Base(archivePath))
	tamperedInstallDir := t.TempDir()
	tamperedCmd := exec.Command("bash", patchedScript, "v"+version, tamperedInstallDir)
	tamperedCmd.Env = gpgEnv
	tamperedOut, err := tamperedCmd.CombinedOutput()
	require.Error(t, err, "installer accepted a tampered checksum manifest:\n%s", string(tamperedOut))
	require.Contains(t, string(tamperedOut), "invalid checksums signature")
	_, err = os.Stat(filepath.Join(tamperedInstallDir, "youtrack"))
	require.ErrorIs(t, err, os.ErrNotExist, "installer wrote a binary after signature verification failed")
}
