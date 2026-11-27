package internal

import (
	"strings"
	"testing"
	"unicode/utf16"
)

// at returns the UTF-16 offset of the first occurrence of fragment in text.
func at(t *testing.T, text, fragment string) TextAnchor {
	t.Helper()
	i := strings.Index(text, fragment)
	if i < 0 {
		t.Fatalf("%q not in %q", fragment, text)
	}
	start := len(utf16.Encode([]rune(text[:i])))
	return TextAnchor{ID: fragment, Start: start, End: start + len(utf16.Encode([]rune(fragment)))}
}

func TestReanchor(t *testing.T) {
	const base = "# Title\n\nalpha beta gamma\n\ndelta epsilon zeta\n\neta theta iota\n"
	tests := []struct {
		name      string
		old, new  string
		fragments []string
		wantMoved []string // fragments whose anchors must move to their position in new
		wantLost  []string
	}{
		{name: "identical text", old: base, new: base, fragments: []string{"beta"}},
		{
			name: "thread before the edit stays", old: base,
			new:       strings.Replace(base, "eta theta", "eta THETA", 1),
			fragments: []string{"beta", "epsilon"},
		},
		{
			name: "thread after the edit shifts", old: base,
			new:       strings.Replace(base, "# Title", "# A much longer title", 1),
			fragments: []string{"beta", "theta"}, wantMoved: []string{"beta", "theta"},
		},
		{
			name: "thread inside the edit is found by context", old: base,
			new:       strings.Replace(base, "delta epsilon zeta", "inserted words delta epsilon and more", 1),
			fragments: []string{"epsilon"}, wantMoved: []string{"epsilon"},
		},
		{
			name: "rewritten fragment is lost", old: base,
			new:       strings.Replace(base, "delta epsilon zeta", "something else", 1),
			fragments: []string{"epsilon", "theta"}, wantMoved: []string{"theta"}, wantLost: []string{"epsilon"},
		},
		{
			name: "ambiguous fragment without context is lost",
			old:  "one x two\n", new: "x x x\n",
			fragments: []string{"x"}, wantLost: []string{"x"},
		},
		{
			name: "offsets count UTF-16 code units", old: "привет 😀 мир\nконец\n",
			new:       "😀😀 привет 😀 мир\nконец\n",
			fragments: []string{"мир", "конец"}, wantMoved: []string{"мир", "конец"},
		},
		{
			name:      "rewritten fragment is not moved onto a copy in unchanged text",
			old:       "See Foo here.\n\nThe thread is on Foo in this sentence.\n",
			new:       "See Foo here.\n\nThe sentence was rewritten.\n",
			fragments: []string{"on Foo"}, wantLost: []string{"on Foo"},
		},
		{
			name: "edit splitting a surrogate pair shifts nothing",
			old:  "a😀b tail\n", new: "a😃b tail\n",
			fragments: []string{"tail"},
		},
		{
			name: "moved paragraph keeps its threads", old: base,
			new:       "# Title\n\neta theta iota\n\nalpha beta gamma\n\ndelta epsilon zeta\n",
			fragments: []string{"theta", "beta"}, wantMoved: []string{"theta", "beta"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var anchors []TextAnchor
			for _, f := range tt.fragments {
				anchors = append(anchors, at(t, tt.old, f))
			}
			moved, lost := Reanchor(tt.old, tt.new, anchors)

			wantMoved := map[string]TextAnchor{}
			for _, f := range tt.wantMoved {
				wantMoved[f] = at(t, tt.new, f)
			}
			if len(moved) != len(wantMoved) {
				t.Fatalf("moved = %v, want %v", moved, tt.wantMoved)
			}
			for _, m := range moved {
				if want, ok := wantMoved[m.ID]; !ok || want != m {
					t.Errorf("moved %v, want %v", m, want)
				}
			}
			if len(lost) != len(tt.wantLost) {
				t.Fatalf("lost = %v, want %v", lost, tt.wantLost)
			}
			for i, l := range lost {
				if l != at(t, tt.old, tt.wantLost[i]) {
					t.Errorf("lost %v must keep its old offsets", l)
				}
			}
		})
	}
}

func TestReanchorInvalidOffsets(t *testing.T) {
	anchors := []TextAnchor{{ID: "neg", Start: -1, End: -1}, {ID: "past", Start: 2, End: 99}, {ID: "rev", Start: 3, End: 1}}
	moved, lost := Reanchor("abcdef", "abcdefg", anchors)
	if len(moved) != 0 || len(lost) != 3 {
		t.Fatalf("moved = %v, lost = %v; invalid anchors must be reported as lost", moved, lost)
	}
}
