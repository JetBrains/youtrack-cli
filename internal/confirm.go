package internal

import (
	"bufio"
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal/prompt"
)

// Confirm asks "<msg>" as a yes/no question on cmd's stderr (a huh field in a terminal, "[y/N]" on piped stdin), defaulting to no.
// Skips the prompt and returns true immediately if --force is set on cmd.
func Confirm(cmd *cobra.Command, msg string) (bool, error) {
	if force, _ := cmd.Flags().GetBool("force"); force {
		return true, nil
	}
	confirmed := false
	if err := prompt.New(cmd.Context(), bufio.NewReader(cmd.InOrStdin()), cmd.ErrOrStderr()).Confirm(msg, &confirmed); err != nil {
		if cmd.Context().Err() != nil || errors.Is(err, context.Canceled) {
			return false, context.Canceled
		}
		return false, fmt.Errorf("confirmation requires interactive input; use --force to skip")
	}
	return confirmed, nil
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
