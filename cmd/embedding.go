package cmd

import (
	"context"
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// WithClientProvider returns an execution context with an optional API client
// source for commands that use the normal credential resolver. Pass this context
// to ExecuteContext. Returning (nil, nil) uses the built-in credential resolution;
// an error stops the command. Authentication and saved-instance management may
// create their own clients and are not redirected by this provider.
//
// The CLI copies a supplied client before applying the command context, verbose
// flag, additional fields projection, and default CLI User-Agent. Existing Fields
// and a custom UserAgent are kept.
// The HTTPClient and its transport are shared with the supplied client, so their
// redirect, timeout, and credential-handling policies remain the provider's.
func WithClientProvider(ctx context.Context, provider func(*cobra.Command) (*api.Client, error)) context.Context {
	return internal.WithClientProvider(ctx, provider)
}

// OutputFormat returns the command's effective format, including TTY detection.
func OutputFormat(command *cobra.Command) string {
	return output.Format(command)
}

// PrintOutput writes data to the command's output using its format and projection.
func PrintOutput(command *cobra.Command, data any) error {
	return output.Print(command, data)
}

// RenderError writes an error to the command's error output using its format and
// quiet flag, preserving the CLI's JSON error codes and suggestions. It returns
// the error for the host's exit policy, or nil after a successful request preview.
func RenderError(command *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	// Optional preview errors can be rendered without depending on write controls.
	var preview interface{ RenderPreview(io.Writer, bool) error }
	if errors.As(err, &preview) {
		return preview.RenderPreview(command.OutOrStdout(), OutputFormat(command) == "json")
	}
	quiet, _ := command.Root().PersistentFlags().GetBool("quiet")
	output.RenderErrorForCommand(command.ErrOrStderr(), err, OutputFormat(command) == "json", quiet, command.Root().Name())
	return err
}
