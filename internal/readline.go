package internal

import (
	"bufio"
	"context"

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
