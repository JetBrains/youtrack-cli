package config

import "testing"

func TestKeyringWriteFor(t *testing.T) {
	cases := []struct {
		name string
		inst Instance
		want keyringWrite
	}{
		{"no token writes nothing", Instance{URL: "https://a"}, keyringWriteNone},
		{"file token moves to the keyring", Instance{URL: "https://a", Token: "perm:x"}, keyringWriteToken},
		{"keyring token with its URL is not rebound", Instance{URL: "https://edited", Token: "perm:x", tokenInKeyring: true, tokenURL: "https://a"}, keyringWriteNone},
		{"keyring token from an older version gets its URL", Instance{URL: "https://a", Token: "perm:x", tokenInKeyring: true, tokenURLMissing: true}, keyringWriteURL},
		{"keyring token with an unreadable URL entry is not rebound", Instance{URL: "https://a", Token: "perm:x", tokenInKeyring: true}, keyringWriteNone},
		// AddInstance replaces the struct, so a refreshed or re-added token is stored again with the URL it was obtained for.
		{"replaced instance stores the new token", Instance{URL: "https://a", Token: "perm:new"}, keyringWriteToken},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := keyringWriteFor(&c.inst); got != c.want {
				t.Errorf("keyringWriteFor() = %d, want %d", got, c.want)
			}
		})
	}
}

func TestTokenURL(t *testing.T) {
	cases := []struct {
		name string
		inst Instance
		want string
	}{
		{"keyring URL wins over the config file", Instance{URL: "https://edited", tokenURL: "https://a"}, "https://a"},
		{"no keyring URL falls back to the config file", Instance{URL: "https://a"}, "https://a"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.inst.TokenURL(); got != c.want {
				t.Errorf("TokenURL() = %q, want %q", got, c.want)
			}
		})
	}
}
