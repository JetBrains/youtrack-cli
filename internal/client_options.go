package internal

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
)

// configureClient gives each command its own settings while sharing the transport.
func configureClient(cmd *cobra.Command, source *api.Client) *api.Client {
	c := *source
	c.WithContext(cmd.Context())
	flags := cmd.Root().PersistentFlags()
	if flags.Changed("verbose") {
		c.Verbose, _ = flags.GetBool("verbose")
	}
	fields, _ := flags.GetString("fields")
	if fields = expandFields(fields); fields != "" {
		if c.Fields != "" {
			c.Fields += ","
		}
		c.Fields += fields
	}
	if c.UserAgent == "" || c.UserAgent == "youtrack-cli" {
		c.UserAgent = "youtrack-cli/" + Version
	}
	dryRun, _ := flags.GetBool("dry-run")
	mute, _ := flags.GetBool("mute-notifications")
	c.SetWriteOptions(dryRun, mute, cmd.ErrOrStderr())
	return &c
}
