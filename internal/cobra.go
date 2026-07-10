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

// RequireSubcommand marks cmd as a pure container command: it does nothing on its
// own and must be invoked with one of its subcommands. Running it bare, or with an
// unrecognized subcommand name, exits non-zero instead of printing help and exiting 0.
//
// Setting Args alone is not enough — cobra treats a command with no Run/RunE as
// non-runnable and skips Args validation entirely, so both must be set together.
func RequireSubcommand(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
	cmd.RunE = func(c *cobra.Command, _ []string) error {
		return fmt.Errorf("requires a subcommand; run '%s --help'", c.CommandPath())
	}
}
