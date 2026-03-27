package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"youtrack-cli/api"
)

// getClient builds an API client from global flags and env vars.
// Returns an error if base URL or token are missing.
func getClient(cmd *cobra.Command) (*api.Client, error) {
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
		return nil, fmt.Errorf("YOUTRACK_BASE_URL is not set. Set it via env or --base-url flag")
	}
	if token == "" {
		return nil, fmt.Errorf("YOUTRACK_TOKEN is not set. Set it via env or --token flag")
	}

	c := api.NewClient(baseURL, token)
	c.Verbose = verbose
	return c, nil
}
