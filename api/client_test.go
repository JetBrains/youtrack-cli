package api

import "testing"

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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := mergeFieldsParam(c.path, c.fields); got != c.out {
				t.Errorf("mergeFieldsParam(%q, %q) = %q, want %q", c.path, c.fields, got, c.out)
			}
		})
	}
}
