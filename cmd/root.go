package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/JetBrains/youtrack-cli/cmd/output"
)

var rootCmd = &cobra.Command{
	Use:   "youtrack",
	Short: "YouTrack REST API CLI",
	Long:  "CLI for interacting with the YouTrack REST API. Designed for AI agents and developers.",
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintLogo(os.Stdout)
		cmd.Help()
	},
}

// Execute is the entrypoint called from main.
func Execute() error {
	loadEnvFile()
	return rootCmd.Execute()
}

func init() {
	f := rootCmd.PersistentFlags()
	f.String("format", "json", "Output format: json or table")
	f.String("base-url", "", "Override YOUTRACK_BASE_URL")
	f.String("token", "", "Override YOUTRACK_TOKEN")
	f.Bool("verbose", false, "Print HTTP request/response to stderr")

	rootCmd.AddCommand(newIssueCmd())
	rootCmd.AddCommand(newProjectCmd())
	rootCmd.AddCommand(newUserCmd())
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
}
