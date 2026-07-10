package api

import (
	"testing"
)

func TestValidateOAuthEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hubURL   string
		endpoint string
		wantErr  bool
	}{
		{name: "same origin", hubURL: "https://hub.example.com/hub", endpoint: "https://hub.example.com/api/rest/oauth2/token"},
		{name: "same origin case insensitive", hubURL: "https://HUB.example.com", endpoint: "https://hub.EXAMPLE.com/oauth"},
		{name: "different host", hubURL: "https://hub.example.com", endpoint: "https://evil.example.com/oauth", wantErr: true},
		{name: "different scheme", hubURL: "https://hub.example.com", endpoint: "http://hub.example.com/oauth", wantErr: true},
		{name: "relative endpoint", hubURL: "https://hub.example.com", endpoint: "/oauth", wantErr: true},
		{name: "invalid hub", hubURL: "://bad", endpoint: "https://hub.example.com/oauth", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOAuthEndpoint(tt.hubURL, tt.endpoint, "token_endpoint")
			if tt.wantErr && err == nil {
				t.Fatal("expected an error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
