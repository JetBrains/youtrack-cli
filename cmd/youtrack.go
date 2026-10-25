package cmd

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/JetBrains/youtrack-cli/cmd/auth"
	"github.com/JetBrains/youtrack-cli/cmd/instance"
	"github.com/JetBrains/youtrack-cli/cmd/issue"
	"github.com/JetBrains/youtrack-cli/cmd/project"
	"github.com/JetBrains/youtrack-cli/cmd/skill"
	"github.com/JetBrains/youtrack-cli/cmd/user"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
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
			animate := firstRun()
			output.PrintLogo(cmd.Context(), os.Stdout, animate)
			if !animate {
				_ = cmd.Help()
				return
			}
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			_ = cmd.Help()
			cmd.SetOut(os.Stdout)
			output.PrintHelpAnimated(cmd.Context(), os.Stdout, buf.String())
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
		// No notice to print, so don't wait; record a release that has already arrived.
		select {
		case latest, ok := <-done:
			if ok {
				internal.RecordLatestVersion(latest)
			}
		default:
		}
		return execErr
	}
	// The throttle timestamp is persisted synchronously in CheckForUpdateAsync, so bounding the wait here
	// only risks skipping this run's notice, never a repeated API call. On throttled runs done is already closed.
	select {
	case latest, ok := <-done:
		if !ok {
			break
		}
		internal.RecordLatestVersion(latest)
		// The startup prompt already offered this release; repeating it as a notice would contradict the answer.
		if !updatePromptShown {
			if notice := internal.UpdateNotice(internal.Version, latest); notice != "" {
				_, _ = fmt.Fprintln(root.ErrOrStderr(), notice)
			}
		}
	case <-time.After(updateNoticeWait):
	}
	return execErr
}

// firstRun reports whether this is the first bare `youtrack` run on this machine and records
// that it happened, so the intro animation plays once and later help output is instant.
func firstRun() bool {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	dir, err := config.ConfigDir()
	if err != nil {
		return false
	}
	marker := filepath.Join(dir, ".intro-shown")
	if _, err := os.Stat(marker); err == nil {
		return false
	}
	if os.MkdirAll(dir, 0o700) != nil || os.WriteFile(marker, nil, 0o600) != nil {
		return false // ponytail: can't record it, so skip the animation rather than replay it every run
	}
	return true
}
