package internal

import (
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// WriteOutput hands write a destination for path, or cmd's stdout when path is "-", and returns the bytes written.
// A temporary sibling is renamed only after the response and file close succeed.
func WriteOutput(cmd *cobra.Command, path string, write func(io.Writer) error) (int64, error) {
	if path == "-" {
		w := &countingWriter{w: cmd.OutOrStdout()}
		err := write(w)
		return w.n, err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".youtrack-download-*")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	w := &countingWriter{w: f}
	err = write(w)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return 0, err
	}
	return w.n, nil
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
