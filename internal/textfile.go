package internal

import (
	"fmt"

	"github.com/spf13/cobra"
)

// TextFromFlags returns the inline text flag or the content of the file flag, where "-" reads standard input.
// set reports whether either flag was given.
func TextFromFlags(cmd *cobra.Command, textFlag, fileFlag string) (text string, set bool, err error) {
	if !cmd.Flags().Changed(fileFlag) {
		text, _ = cmd.Flags().GetString(textFlag)
		return text, cmd.Flags().Changed(textFlag), nil
	}
	path, _ := cmd.Flags().GetString(fileFlag)
	b, err := ReadInput(cmd, path)
	if err != nil {
		return "", false, fmt.Errorf("read --%s: %w", fileFlag, err)
	}
	return string(b), true, nil
}

// AddTextFileFlag registers fileFlag as a file or stdin alternative to textFlag; the two are mutually exclusive.
func AddTextFileFlag(cmd *cobra.Command, textFlag, fileFlag, what string) {
	cmd.Flags().String(fileFlag, "", "Read "+what+" from a file, or from standard input with -")
	cmd.MarkFlagsMutuallyExclusive(textFlag, fileFlag)
}
