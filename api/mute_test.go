package api

import "testing"

func TestWithMuteParam(t *testing.T) {
	cases := []struct{ path, want string }{
		{"/api/issues/FOO-1", "/api/issues/FOO-1?muteUpdateNotifications=true"},
		{"/api/issues/FOO-1?fields=id", "/api/issues/FOO-1?fields=id&muteUpdateNotifications=true"},
		{"/api/issues/FOO-1?fields=id&muteUpdateNotifications=true", "/api/issues/FOO-1?fields=id&muteUpdateNotifications=true"},
		{"/api/issues/FOO-1?muteUpdateNotifications=false", "/api/issues/FOO-1?muteUpdateNotifications=true"},
		{"/api/issues/FOO-1?%6duteUpdateNotifications=false&muteUpdateNotifications=false", "/api/issues/FOO-1?muteUpdateNotifications=true"},
	}
	for _, c := range cases {
		if got := withMuteParam(c.path); got != c.want {
			t.Errorf("withMuteParam(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}
