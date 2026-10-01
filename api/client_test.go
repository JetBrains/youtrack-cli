package api

import (
	"net/http"
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
