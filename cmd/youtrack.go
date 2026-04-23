package cmd

import (
	"bytes"
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
func NewRootCmd() *cobra.Command {
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
	f.String("format", "json", "Output format: json or table")
	f.String("base-url", "", "Override YOUTRACK_BASE_URL")
	f.String("token", "", "Override YOUTRACK_TOKEN")
	f.Bool("verbose", false, "Print HTTP request/response to stderr")
	f.Bool("quiet", false, "Suppress tips and hints (useful for scripting)")

	// Domain sub-packages
	rootCmd.AddCommand(auth.NewCmd())
	rootCmd.AddCommand(instance.NewCmd())
	rootCmd.AddCommand(issue.NewCmd())
	rootCmd.AddCommand(project.NewCmd())
	rootCmd.AddCommand(user.NewCmd())

	// Single-file commands (stay in package cmd)
	rootCmd.AddCommand(newSettingsCmd())
	rootCmd.AddCommand(newVCSCmd())
	rootCmd.AddCommand(newMailboxCmd())
	rootCmd.AddCommand(newTagCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newOrgCmd())
	rootCmd.AddCommand(newAppCmd())
	rootCmd.AddCommand(newCanvasCmd())
	rootCmd.AddCommand(newReportCmd())
	rootCmd.AddCommand(newArticleCmd())
	rootCmd.AddCommand(newGroupCmd())
	rootCmd.AddCommand(newRoleCmd())
	rootCmd.AddCommand(newWorkflowCmd())
	rootCmd.AddCommand(newFieldCmd())
	rootCmd.AddCommand(newBoardCmd())
	rootCmd.AddCommand(newDashboardCmd())
	rootCmd.AddCommand(newRawCmd())
	rootCmd.AddCommand(skill.NewCmd())

	return rootCmd
}

// Execute is the entrypoint called from main.go.
func Execute() error {
	internal.LoadEnvFile()
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		format, _ := root.PersistentFlags().GetString("format")
		quiet, _ := root.PersistentFlags().GetBool("quiet")
		output.RenderError(root.ErrOrStderr(), root.OutOrStdout(), err, format == "json", quiet)
		return err
	}
	return nil
}
