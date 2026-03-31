package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

// NewRootCmd constructs the root command. Called by Execute() and by tests.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "youtrack",
		Short: "YouTrack REST API CLI",
		Long:  "CLI for interacting with the YouTrack REST API. Designed for AI agents and developers.",
		Run: func(cmd *cobra.Command, args []string) {
			output.PrintLogo(os.Stdout)
			cmd.Help()
		},
	}

	f := rootCmd.PersistentFlags()
	f.String("format", "json", "Output format: json or table")
	f.String("base-url", "", "Override YOUTRACK_BASE_URL")
	f.String("token", "", "Override YOUTRACK_TOKEN")
	f.Bool("verbose", false, "Print HTTP request/response to stderr")

	// Domain sub-packages (will switch to issue.NewCmd() etc. in Tasks 3-5)
	rootCmd.AddCommand(newIssueCmd())
	rootCmd.AddCommand(newProjectCmd())
	rootCmd.AddCommand(newUserCmd())

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

	return rootCmd
}

// Execute is the entrypoint called from main.go.
func Execute() error {
	internal.LoadEnvFile()
	return NewRootCmd().Execute()
}
