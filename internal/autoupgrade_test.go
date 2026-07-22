package internal

import (
	"bytes"
	"testing"
)

// TestRunAutoUpgrade_UnsupportedMethodNotAttempted is the one case that stays a
// plain unit test: InstallApt never reaches exec.Command at all (AutoUpgradeCommand
// returns ok=false), so there's no subprocess dispatch to exercise — just a pure
// decision. The Homebrew/go-install cases that actually spawn a process are
// covered end-to-end in test/update_test.go instead.
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
