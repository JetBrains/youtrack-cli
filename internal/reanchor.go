package internal

import "unicode/utf16"

// reanchorContext is how many UTF-16 code units around a fragment help to find it again after an edit.
const reanchorContext = 30

// TextAnchor is a range of a text, in UTF-16 code units, as YouTrack stores inline thread offsets.
type TextAnchor struct {
	ID    string
	Start int
	End   int
}

// Reanchor maps anchors from oldText onto newText.
// Anchors before the changed region keep their offsets, anchors after it shift by the length difference,
// and anchors that touch it are found again by their fragment and its surrounding context.
// moved lists the anchors whose offsets changed, with the new offsets.
// lost lists the anchors that could not be found, with their old offsets; callers should leave them where they are.
func Reanchor(oldText, newText string, anchors []TextAnchor) (moved, lost []TextAnchor) {
	if oldText == newText {
		return nil, nil
	}
	old, cur := utf16.Encode([]rune(oldText)), utf16.Encode([]rune(newText))
	prefix := commonPrefix(old, cur)
	suffix := commonSuffix(old[prefix:], cur[prefix:])
	changedEnd := len(old) - suffix
	changed := [2]int{prefix, len(cur) - suffix}
	delta := len(cur) - len(old)
	for _, a := range anchors {
		switch {
		case a.Start < 0 || a.End < a.Start || a.End > len(old):
			lost = append(lost, a)
		case a.End <= prefix:
			// Entirely in the unchanged text before the edit.
		case a.Start >= changedEnd:
			if delta != 0 {
				moved = append(moved, TextAnchor{ID: a.ID, Start: a.Start + delta, End: a.End + delta})
			}
		default:
			start := findFragment(old, cur, a.Start, a.End, changed)
			if start < 0 {
				lost = append(lost, a)
			} else if start != a.Start {
				moved = append(moved, TextAnchor{ID: a.ID, Start: start, End: start + a.End - a.Start})
			}
		}
	}
	return moved, lost
}

// findFragment returns where old[start:end] is in cur, judged by its context, or -1 when it is missing or ambiguous.
// The match must overlap the changed region of cur: a copy in the unchanged text is a different occurrence.
func findFragment(old, cur []uint16, start, end int, changed [2]int) int {
	fragment := old[start:end]
	if len(fragment) == 0 {
		return -1
	}
	before := old[max(0, start-reanchorContext):start]
	after := old[end:min(len(old), end+reanchorContext)]
	for _, probe := range [][2][]uint16{{before, after}, {before, nil}, {nil, after}, {nil, nil}} {
		needle := concat(probe[0], fragment, probe[1])
		i := uniqueIndex(cur, needle)
		if i < 0 {
			continue
		}
		if at := i + len(probe[0]); at < changed[1] && at+len(fragment) > changed[0] {
			return at
		}
		return -1
	}
	return -1
}

func commonPrefix(a, b []uint16) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

func commonSuffix(a, b []uint16) int {
	n := 0
	for n < len(a) && n < len(b) && a[len(a)-1-n] == b[len(b)-1-n] {
		n++
	}
	return n
}

// uniqueIndex returns the index of the only occurrence of needle in haystack, or -1 when there is none or more than one.
func uniqueIndex(haystack, needle []uint16) int {
	found := -1
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if !equalAt(haystack, needle, i) {
			continue
		}
		if found >= 0 {
			return -1
		}
		found = i
	}
	return found
}

func equalAt(haystack, needle []uint16, at int) bool {
	for j, u := range needle {
		if haystack[at+j] != u {
			return false
		}
	}
	return true
}

func concat(parts ...[]uint16) []uint16 {
	var out []uint16
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
