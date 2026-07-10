package api

import "testing"

func TestWrapUserValues(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		{"plain login is untouched", "assignee alice", "assignee alice"},
		{"dotted login gets braced", "assignee alice.qa", "assignee {alice.qa}"},
		{"hyphenated login gets braced", "reporter bob-dev", "reporter {bob-dev}"},
		{"email-style login gets braced", "assignee bob@example.com", "assignee {bob@example.com}"},
		{"already-braced login is untouched", "assignee {alice.qa}", "assignee {alice.qa}"},
		{"trailing query is preserved", "reporter bob-dev State Fixed", "reporter {bob-dev} State Fixed"},
		{"non-user keyword is left alone", "field alice.qa", "field alice.qa"},
		{"keyword casing is preserved", "Assignee alice.qa", "Assignee {alice.qa}"},
		{"multiple matches both get braced", "assignee alice.qa reporter bob-dev", "assignee {alice.qa} reporter {bob-dev}"},
		{"empty string is untouched", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := WrapUserValues(c.in); got != c.out {
				t.Errorf("WrapUserValues(%q) = %q, want %q", c.in, got, c.out)
			}
		})
	}
}
