package internal

import "testing"

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"equal versions", "1.2.3", "1.2.3", false},
		{"newer patch", "1.2.3", "1.2.4", true},
		{"older patch", "1.2.4", "1.2.3", false},
		{"newer minor beats patch drop", "1.2.9", "1.3.0", true},
		{"newer major beats minor/patch drop", "1.9.9", "2.0.0", true},
		{"leading v is stripped", "v1.2.3", "v1.2.4", true},
		{"pre-release suffix is ignored", "1.2.3", "1.2.4-rc1", true},
		{"unparsable input defaults to zero", "dev", "1.0.0", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsNewerVersion(c.current, c.latest); got != c.want {
				t.Errorf("IsNewerVersion(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
			}
		})
	}
}
