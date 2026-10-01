package config

import "testing"

func TestKeyringSecretRoundTrip(t *testing.T) {
	tok, url := decodeKeyringSecret(encodeKeyringSecret("https://yt.example.com", "perm:abc.def"))
	if tok != "perm:abc.def" || url != "https://yt.example.com" {
		t.Errorf("round trip = (%q, %q)", tok, url)
	}
	// A bare token saved without a URL decodes with an empty URL.
	if tok, url := decodeKeyringSecret("perm:abc.def"); tok != "perm:abc.def" || url != "" {
		t.Errorf("bare token = (%q, %q)", tok, url)
	}
}
