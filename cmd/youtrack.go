package cmd

import (
	"bytes"
	"embed"
	"os"

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
		&cobra.Group{ID: "youtrack", Title: "YouTrack Commands:"},
		&cobra.Group{ID: "access", Title: "Access Commands:"},
		&cobra.Group{ID: "util", Title: "Utilities:"},
	)

	add := func(cmd *cobra.Command, groupID string) {
		cmd.GroupID = groupID
		rootCmd.AddCommand(cmd)
	}

	add(auth.NewCmd(), "access")
	add(instance.NewCmd(), "access")

	add(issue.NewCmd(), "youtrack")
	add(project.NewCmd(), "youtrack")
	add(user.NewCmd(), "youtrack")
	add(newTagCmd(), "youtrack")
	add(newSearchCmd(), "youtrack")
	add(newCanvasCmd(), "youtrack")
	add(newGanttCmd(), "youtrack")
	add(newArticleCmd(), "youtrack")
	add(newBoardCmd(), "youtrack")
	add(newDashboardCmd(), "youtrack")

	add(newSettingsCmd(), "youtrack")
	add(newVCSCmd(), "youtrack")
	add(newMailboxCmd(), "youtrack")
	add(newOrgCmd(), "youtrack")
	add(newAppCmd(), "youtrack")
	add(newGroupCmd(), "youtrack")
	add(newRoleCmd(), "youtrack")
	add(newWorkflowCmd(), "youtrack")
	add(newFieldCmd(), "youtrack")

	add(newRawCmd(), "util")
	add(skill.NewCmd(fs), "util")

	return rootCmd
}

// Execute is the entrypoint called from main.go.
func Execute(fs embed.FS) error {
	root := NewRootCmd(fs)
	if err := root.Execute(); err != nil {
		format := output.Format(root)
		quiet, _ := root.PersistentFlags().GetBool("quiet")
		output.RenderError(root.ErrOrStderr(), root.OutOrStdout(), err, format == "json", quiet)
		return err
	}
	return nil
}
