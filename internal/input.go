package internal

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

// ReadInput returns the contents of path, or of cmd's stdin when path is "-".
func ReadInput(cmd *cobra.Command, path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(path)
}
