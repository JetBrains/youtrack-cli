package internal

import "testing"

func TestExpandFields(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		{"empty input stays empty", "", ""},
		{"bare reporter expands", "reporter", "reporter(id,login,fullName)"},
		{"bare updater expands", "updater", "updater(id,login,fullName)"},
		{"bare draftOwner expands", "draftOwner", "draftOwner(id,login,fullName)"},
		{"bare project expands", "project", "project(id,shortName,name)"},
		{"unmapped field is untouched", "summary", "summary"},
		{"field with sub-spec is not double-expanded", "reporter(login)", "reporter(login)"},
		{"mixed list expands only bare defaults", "idReadable,reporter,summary", "idReadable,reporter(id,login,fullName),summary"},
		{"paren-aware split ignores commas inside sub-spec", "reporter(login,fullName),project", "reporter(login,fullName),project(id,shortName,name)"},
		{"surrounding whitespace is trimmed", " reporter , project ", "reporter(id,login,fullName),project(id,shortName,name)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := expandFields(c.in); got != c.out {
				t.Errorf("expandFields(%q) = %q, want %q", c.in, got, c.out)
			}
		})
	}
}

func TestSplitFieldsInternal(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  []string
	}{
		{"single field", "summary", []string{"summary"}},
		{"simple comma list", "id,summary", []string{"id", "summary"}},
		{"nested parens are not split on", "comments(id,text),summary", []string{"comments(id,text)", "summary"}},
		{"nested commas at multiple depths", "reporter(login,fullName),project(shortName)", []string{"reporter(login,fullName)", "project(shortName)"}},
		{"whitespace around fields is trimmed", " id , summary ", []string{"id", "summary"}},
		{"empty string yields one empty field", "", []string{""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := splitFieldsInternal(c.in)
			if len(got) != len(c.out) {
				t.Fatalf("splitFieldsInternal(%q) = %v, want %v", c.in, got, c.out)
			}
			for i := range got {
				if got[i] != c.out[i] {
					t.Errorf("splitFieldsInternal(%q)[%d] = %q, want %q", c.in, i, got[i], c.out[i])
				}
			}
		})
	}
}
