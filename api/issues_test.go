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

func TestReadableIssueIDRe(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"FOO-42", true},
		{"foo-1", true},
		{"MY_PROJ2-7", true},
		{"2-15", false},
		{"FOO", false},
		{"FOO-", false},
		{"-42", false},
		{"FOO-42-1", false},
		{"FOO-4x", false},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := readableIssueIDRe.MatchString(tt.id); got != tt.want {
				t.Errorf("readableIssueIDRe.MatchString(%q) = %t, want %t", tt.id, got, tt.want)
			}
		})
	}
}
