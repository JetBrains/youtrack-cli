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
