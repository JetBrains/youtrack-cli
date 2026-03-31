package output

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"
)

const logo = ` ⢇⢸ ⢀⡀ ⡀⢀ ⢹⠁ ⡀⣀ ⢀⣀ ⢀⣀ ⡇⡠
  ⠇ ⠣⠜ ⠣⠼ ⠸  ⠏  ⠣⠼ ⠣⠤ ⠏⠢`

const (
	ansiBrightMagenta = "\033[1;35m"
	ansiMagenta       = "\033[35m"
	ansiDimMagenta    = "\033[2;35m"
	ansiReset         = "\033[0m"
)

// glyphs use braille characters to match the Small Braille aesthetic during flicker.
var glyphs = []rune("⠁⠂⠃⠄⠅⠆⠇⡀⡁⡂⠈⠉⠊⠋⠌⠍⠎⠏⠐⠑⡄⡅⡆⡇⢀⢁⢂⢃⢄⢅⢆⢇⣀⣁⣂⣃")

// PrintLogo prints the YouTrack ASCII logo to w.
// If w is a terminal, it plays a matrix-style reveal animation in magenta.
// Non-terminal output (pipes, agents) is silently skipped.
func PrintLogo(w io.Writer) {
	if !isTerminal(w) {
		return
	}

	lines := strings.Split(logo, "\n")
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

	rowColor := func(r int) string {
		if r == 0 {
			return ansiBrightMagenta
		}
		return ansiMagenta
	}

	render := func() {
		for r, line := range lines {
			color := rowColor(r)
			for c, ch := range []rune(line) {
				switch {
				case ch == ' ':
					fmt.Fprint(w, " ")
				case revealed[pos{r, c}]:
					fmt.Fprint(w, color+string(ch)+ansiReset)
				default:
					fmt.Fprint(w, ansiDimMagenta+string(glyphs[rand.Intn(len(glyphs))])+ansiReset)
				}
			}
			fmt.Fprint(w, "\033[K\n")
		}
	}

	moveUp := fmt.Sprintf("\033[%dA", height)
	frame := func(d time.Duration) {
		render()
		time.Sleep(d)
		fmt.Fprint(w, moveUp)
	}

	fmt.Fprint(w, "\033[?25l\n") // hide cursor + blank line before logo
	defer fmt.Fprint(w, "\033[?25h\n") // show cursor + newline after logo

	// Initial flicker
	for range 10 {
		frame(50 * time.Millisecond)
	}

	// Progressive reveal in random order
	perFrame := max(len(chars)/15, 2)
	for i := 0; i < len(chars); i += perFrame {
		end := min(i+perFrame, len(chars))
		for j := i; j < end; j++ {
			revealed[chars[j]] = true
		}
		frame(40 * time.Millisecond)
	}

	// Hold
	for range 6 {
		frame(50 * time.Millisecond)
	}

	// Final clean render with 3D gradient
	for r, line := range lines {
		fmt.Fprint(w, rowColor(r)+line+ansiReset+"\033[K\n")
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
