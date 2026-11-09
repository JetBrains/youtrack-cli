package output

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

const logo = ` __   __         _____               _
 \ \ / /__  _   |_   _| __ __ _  ___| | __
  \ V / _ \| | | || || '__/ _` + "`" + `|/ __| |/ /
   | | (_) | |_| || || | | (_| | (__|   <
   |_|\___/ \__,_||_||_|  \__,_|\___|_|\_\`

// Phase 1 — katakana rain before the logo materializes.
var katakanaGlyphs = []rune("ｦｱｲｳｴｵｶｷｸｹｺｻｼｽｾｿﾀﾁﾂﾃﾄﾅﾆﾇﾈﾉﾊﾋﾌﾍﾎﾏﾐﾑﾒﾓﾔﾕﾖﾗﾘﾙﾚﾛﾜﾝ")

// Phase 2 — arrows as the logo snaps into focus.
var arrowGlyphs = []rune("←↑→↓↖↗↘↙⇐⇑⇒⇓↺↻↯↩↪")

// Brand gradient of the YouTrack logo, left to right (see internal/browserflow/logo/youtrack.svg).
var (
	gradientFrom = [3]int{0xFB, 0x43, 0xFF}
	gradientTo   = [3]int{0xFB, 0x40, 0x6D}
)

// gradientRGB returns the brand gradient color of column c in a logo that is width columns wide.
func gradientRGB(c, width int) [3]int {
	t := float64(c) / float64(max(width-1, 1))
	var rgb [3]int
	for i := range rgb {
		rgb[i] = gradientFrom[i] + int(math.Round(t*float64(gradientTo[i]-gradientFrom[i])))
	}
	return rgb
}

// columnStyles returns one style per logo column with attr (Bold or Faint): the brand gradient on
// true-color terminals, else ANSI magenta, which the terminal theme maps to its own shade.
func columnStyles(width int, attr color.Attribute) []*color.Color {
	trueColor := os.Getenv("COLORTERM") == "truecolor" || os.Getenv("COLORTERM") == "24bit"
	styles := make([]*color.Color, width)
	for c := range styles {
		if !trueColor {
			styles[c] = color.New(color.FgMagenta, attr)
			continue
		}
		rgb := gradientRGB(c, width)
		styles[c] = color.New(attr).AddRGB(rgb[0], rgb[1], rgb[2])
	}
	return styles
}

// PrintLogo prints the YouTrack ASCII logo to w in the brand gradient.
// When animate is true, it plays a matrix-style reveal animation first; canceling ctx skips to the final logo.
// Non-terminal output (pipes, agents) is silently skipped.
func PrintLogo(ctx context.Context, w io.Writer, animate bool) {
	if !isTerminal(w) {
		return
	}

	lines := strings.Split(logo, "\n")
	width := 0
	for _, line := range lines {
		width = max(width, len(line))
	}
	bright, dim := columnStyles(width, color.Bold), columnStyles(width, color.Faint)
	paint := func(line string) string {
		var b strings.Builder
		for c, ch := range line {
			if ch == ' ' {
				b.WriteRune(ch)
			} else {
				b.WriteString(bright[c].Sprint(string(ch)))
			}
		}
		return b.String()
	}

	if !animate {
		_, _ = fmt.Fprint(w, "\n")
		for _, line := range lines {
			_, _ = fmt.Fprint(w, paint(line)+"\n")
		}
		_, _ = fmt.Fprint(w, "\n")
		return
	}
	height := len(lines)

	type pos struct{ r, c int }
	var chars []pos
	for r, line := range lines {
		for c, ch := range []rune(line) {
			if ch != ' ' {
				chars = append(chars, pos{r, c})
			}
		}
	}
	rand.Shuffle(len(chars), func(i, j int) { chars[i], chars[j] = chars[j], chars[i] })

	revealed := make(map[pos]bool)

	render := func(gs []rune) {
		for r, line := range lines {
			for c, ch := range []rune(line) {
				switch {
				case ch == ' ':
					_, _ = fmt.Fprint(w, " ")
				case revealed[pos{r, c}]:
					_, _ = fmt.Fprint(w, bright[c].Sprint(string(ch)))
				default:
					_, _ = fmt.Fprint(w, dim[c].Sprint(string(gs[rand.Intn(len(gs))])))
				}
			}
			_, _ = fmt.Fprint(w, "\033[K\n")
		}
	}

	moveUp := fmt.Sprintf("\033[%dA", height)
	frame := func(d time.Duration, gs []rune) {
		if ctx.Err() != nil {
			return
		}
		render(gs)
		pause(ctx, d)
		_, _ = fmt.Fprint(w, moveUp)
	}

	_, _ = fmt.Fprint(w, "\033[?25l\n")                    // hide cursor + blank line before logo
	defer func() { _, _ = fmt.Fprint(w, "\033[?25h\n") }() // show cursor + newline after logo

	// Phase 1 — katakana rain
	for range 10 {
		frame(50*time.Millisecond, katakanaGlyphs)
	}

	// Phase 2 — reveal: first half katakana, second half arrows
	perFrame := max(len(chars)/15, 2)
	totalFrames := (len(chars) + perFrame - 1) / perFrame
	for i, frameIdx := 0, 0; i < len(chars); i, frameIdx = i+perFrame, frameIdx+1 {
		end := min(i+perFrame, len(chars))
		for j := i; j < end; j++ {
			revealed[chars[j]] = true
		}
		gs := katakanaGlyphs
		if frameIdx >= totalFrames/2 {
			gs = arrowGlyphs
		}
		frame(40*time.Millisecond, gs)
	}

	// Phase 3 — hold with arrows
	for range 6 {
		frame(50*time.Millisecond, arrowGlyphs)
	}

	// Final clean render
	for _, line := range lines {
		_, _ = fmt.Fprint(w, paint(line)+"\033[K\n")
	}
}

// pause waits for d, or returns at once when ctx is canceled (Ctrl+C).
func pause(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
