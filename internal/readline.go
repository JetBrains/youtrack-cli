package internal

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ReadLineContext reads a line from reader, returning ctx.Err() as soon as ctx is
// canceled — even while the underlying read is still blocked waiting on stdin.
// A plain bufio.Reader.ReadString call ignores context cancellation entirely, which
// is why Ctrl+C during an interactive prompt (e.g. "auth login") did not stop the CLI.
func ReadLineContext(ctx context.Context, reader *bufio.Reader) (string, error) {
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

// ReadPasswordContext reads a password from fd (with terminal echo disabled),
// returning ctx.Err() as soon as ctx is canceled — see ReadLineContext.
func ReadPasswordContext(ctx context.Context, fd int) ([]byte, error) {
	type result struct {
		pw  []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		pw, err := term.ReadPassword(fd)
		ch <- result{pw, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.pw, r.err
	}
}

// ReadLineWithDefault prompts on the terminal stdin with value pre-filled and editable, like a form field; Ctrl+C or Ctrl+D returns context.Canceled.
func ReadLineWithDefault(ctx context.Context, prompt, value string) (string, error) {
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(fd, state) }()

	// The pre-filled value is fed in as typed input, so the line editor echoes it and lets the user edit it.
	t := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{io.MultiReader(strings.NewReader(value), os.Stdin), os.Stderr}, prompt)

	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := t.ReadLine()
		ch <- result{line, err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ch:
		if errors.Is(r.err, io.EOF) {
			_, _ = os.Stderr.WriteString("\r\n")
			return "", context.Canceled
		}
		return r.line, r.err
	}
}
