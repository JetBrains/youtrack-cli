package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestMergeFieldsParam(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		fields string
		out    string
	}{
		{"no existing query adds fields", "/api/issues/FOO-1", "id,summary", "/api/issues/FOO-1?fields=id,summary"},
		{"existing unrelated query appends fields", "/api/issues/FOO-1?muteUpdateNotifications=true", "id,summary", "/api/issues/FOO-1?muteUpdateNotifications=true&fields=id,summary"},
		{"existing fields param is extended", "/api/issues/FOO-1?fields=id", "summary", "/api/issues/FOO-1?fields=id,summary"},
		{"fields param followed by another param is extended in place", "/api/issues/FOO-1?fields=id&muteUpdateNotifications=true", "summary", "/api/issues/FOO-1?fields=id,summary&muteUpdateNotifications=true"},
		{"fields param preceded by another param is extended", "/api/issues/FOO-1?muteUpdateNotifications=true&fields=id", "summary", "/api/issues/FOO-1?muteUpdateNotifications=true&fields=id,summary"},
		{"param ending in fields= is not mistaken for fields", "/api/issues?customFields=Priority", "id", "/api/issues?customFields=Priority&fields=id"},
		{"param ending in fields= before the real one is skipped", "/api/issues?customFields=Priority&fields=id", "summary", "/api/issues?customFields=Priority&fields=id,summary"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := mergeFieldsParam(c.path, c.fields); got != c.out {
				t.Errorf("mergeFieldsParam(%q, %q) = %q, want %q", c.path, c.fields, got, c.out)
			}
		})
	}
}

func TestValidRequestPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/api/issues", want: true},
		{path: "/api/issues?fields=id", want: true},
		{path: "https://attacker.example/steal", want: false},
		{path: "//attacker.example/steal", want: false},
		{path: "@attacker.example/steal", want: false},
		{path: "api/issues", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := validRequestPath(tt.path); got != tt.want {
				t.Errorf("validRequestPath(%q) = %t, want %t", tt.path, got, tt.want)
			}
		})
	}
}

