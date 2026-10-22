package internal

import "testing"

func TestPendingUpdate(t *testing.T) {
	tests := []struct {
		name                       string
		current, latest, dismissed string
		want                       bool
	}{
		{"newer release", "0.5.0", "0.6.0", "", true},
		{"v prefix on current", "v0.5.0", "0.6.0", "", true},
		{"same version", "0.6.0", "0.6.0", "", false},
		{"older cached release", "0.6.0", "0.5.0", "", false},
		{"no cached release", "0.5.0", "", "", false},
		{"dismissed release", "0.5.0", "0.6.0", "0.6.0", false},
		{"newer than dismissed", "0.5.0", "0.7.0", "0.6.0", true},
		{"dev build", "dev", "0.6.0", "", false},
		{"unknown build", "unknown", "0.6.0", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PendingUpdate(tt.current, tt.latest, tt.dismissed); got != tt.want {
				t.Errorf("PendingUpdate(%q, %q, %q) = %v, want %v", tt.current, tt.latest, tt.dismissed, got, tt.want)
			}
		})
	}
}
