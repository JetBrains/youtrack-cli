package internal

import (
	"bytes"
	"testing"
)

// TestRunAutoUpgrade_UnsupportedMethodNotAttempted stays a plain unit test: these
// methods never reach exec.Command (AutoUpgradeCommand returns ok=false), so there
// is no subprocess dispatch to exercise — just a pure decision. The Homebrew and
// go-install cases that spawn a process are covered end-to-end instead.
func TestRunAutoUpgrade_UnsupportedMethodNotAttempted(t *testing.T) {
	for _, method := range []InstallMethod{InstallApt, InstallScript} {
		var out, errOut bytes.Buffer
		attempted, err := RunAutoUpgrade(method, "1.2.3", &out, &errOut)
		if attempted {
			t.Errorf("RunAutoUpgrade(%s) attempted = true, want false", method)
		}
		if err != nil {
			t.Errorf("RunAutoUpgrade(%s) err = %v, want nil", method, err)
		}
	}
}
