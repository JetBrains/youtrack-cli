package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "youtrack",
	Short: "YouTrack REST API CLI",
	Long:  "CLI for interacting with the YouTrack REST API. Designed for AI agents and developers.",
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
}