func TestHeaderBreakEscaper(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain name is unchanged", "report.pdf", "report.pdf"},
		{"quote is left for the server to read", `say "hi".txt`, `say "hi".txt`},
		{"CR and LF cannot start a new header", "a.txt\r\nContent-Type: text/html", "a.txt%0D%0AContent-Type: text/html"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := headerBreakEscaper.Replace(tt.in); got != tt.want {
				t.Errorf("headerBreakEscaper.Replace(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSameOriginRedirectAuthorization(t *testing.T) {
	cases := []struct {
		name, first, target string
		keep                bool
	}{
		{"same origin keeps header", "https://yt.example.com/api/me", "https://yt.example.com/api/x", true},
		{"host case is ignored", "https://yt.example.com/api/me", "https://YT.example.com/api/x", true},
		{"explicit default port keeps header", "https://yt.example.com/api/me", "https://yt.example.com:443/api/x", true},
		{"dropped default port keeps header", "https://yt.example.com:443/api/me", "https://yt.example.com/api/x", true},
		{"http downgrade drops header", "https://yt.example.com/api/me", "http://yt.example.com/api/x", false},
		{"subdomain drops header", "https://yt.example.com/api/me", "https://evil.yt.example.com/api/x", false},
		{"other port drops header", "https://yt.example.com/api/me", "https://yt.example.com:8443/api/x", false},
		{"https upgrade on same host keeps header", "http://yt.example.com/api/me", "https://yt.example.com/api/x", true},
		{"https upgrade to other port on same host keeps header", "http://yt.example.com:8080/api/me", "https://yt.example.com:8443/api/x", true},
		{"https upgrade to other host drops header", "http://yt.example.com/api/me", "https://other.example.com/api/x", false},
		{"http to other port drops header", "http://yt.example.com/api/me", "http://yt.example.com:8080/api/x", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			first, _ := http.NewRequest(http.MethodGet, c.first, nil)
			next, _ := http.NewRequest(http.MethodGet, c.target, nil)
			next.Header.Set("Authorization", "Bearer secret")
			if err := sameOriginRedirect(next, []*http.Request{first}); err != nil {
				t.Fatal(err)
			}
			if got := next.Header.Get("Authorization") != ""; got != c.keep {
				t.Errorf("Authorization kept = %v, want %v", got, c.keep)
			}
		})
	}
}

func TestRedactSignature(t *testing.T) {
	cases := []struct{ in, out string }{
		{"/api/files/1-2/sign=secret", "/api/files/1-2/sign=REDACTED"},
		{"/api/files/1-2/%73ign%3Dsecret", "/api/files/1-2/sign=REDACTED"},
		{"/api/files/1-2?%73ign=secret", "/api/files/1-2?sign=REDACTED"},
		{"/api/files/1-2?sign=abc&updated=1", "/api/files/1-2?sign=REDACTED&updated=1"},
		{"http://h/api/files/1-2?updated=1&sign=abc", "http://h/api/files/1-2?updated=1&sign=REDACTED"},
		{"/api/issues?query=design", "/api/issues?query=design"},
		{"/api/issues/FOO-1", "/api/issues/FOO-1"},
	}
	for _, c := range cases {
		if got := redactSignature(c.in); got != c.out {
			t.Errorf("redactSignature(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

func TestInstancePath(t *testing.T) {
	cases := []struct {
		name, baseURL, ref, out string
		wantErr                 bool
	}{
		{"root-relative on a root instance", "https://yt.example.com", "/api/files/1-2?sign=s", "/api/files/1-2?sign=s", false},
		{"root-relative with context path", "https://example.com/youtrack", "/youtrack/api/files/1-2?sign=s", "/api/files/1-2?sign=s", false},
		{"api path without context path", "https://example.com/youtrack", "/api/files/1-2?sign=s", "/api/files/1-2?sign=s", false},
		{"absolute on the instance", "https://yt.example.com", "https://yt.example.com/api/files/1-2?sign=s", "/api/files/1-2?sign=s", false},
		{"absolute on another host", "https://yt.example.com", "https://evil.example.com/api/files/1-2?sign=s", "", true},
		{"traversal fallback", "https://example.com/youtrack", "/api/../../admin", "", true},
		{"escaped traversal", "https://example.com/youtrack", "/api/%2e%2e/%2e%2e/admin", "", true},
		{"escaped slash traversal", "https://example.com/youtrack", "/api/%2e%2e%2f%2e%2e/admin", "", true},
		{"context traversal", "https://example.com/youtrack", "/youtrack/api/../../admin", "", true},
		{"other scheme", "https://example.com/youtrack", "http://example.com/api/files/1?sign=s", "", true},
		{"other port", "https://example.com/youtrack", "https://example.com:444/api/files/1?sign=s", "", true},
		{"default port", "https://example.com:443/youtrack", "https://example.com/youtrack/api/files/1?sign=s", "/api/files/1?sign=s", false},
		{"host case", "https://EXAMPLE.com/youtrack", "https://example.COM/youtrack/api/files/1?sign=s", "/api/files/1?sign=s", false},
		{"user info", "https://example.com/youtrack", "https://user@example.com/api/files/1?sign=s", "", true},
		{"outside the context path", "https://example.com/youtrack", "/other/files/1-2", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NewClient(c.baseURL, "").instancePath(c.ref)
			if c.wantErr {
				if err == nil {
					t.Fatalf("instancePath(%q) = %q, want error", c.ref, got)
				}
				if strings.Contains(err.Error(), "sign=") {
					t.Errorf("error leaks the signed URL: %v", err)
				}
				return
			}
			if err != nil || got != c.out {
				t.Errorf("instancePath(%q) = %q, %v; want %q", c.ref, got, err, c.out)
			}
		})
	}
}
