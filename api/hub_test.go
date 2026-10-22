package api

import "testing"

func TestValidateHubURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		hubURL  string
		wantErr bool
	}{
		{name: "https hub", baseURL: "https://work.youtrack.cloud", hubURL: "https://hub.example.com/hub"},
		{name: "https hub for http instance", baseURL: "http://localhost:8080", hubURL: "https://hub.example.com"},
		{name: "http hub for http instance", baseURL: "http://localhost:8080", hubURL: "http://localhost:8080/hub"},
		{name: "scheme case-insensitive", baseURL: "https://work.youtrack.cloud", hubURL: "HTTPS://hub.example.com"},
		{name: "http hub downgrades https instance", baseURL: "https://work.youtrack.cloud", hubURL: "http://hub.example.com", wantErr: true},
		{name: "javascript scheme", baseURL: "https://work.youtrack.cloud", hubURL: "javascript:alert(1)", wantErr: true},
		{name: "file scheme", baseURL: "https://work.youtrack.cloud", hubURL: "file:///etc/passwd", wantErr: true},
		{name: "custom handler scheme", baseURL: "https://work.youtrack.cloud", hubURL: "ms-settings://hub.example.com", wantErr: true},
		{name: "userinfo spoofing", baseURL: "https://work.youtrack.cloud", hubURL: "https://work.youtrack.cloud@evil.example.com", wantErr: true},
		{name: "relative url", baseURL: "https://work.youtrack.cloud", hubURL: "/hub", wantErr: true},
		{name: "control characters", baseURL: "https://work.youtrack.cloud", hubURL: "https://hub.example.com/\x1b]8;;evil\x1b\\", wantErr: true},
		{name: "empty", baseURL: "https://work.youtrack.cloud", hubURL: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHubURL(tt.baseURL, tt.hubURL)
			if tt.wantErr && err == nil {
				t.Fatal("expected an error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsInsecureRemoteHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want bool
	}{
		{"https://work.youtrack.cloud", false},
		{"http://localhost:8080", false},
		{"http://LOCALHOST", false},
		{"http://127.0.0.1:8080", false},
		{"http://127.1.2.3", false},
		{"http://[::1]:8080", false},
		{"http://youtrack.intranet", true},
		{"HTTP://youtrack.intranet", true},
		{"http://10.0.0.5:8080", true},
		{"http://localhost.attacker.net", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := IsInsecureRemoteHTTP(tt.url); got != tt.want {
				t.Errorf("IsInsecureRemoteHTTP(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestParsePermanentToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		token     string
		wantLogin string
		wantID    string
		wantOK    bool
	}{
		{name: "current format", token: "perm-dmlrdG9yLnRpdWxwaW4=.OTItMjE2OTg=.secret", wantLogin: "viktor.tiulpin", wantID: "92-21698", wantOK: true},
		{name: "legacy prefix", token: "perm:YWRtaW4=.MS0y.secret", wantLogin: "admin", wantID: "1-2", wantOK: true},
		{name: "oauth access token", token: "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ4In0.sig"},
		{name: "missing secret", token: "perm-YWRtaW4=.MS0y."},
		{name: "two segments", token: "perm-YWRtaW4=.MS0y"},
		{name: "invalid base64", token: "perm-***.MS0y.secret"},
		{name: "empty", token: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			login, id, ok := ParsePermanentToken(tt.token)
			if ok != tt.wantOK || login != tt.wantLogin || id != tt.wantID {
				t.Fatalf("ParsePermanentToken(%q) = %q, %q, %v; want %q, %q, %v", tt.token, login, id, ok, tt.wantLogin, tt.wantID, tt.wantOK)
			}
		})
	}
}
