package issue

import (
	"fmt"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/spf13/cobra"
)

// textFromFlags returns the inline text flag or the content of the file flag, where "-" reads standard input.
// set reports whether either flag was given.
func textFromFlags(cmd *cobra.Command, textFlag, fileFlag string) (text string, set bool, err error) {
	if !cmd.Flags().Changed(fileFlag) {
		text, _ = cmd.Flags().GetString(textFlag)
		return text, cmd.Flags().Changed(textFlag), nil
	}
	path, _ := cmd.Flags().GetString(fileFlag)
	b, err := internal.ReadInput(cmd, path)
	if err != nil {
		return "", false, fmt.Errorf("read --%s: %w", fileFlag, err)
	}
	return string(b), true, nil
}

func addTextFileFlag(cmd *cobra.Command, textFlag, fileFlag, what string) {
	cmd.Flags().String(fileFlag, "", "Read "+what+" from a file, or from standard input with -")
	cmd.MarkFlagsMutuallyExclusive(textFlag, fileFlag)
}
