package api

import (
	"encoding/json"
	"testing"
)

func TestVisibilityChangeMarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		in   VisibilityChange
		want string
	}{
		{"public", VisibilityChange{Public: true, GroupIDs: []string{"3-1"}}, `{"$type":"UnlimitedVisibility"}`},
		{"users only clears groups", VisibilityChange{UserIDs: []string{"1-2"}},
			`{"$type":"LimitedVisibility","permittedGroups":[],"permittedUsers":[{"id":"1-2"}]}`},
		{"groups only clears users", VisibilityChange{GroupIDs: []string{"3-1", "3-2"}},
			`{"$type":"LimitedVisibility","permittedGroups":[{"id":"3-1"},{"id":"3-2"}],"permittedUsers":[]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := json.Marshal(c.in)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}

func TestVisibilityChangeMatches(t *testing.T) {
	limited := &Visibility{
		Type:            "LimitedVisibility",
		PermittedGroups: []Group{{ID: "3-1"}},
		PermittedUsers:  []User{{ID: "1-2"}, {ID: "1-3"}},
	}
	cases := []struct {
		name string
		want VisibilityChange
		got  *Visibility
		ok   bool
	}{
		{"public matches unlimited", VisibilityChange{Public: true}, &Visibility{Type: "UnlimitedVisibility"}, true},
		{"public matches missing visibility", VisibilityChange{Public: true}, nil, true},
		{"public matches empty limited", VisibilityChange{Public: true}, &Visibility{Type: "LimitedVisibility"}, true},
		{"public does not match limited", VisibilityChange{Public: true}, limited, false},
		{"same sets in any order", VisibilityChange{GroupIDs: []string{"3-1"}, UserIDs: []string{"1-3", "1-2"}}, limited, true},
		{"extra group on server", VisibilityChange{UserIDs: []string{"1-2", "1-3"}}, limited, false},
		{"missing user on server", VisibilityChange{GroupIDs: []string{"3-1"}, UserIDs: []string{"1-2", "1-3", "1-4"}}, limited, false},
		{"limited does not match public", VisibilityChange{UserIDs: []string{"1-2"}}, &Visibility{Type: "UnlimitedVisibility"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.want.Matches(c.got); got != c.ok {
				t.Errorf("Matches() = %v, want %v", got, c.ok)
			}
		})
	}
}

func TestVisibilityChangeMatchesAllowingReporter(t *testing.T) {
	withReporter := &Visibility{
		Type:            "LimitedVisibility",
		PermittedGroups: []Group{{ID: "3-1"}},
		PermittedUsers:  []User{{ID: "1-9"}},
	}
	cases := []struct {
		name     string
		want     VisibilityChange
		reporter string
		ok       bool
	}{
		{"reporter added by the server", VisibilityChange{GroupIDs: []string{"3-1"}}, "1-9", true},
		{"another user added", VisibilityChange{GroupIDs: []string{"3-1"}}, "1-8", false},
		{"reporter requested explicitly", VisibilityChange{GroupIDs: []string{"3-1"}, UserIDs: []string{"1-9"}}, "1-9", true},
		{"no reporter known", VisibilityChange{GroupIDs: []string{"3-1"}}, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.want.MatchesAllowing(withReporter, c.reporter); got != c.ok {
				t.Errorf("Matches() = %v, want %v", got, c.ok)
			}
		})
	}
}

func TestDescribeVisibility(t *testing.T) {
	cases := []struct {
		in   *Visibility
		want string
	}{
		{nil, "public"},
		{&Visibility{Type: "UnlimitedVisibility"}, "public"},
		{&Visibility{Type: "LimitedVisibility", PermittedUsers: []User{{ID: "1-2", Login: "alice"}}}, "groups: none; users: alice"},
		{&Visibility{Type: "LimitedVisibility", PermittedGroups: []Group{{ID: "3-1", Name: "Core"}, {ID: "3-2"}}}, "groups: Core, 3-2; users: none"},
	}
	for _, c := range cases {
		if got := DescribeVisibility(c.in); got != c.want {
			t.Errorf("DescribeVisibility(%+v) = %q, want %q", c.in, got, c.want)
		}
	}
}
