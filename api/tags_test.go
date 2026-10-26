package api

import (
	"strings"
	"testing"
)

func TestIsTagID(t *testing.T) {
	for ref, want := range map[string]bool{
		"6-12":       true,
		"10-0":       true,
		"6-":         false,
		"-12":        false,
		"star":       false,
		"6-12 extra": false,
		"v1-2":       false,
		"1-2-3":      false,
		"":           false,
	} {
		if got := IsTagID(ref); got != want {
			t.Errorf("IsTagID(%q) = %v, want %v", ref, got, want)
		}
	}
}

func TestPickTagByName(t *testing.T) {
	tags := []Tag{
		{ID: "6-1", Name: "needs triage", Owner: &User{Login: "alice"}},
		{ID: "6-2", Name: "Needs Triage", Owner: &User{Login: "alice"}},
		{ID: "6-3", Name: "needs triage later", Owner: &User{Login: "bob"}},
		{ID: "6-4", Name: "shared", Owner: &User{Login: "alice"}},
		{ID: "6-5", Name: "shared", Owner: &User{Login: "bob"}},
		{ID: "6-6", Name: "shared"},
	}
	for _, tc := range []struct {
		name    string
		ref     string
		wantID  string
		wantErr []string
	}{
		{name: "exact match", ref: "needs triage", wantID: "6-1"},
		{name: "case matters", ref: "Needs Triage", wantID: "6-2"},
		{name: "different case only", ref: "NEEDS TRIAGE", wantErr: []string{`no tag named exactly "NEEDS TRIAGE"`, `"needs triage" (6-1)`, `"Needs Triage" (6-2)`}},
		{name: "prefix does not match", ref: "needs", wantErr: []string{`no tag named "needs"`, "tag create"}},
		{name: "ambiguous", ref: "shared", wantErr: []string{`3 tags named "shared"`, "6-4 (owner alice)", "6-5 (owner bob)", "6-6 (unknown owner)", "pass a tag ID"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id, err := pickTagByName(tags, tc.ref, "visible to you")
			if tc.wantErr == nil {
				if err != nil || id != tc.wantID {
					t.Fatalf("pickTagByName(%q) = %q, %v; want %q", tc.ref, id, err, tc.wantID)
				}
				return
			}
			if err == nil {
				t.Fatalf("pickTagByName(%q) = %q, want error", tc.ref, id)
			}
			for _, part := range tc.wantErr {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q does not contain %q", err, part)
				}
			}
		})
	}
}
