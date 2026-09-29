package output

import (
	"bytes"
	"fmt"
	"testing"
)

func TestSanitizeTerminal(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		{"plain text untouched", "Fix login bug", "Fix login bug"},
		{"tabs and newlines kept", "a\tb\nc", "a\tb\nc"},
		{"unicode kept", "Ошибка — 修复 ✓", "Ошибка — 修复 ✓"},
		{"CSI color sequence stripped", "\x1b[31mred\x1b[0m", "[31mred[0m"},
		{"OSC 8 hyperlink stripped", "\x1b]8;;https://evil\x1b\\click\x1b]8;;\x1b\\", "]8;;https://evil\\click]8;;\\"},
		{"OSC title set with BEL stripped", "\x1b]0;pwned\x07", "]0;pwned"},
		{"carriage return stripped", "safe\rEVIL", "safeEVIL"},
		{"backspace stripped", "ab\bc", "abc"},
		{"DEL stripped", "a\x7fb", "ab"},
		{"UTF-8 C1 CSI stripped", "a\u009b31mb", "a31mb"},
		{"raw 8-bit C1 byte stripped", "a\x9b31mb", "a31mb"},
		{"other invalid UTF-8 byte kept", "a\xffb", "a\xffb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sanitizeTerminal(c.in); got != c.out {
				t.Errorf("sanitizeTerminal(%q) = %q, want %q", c.in, got, c.out)
			}
		})
	}
}

func TestSafeWriterReportsFullLength(t *testing.T) {
	var buf bytes.Buffer
	n, err := fmt.Fprint(safeWriter{&buf}, "a\x1bb")
	if err != nil || n != 3 {
		t.Fatalf("Fprint = (%d, %v), want (3, nil)", n, err)
	}
	if buf.String() != "ab" {
		t.Errorf("wrote %q, want %q", buf.String(), "ab")
	}
}
