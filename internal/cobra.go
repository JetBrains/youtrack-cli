package internal

import (
	"fmt"

	"github.com/spf13/cobra"
)

// MustMarkRequired marks the given flags as required, panicking if any flag name is invalid.
func MustMarkRequired(cmd *cobra.Command, names ...string) {
	for _, n := range names {
		if err := cmd.MarkFlagRequired(n); err != nil {
			panic(err)
		}
	}
}

// AddToGroup registers cmds on parent under the help group with the given ID.
func AddToGroup(parent *cobra.Command, groupID string, cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.GroupID = groupID
		parent.AddCommand(c)
	}
}

// RequireSubcommand makes cmd a pure container: running it bare or with an unknown subcommand exits non-zero instead of printing help.
func RequireSubcommand(cmd *cobra.Command) {
	// Args alone is ignored — cobra skips Args validation on commands with no RunE — so set both.
	cmd.Args = cobra.NoArgs
	cmd.RunE = func(c *cobra.Command, _ []string) error {
		return fmt.Errorf("requires a subcommand; run '%s --help'", c.CommandPath())
	}
}
