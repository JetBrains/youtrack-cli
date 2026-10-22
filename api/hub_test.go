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

func TestResolveHubURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		ref     string
		want    string
	}{
		{name: "root-relative built-in hub", baseURL: "http://127.0.0.1:8080", ref: "/hub", want: "http://127.0.0.1:8080/hub"},
		{name: "context path included", baseURL: "https://host/youtrack", ref: "/youtrack/hub", want: "https://host/youtrack/hub"},
		{name: "trailing slash on base", baseURL: "https://host/youtrack/", ref: "/youtrack/hub", want: "https://host/youtrack/hub"},
		{name: "absolute hub kept", baseURL: "https://youtrack.jetbrains.com", ref: "https://hub.jetbrains.com", want: "https://hub.jetbrains.com"},
		{name: "empty", baseURL: "https://host", ref: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveHubURL(tt.baseURL, tt.ref); got != tt.want {
				t.Fatalf("ResolveHubURL(%q, %q) = %q, want %q", tt.baseURL, tt.ref, got, tt.want)
			}
		})
	}
}
