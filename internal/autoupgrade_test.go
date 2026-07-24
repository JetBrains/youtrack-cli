package internal

import (
	"bytes"
	"testing"
)

func TestReleaseTag(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		wantErr     bool
	}{
		{input: "1.2.3", want: "v1.2.3"},
		{input: "v1.2.3-eap.1", want: "v1.2.3-eap.1"},
		{input: "--repo", wantErr: true},
		{input: "1.2", wantErr: true},
	} {
		got, err := releaseTag(tc.input)
		if (err != nil) != tc.wantErr {
			t.Fatalf("releaseTag(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
		}
		if got != tc.want {
			t.Errorf("releaseTag(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestRunAutoUpgrade_UnsupportedMethodNotAttempted is the one case that stays a
// plain unit test: InstallApt never reaches exec.Command at all (AutoUpgradeCommand
// returns ok=false), so there's no subprocess dispatch to exercise — just a pure
// decision. The Homebrew/go-install cases that actually spawn a process are
// covered end-to-end in test/update_test.go instead.
func TestRunAutoUpgrade_UnsupportedMethodNotAttempted(t *testing.T) {
	var out, errOut bytes.Buffer
	attempted, err := RunAutoUpgrade(InstallApt, "1.2.3", &out, &errOut)
	if attempted {
		t.Fatal("RunAutoUpgrade(InstallApt) attempted = true, want false")
	}
	if err != nil {
		t.Errorf("RunAutoUpgrade(InstallApt) err = %v, want nil", err)
	}
}
