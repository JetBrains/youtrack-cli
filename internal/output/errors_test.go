package output

import "testing"

func TestIsBinaryContentType(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		want        bool
	}{
		{"image/png", true},
		{"application/octet-stream", true},
		{"application/json;charset=UTF-8", false},
		{"text/html; charset=utf-8", false},
		{"", false},
	} {
		if got := isBinaryContentType(tc.contentType); got != tc.want {
			t.Errorf("isBinaryContentType(%q) = %v, want %v", tc.contentType, got, tc.want)
		}
	}
}
