package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"youtrack-cli/api"
)

// getClient builds an API client from global flags and env vars.
// Exits with an error message if base URL or token are missing.
func getClient(cmd *cobra.Command) *api.Client {
	flags := cmd.Root().PersistentFlags()

	baseURL, _ := flags.GetString("base-url")
	token, _ := flags.GetString("token")
	verbose, _ := flags.GetBool("verbose")

	if baseURL == "" {
		baseURL = os.Getenv("YOUTRACK_BASE_URL")
	}
	if token == "" {
		token = os.Getenv("YOUTRACK_TOKEN")
	}

	if baseURL == "" {
		fmt.Fprintln(os.Stderr, "error: YOUTRACK_BASE_URL is not set. Set it via env or --base-url flag.")
		os.Exit(1)
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: YOUTRACK_TOKEN is not set. Set it via env or --token flag.")
		os.Exit(1)
	}

	c := api.NewClient(baseURL, token)
	c.Verbose = verbose
	return c
}
