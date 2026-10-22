// Package prompt asks interactive questions with the same huh fields and theme as teamcity-cli, falling back to plain line prompts when stdin is not a terminal so scripts can pipe answers in.
package prompt

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"golang.org/x/term"
)

// Option is one choice of a Select.
type Option struct {
	Label string
	Value string
}

// Prompter asks questions on out, reading piped answers from in when stdin is not a terminal.
type Prompter struct {
	ctx context.Context
	in  *bufio.Reader
	out io.Writer
	tty bool
}

// New returns a Prompter that writes to out and reads piped answers from in; all reads share in, so one Prompter (or one reader) must serve a whole command.
func New(ctx context.Context, in *bufio.Reader, out io.Writer) *Prompter {
	return &Prompter{ctx: ctx, in: in, out: out, tty: term.IsTerminal(int(os.Stdin.Fd()))}
}

// Interactive reports whether questions are shown as huh fields.
func (p *Prompter) Interactive() bool {
	return p.tty
}

// Input asks for text, pre-filled with *value and editable; an empty answer is rejected unless *value was pre-filled, in which case piped Enter keeps it.
func (p *Prompter) Input(title string, value *string) error {
	if !p.tty {
		label := title
		if *value != "" {
			label = fmt.Sprintf("%s [%s]", title, *value)
		}
		answer, err := p.readLine(label)
		if err != nil {
			return err
		}
		if answer != "" {
			*value = answer
		}
		return RequireNonEmpty(*value)
	}
	if err := p.run(huh.NewInput().Title(title).Validate(RequireNonEmpty).Value(value)); err != nil {
		return err
	}
	p.echo(title, *value)
	return nil
}

// Secret asks for a hidden value and never echoes it back.
func (p *Prompter) Secret(title string, value *string) error {
	if !p.tty {
		answer, err := p.readLine(title)
		if err != nil {
			return err
		}
		*value = answer
		return RequireNonEmpty(*value)
	}
	return p.run(huh.NewInput().Title(title).EchoMode(huh.EchoModePassword).Validate(RequireNonEmpty).Value(value))
}

// Confirm asks a yes/no question; *value is the default.
func (p *Prompter) Confirm(title string, value *bool) error {
	if !p.tty {
		hint := "[y/N]"
		if *value {
			hint = "[Y/n]"
		}
		answer, err := p.readLine(title + " " + hint)
		if err != nil {
			return err
		}
		switch strings.ToLower(answer) {
		case "y", "yes":
			*value = true
		case "n", "no":
			*value = false
		}
		return nil
	}
	return p.run(huh.NewConfirm().
		Title(title).
		Affirmative("yes").
		Negative("no").
		Inline(true).
		WithButtonAlignment(lipgloss.Left).
		Value(value))
}

// Select asks for one of options; piped answers pick by 1-based number.
func (p *Prompter) Select(title string, options []Option, value *string) error {
	if !p.tty {
		var b strings.Builder
		b.WriteString(title + ":\n")
		for i, o := range options {
			fmt.Fprintf(&b, "  %d) %s\n", i+1, o.Label)
		}
		fmt.Fprintf(&b, "Select [1-%d]", len(options))
		answer, err := p.readLine(b.String())
		if err != nil {
			return err
		}
		n, err := strconv.Atoi(answer)
		if err != nil || n < 1 || n > len(options) {
			return fmt.Errorf("invalid choice %q: enter a number from 1 to %d", answer, len(options))
		}
		*value = options[n-1].Value
		return nil
	}
	huhOptions := make([]huh.Option[string], len(options))
	for i, o := range options {
		huhOptions[i] = huh.NewOption(o.Label, o.Value)
	}
	if err := p.run(huh.NewSelect[string]().Title(title).Options(huhOptions...).Value(value)); err != nil {
		return err
	}
	for _, o := range options {
		if o.Value == *value {
			p.echo(title, o.Label)
		}
	}
	return nil
}

// RequireNonEmpty rejects empty or whitespace-only input.
func RequireNonEmpty(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("value is required")
	}
	return nil
}

func (p *Prompter) readLine(label string) (string, error) {
	if _, err := fmt.Fprintf(p.out, "%s: ", label); err != nil {
		return "", err
	}
	line, err := ReadLine(p.ctx, p.in)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// ReadLine reads a line from reader, returning ctx.Err() as soon as ctx is canceled — even while the read is still blocked on stdin, which a plain ReadString ignores, so Ctrl+C stops the CLI.
func ReadLine(ctx context.Context, reader *bufio.Reader) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := reader.ReadString('\n')
		ch <- result{line, err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ch:
		return r.line, r.err
	}
}

func (p *Prompter) run(field huh.Field) error {
	if in, ok := field.(*huh.Input); ok {
		in.Prompt("")
	}
	err := huh.NewForm(huh.NewGroup(field)).
		WithTheme(theme()).
		WithShowHelp(true).
		WithOutput(p.out).
		RunWithContext(p.ctx)
	if errors.Is(err, huh.ErrUserAborted) {
		return context.Canceled
	}
	return err
}

// echo prints the answer once the field closes, so it survives in scrollback.
func (p *Prompter) echo(title, value string) {
	if value != "" {
		_, _ = fmt.Fprintf(p.out, "%s: %s\n", title, color.CyanString(value))
	}
}

// theme renders prompts in the 16-color palette of teamcity-cli, with no borders or magenta accents.
var theme = sync.OnceValue(func() *huh.Theme {
	t := huh.ThemeBase()

	var (
		cyan   = lipgloss.Color("6")
		green  = lipgloss.Color("2")
		yellow = lipgloss.Color("3")
		red    = lipgloss.Color("1")
		faint  = lipgloss.Color("8")
		plain  = lipgloss.NewStyle()
	)

	t.Focused.Base = plain
	t.Focused.Card = plain
	t.Focused.Title = plain.Bold(true)
	t.Focused.NoteTitle = plain.Bold(true)
	t.Focused.Description = plain.Foreground(faint)
	t.Focused.ErrorIndicator = plain.Foreground(red).SetString(" ✗")
	t.Focused.ErrorMessage = plain.Foreground(red)

	t.Focused.SelectSelector = plain.Foreground(yellow).SetString("→ ")
	t.Focused.NextIndicator = plain.Foreground(yellow).MarginLeft(1).SetString("→")
	t.Focused.PrevIndicator = plain.Foreground(yellow).MarginRight(1).SetString("←")
	t.Focused.Option = plain
	t.Focused.SelectedOption = plain

	t.Focused.MultiSelectSelector = plain.Foreground(yellow).SetString("→ ")
	t.Focused.SelectedPrefix = plain.Foreground(green).SetString("✓ ")
	t.Focused.UnselectedPrefix = plain.Foreground(faint).SetString("• ")
	t.Focused.UnselectedOption = plain

	t.Focused.FocusedButton = plain.Bold(true).Foreground(cyan).MarginLeft(3)
	t.Focused.BlurredButton = plain.Foreground(faint).MarginLeft(3)

	t.Focused.TextInput.Cursor = plain.Foreground(cyan)
	t.Focused.TextInput.Placeholder = plain.Foreground(faint)
	t.Focused.TextInput.Prompt = plain.Foreground(yellow)

	t.Blurred = t.Focused
	t.Blurred.Base = plain
	t.Blurred.Card = plain
	t.Blurred.NextIndicator = plain
	t.Blurred.PrevIndicator = plain

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
})
