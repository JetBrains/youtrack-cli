package internal

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Confirm prompts the user with "<msg> [y/N]: " on cmd's stderr and reads from stdin.
// Returns true if the user typed "y" or "yes" (case-insensitive).
// Skips the prompt and returns true immediately if --force is set on cmd.
func Confirm(cmd *cobra.Command, msg string) (bool, error) {
	if force, _ := cmd.Flags().GetBool("force"); force {
		return true, nil
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N]: ", msg); err != nil {
		return false, err
	}
	reader := bufio.NewReader(cmd.InOrStdin())
	answer, err := ReadLineContext(cmd.Context(), reader)
	if err != nil {
		if cmd.Context().Err() != nil {
			return false, err
		}
		return false, fmt.Errorf("confirmation requires interactive input; use --force to skip")
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

// ConfirmOrAbort wraps Confirm for the common "confirm before destructive action" pattern.
// abort is true if the caller should return immediately (either err is non-nil, or the
// user declined and "Canceled." has already been printed).
func ConfirmOrAbort(cmd *cobra.Command, msg string) (abort bool, err error) {
	confirmed, err := Confirm(cmd, msg)
	if err != nil {
		return true, err
	}
	if !confirmed {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
		return true, nil
	}
	return false, nil
}
