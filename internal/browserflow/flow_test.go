package browserflow

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestGenerateState(t *testing.T) {
	a, err := GenerateState()
	if err != nil {
		t.Fatal(err)
	}
	b, err := GenerateState()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 22 { // 16 random bytes, unpadded base64url
		t.Fatalf("state length = %d, want 22", len(a))
	}
	if a == b {
		t.Fatal("two states are equal")
	}
	for _, r := range a {
		if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_", r) {
			t.Fatalf("state %q has a character outside the URL-safe alphabet", a)
		}
	}
}

func TestValidateCallback(t *testing.T) {
	const state = "expected-state"
	for _, tc := range []struct {
		name     string
		query    string
		wantCode string
		wantErr  string
	}{
		{name: "valid", query: "code=abc&state=expected-state", wantCode: "abc"},
		{name: "unknown params ignored", query: "code=abc&state=expected-state&iss=x", wantCode: "abc"},
		{name: "wrong state", query: "code=abc&state=other", wantErr: "state mismatch"},
		{name: "missing state", query: "code=abc", wantErr: "state mismatch"},
		{name: "state prefix", query: "code=abc&state=expected-stat", wantErr: "state mismatch"},
		{name: "error needs state too", query: "error=access_denied&state=other", wantErr: "state mismatch"},
		{name: "access denied", query: "error=access_denied&state=expected-state", wantErr: "access_denied"},
		{name: "no code", query: "state=expected-state", wantErr: "without code"},
		{name: "duplicate code", query: "code=a&code=b&state=expected-state", wantErr: "repeats"},
		{name: "duplicate state", query: "code=a&state=expected-state&state=expected-state", wantErr: "repeats"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, err := url.ParseQuery(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			code, err := validateCallback(q, state)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				if code != "" {
					t.Fatalf("code %q returned alongside an error", code)
				}
				return
			}
			if err != nil || code != tc.wantCode {
				t.Fatalf("got (%q, %v), want %q", code, err, tc.wantCode)
			}
		})
	}
}

func TestParseRedirectURL(t *testing.T) {
	const redirect = "http://127.0.0.1:43210"
	const state = "st"
	for _, tc := range []struct {
		name     string
		pasted   string
		wantCode string
		wantErr  string
	}{
		{name: "as Hub sends it", pasted: "http://127.0.0.1:43210?code=c1&state=st", wantCode: "c1"},
		{name: "with slash and whitespace", pasted: "  http://127.0.0.1:43210/?code=c1&state=st\r\n", wantCode: "c1"},
		{name: "bare code refused", pasted: "c1", wantErr: "full URL"},
		{name: "other port", pasted: "http://127.0.0.1:1234/?code=c1&state=st", wantErr: "does not point to"},
		{name: "other host", pasted: "http://evil.example:43210/?code=c1&state=st", wantErr: "does not point to"},
		{name: "other path", pasted: "http://127.0.0.1:43210/x?code=c1&state=st", wantErr: "does not point to"},
		{name: "https scheme", pasted: "https://127.0.0.1:43210/?code=c1&state=st", wantErr: "does not point to"},
		{name: "state mismatch", pasted: "http://127.0.0.1:43210/?code=c1&state=zz", wantErr: "state mismatch"},
		{name: "denied", pasted: "http://127.0.0.1:43210/?error=access_denied&state=st", wantErr: "access_denied"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, err := ParseRedirectURL(tc.pasted, redirect, state)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || code != tc.wantCode {
				t.Fatalf("got (%q, %v), want %q", code, err, tc.wantCode)
			}
		})
	}

	_, err := ParseRedirectURL("http://127.0.0.1:43210/?code=c1&state=zz", redirect, state)
	if !errors.Is(err, errPastedStateMismatch) {
		t.Fatalf("state mismatch must abort the flow, got %v", err)
	}
}

func TestSanitizeErrorCode(t *testing.T) {
	if got := sanitizeErrorCode("access_denied\x1b]8;;http://evil\x07"); strings.ContainsAny(got, "\x1b\x07") {
		t.Fatalf("control characters survived: %q", got)
	}
	if got := sanitizeErrorCode(strings.Repeat("a", 500)); len(got) > 64 {
		t.Fatalf("error code not truncated: %d chars", len(got))
	}
}
