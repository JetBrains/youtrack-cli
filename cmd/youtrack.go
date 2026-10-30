package cmd

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/JetBrains/youtrack-cli/api"
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
			if updatePromptShown { // the update prompt is the whole answer; help would push it off screen
				return
			}
			if !term.IsTerminal(int(os.Stdout.Fd())) { // agents and pipes read the full command list
				_ = cmd.Help()
				if noInstanceConfigured() {
					output.PrintTip(cmd, "No YouTrack instance configured — run 'youtrack auth login' to connect one")
				}
				return
			}
			output.PrintLogo(cmd.Context(), os.Stdout, firstRun())
			printNextSteps(cmd.OutOrStdout(), cmd.Long, !noInstanceConfigured())
		},
	}

	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := rejectLocalDryRun(cmd); err != nil {
			return err
		}
		maybePromptForUpdate(cmd)
		return nil
	}

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
	f.Bool("mute-notifications", false, "Don't notify watchers about writes; needs Apply Commands Silently in each affected project, otherwise the write fails with HTTP 403")
	f.Bool("dry-run", false, "Print the first write request (method, URL, body) instead of sending it; reads still run")
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
	var dryRun *api.DryRunError
	if errors.As(execErr, &dryRun) {
		execErr = output.RenderDryRun(root.OutOrStdout(), dryRun, jsonMode)
	}
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

// firstRun reports whether this is the first bare `youtrack` run in a terminal on this machine and
// records that it happened, so the intro animation plays once and later runs are instant.
func firstRun() bool {
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

// noInstanceConfigured reports whether neither YOUTRACK_BASE_URL nor a saved instance tells the CLI which YouTrack to use.
func noInstanceConfigured() bool {
	if os.Getenv("YOUTRACK_BASE_URL") != "" {
		return false
	}
	cfg, err := config.Load()
	return err == nil && len(cfg.Instances) == 0
}

// printNextSteps prints the short start screen of bare `youtrack` in a terminal: what the CLI is and
// what to run next. `youtrack --help` lists every command.
func printNextSteps(w io.Writer, about string, configured bool) {
	title, steps := "Get started:", [][2]string{
		{"youtrack auth login", "Connect to your YouTrack instance"},
	}
	if configured {
		title, steps = "Try:", [][2]string{
			{`youtrack issue list --query "for: me"`, "List issues assigned to you"},
			{"youtrack auth status", "Show the connected instance and user"},
		}
	}
	steps = append(steps, [2]string{"youtrack --help", "Show all commands"})
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintf(tw, "%s\n\n%s\n", about, title)
	for _, s := range steps {
		_, _ = fmt.Fprintf(tw, "  %s\t%s\n", s[0], s[1])
	}
	_ = tw.Flush()
}

// localStateGroups change credentials, config, installed skills, or the binary itself without the API client, so --dry-run cannot hold them back.
var localStateGroups = map[string]bool{"auth": true, "instance": true, "skill": true, "update": true}

func rejectLocalDryRun(cmd *cobra.Command) error {
	if dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run"); !dryRun {
		return nil
	}
	top := cmd
	for top.HasParent() && top.Parent().HasParent() {
		top = top.Parent()
	}
	if top.HasParent() && localStateGroups[top.Name()] {
		return fmt.Errorf("--dry-run covers YouTrack API writes only; %q changes local state and does not support it", cmd.CommandPath())
	}
	return nil
}
