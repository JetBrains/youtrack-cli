package cmd

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/auth"
	"github.com/JetBrains/youtrack-cli/cmd/instance"
	"github.com/JetBrains/youtrack-cli/cmd/issue"
	"github.com/JetBrains/youtrack-cli/cmd/project"
	"github.com/JetBrains/youtrack-cli/cmd/skill"
	"github.com/JetBrains/youtrack-cli/cmd/user"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

const updateNoticeWait = 500 * time.Millisecond

// NewRootCmd constructs the root command. Called by Execute() and by tests.
func NewRootCmd(fs embed.FS) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "youtrack",
		Short:         "YouTrack CLI",
		Long:          "CLI for interacting with the YouTrack REST API. Designed for AI agents and developers.",
		Version:       internal.Version + " (" + internal.Commit + ", " + internal.Date + ")",
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			output.PrintLogo(os.Stdout)
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			_ = cmd.Help()
			cmd.SetOut(os.Stdout)
			output.PrintHelpAnimated(os.Stdout, buf.String())
		},
	}

	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) { maybePromptForUpdate(cmd) }

	f := rootCmd.PersistentFlags()
	f.String("format", "table", "Output format: table, json, or plain (deprecated: use --json or --plain)")
	f.BoolP("json", "j", false, "Output as JSON (auto-enabled when stdout is not a TTY)")
	f.Bool("plain", false, "Tab-separated output, no headers or colors (for grep/awk pipelines)")
	f.StringP("server", "s", "", "Select named server profile (see 'youtrack instance list')")
	f.StringP("fields", "f", "", "Comma-separated field projection (e.g. id,summary,project.shortName)")
	f.String("base-url", "", "Override YOUTRACK_BASE_URL")
	f.String("token", "", "Override YOUTRACK_TOKEN")
	f.Bool("verbose", false, "Print HTTP request/response to stderr")
	f.Bool("quiet", false, "Suppress tips and hints (useful for scripting)")
	_ = f.MarkHidden("format")

	rootCmd.AddGroup(
		&cobra.Group{ID: "core", Title: "Core:"},
		&cobra.Group{ID: "admin", Title: "Administration:"},
		&cobra.Group{ID: "integrations", Title: "Integrations:"},
		&cobra.Group{ID: "planning", Title: "Planning:"},
		&cobra.Group{ID: "access", Title: "Access:"},
		&cobra.Group{ID: "util", Title: "Utilities:"},
	)

	internal.AddToGroup(rootCmd, "core",
		issue.NewCmd(), project.NewCmd(), newArticleCmd(),
		newBoardCmd(), newTagCmd(), newSearchCmd(),
	)
	internal.AddToGroup(rootCmd, "admin",
		user.NewCmd(), newGroupCmd(), newRoleCmd(), newOrgCmd(),
		newFieldCmd(), newWorkflowCmd(), newSettingsCmd(),
	)
	internal.AddToGroup(rootCmd, "integrations",
		newAppCmd(),
	)
	internal.AddToGroup(rootCmd, "planning",
		newDashboardCmd(), newGanttCmd(), newWhiteboardCmd(),
	)
	internal.AddToGroup(rootCmd, "access",
		auth.NewCmd(), instance.NewCmd(),
	)
	internal.AddToGroup(rootCmd, "util",
		newRawCmd(), newUpdateCmd(), skill.NewCmd(fs),
	)

	return rootCmd
}

// Execute is the entrypoint called from main.go.
func Execute(fs embed.FS) error {
	root := NewRootCmd(fs)
	done := internal.CheckForUpdateAsync(internal.Version)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	execErr := root.ExecuteContext(ctx)
	quiet, _ := root.PersistentFlags().GetBool("quiet")
	jsonMode := output.Format(root) == "json"
	if execErr != nil {
		// A user-initiated Ctrl+C during an interactive prompt cancels the root
		// context; that's not a failure worth an "✗ Error" — print a plain
		// cancellation notice instead (skipped in JSON mode: there's no
		// machine-readable "canceled" state worth emitting to a script).
		if errors.Is(execErr, context.Canceled) {
			if !jsonMode {
				_, _ = fmt.Fprintln(root.ErrOrStderr(), "Canceled.")
			}
		} else {
			output.RenderError(root.ErrOrStderr(), execErr, jsonMode, quiet)
		}
	}
	// Suppress the plain-text update notice when a JSON error envelope was written to stderr, so that stream stays parseable.
	jsonErrEmitted := jsonMode && execErr != nil
	if quiet || jsonErrEmitted {
		return execErr
	}
	// The throttle timestamp is persisted synchronously in CheckForUpdateAsync, so bounding the wait here
	// only risks skipping this run's notice, never a repeated API call. On throttled runs done is already closed.
	select {
	case notice := <-done:
		if notice != "" {
			_, _ = fmt.Fprintln(root.ErrOrStderr(), notice)
		}
	case <-time.After(updateNoticeWait):
	}
	return execErr
}
