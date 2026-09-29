package output

import (
	"io"
	"unicode/utf8"
)

// sanitizeTerminal drops control characters a terminal could interpret (C0 except \t and \n, DEL, C1)
// so untrusted YouTrack content cannot inject escape sequences into human-readable output.
func sanitizeTerminal(s string) string {
	return string(sanitizeTerminalBytes([]byte(s)))
}

func sanitizeTerminalBytes(p []byte) []byte {
	if !needsSanitizing(p) {
		return p
	}
	out := make([]byte, 0, len(p))
	for i := 0; i < len(p); {
		r, size := utf8.DecodeRune(p[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8: keep the byte unless it is a raw 8-bit C1 control.
			if p[i] < 0x80 || p[i] > 0x9f {
				out = append(out, p[i])
			}
		} else if !isTerminalControl(r) {
			out = append(out, p[i:i+size]...)
		}
		i += size
	}
	return out
}

func needsSanitizing(p []byte) bool {
	for i := 0; i < len(p); {
		r, size := utf8.DecodeRune(p[i:])
		if (r == utf8.RuneError && size == 1 && p[i] >= 0x80 && p[i] <= 0x9f) || isTerminalControl(r) {
			return true
		}
		i += size
	}
	return false
}

func isTerminalControl(r rune) bool {
	switch {
	case r == '\t' || r == '\n':
		return false
	case r < 0x20, r == 0x7f:
		return true
	case r >= 0x80 && r <= 0x9f:
		return true
	}
	return false
}

// safeWriter sanitizes each Write; callers write whole formatted strings (fmt.Fprintf) so multi-byte runes are never split across calls.
type safeWriter struct {
	w io.Writer
}

func (s safeWriter) Write(p []byte) (int, error) {
	if _, err := s.w.Write(sanitizeTerminalBytes(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}
