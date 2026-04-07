package internal

import "github.com/spf13/cobra"

// MustMarkRequired marks the given flags as required, panicking if any flag name is invalid.
func MustMarkRequired(cmd *cobra.Command, names ...string) {
	for _, n := range names {
		if err := cmd.MarkFlagRequired(n); err != nil {
			panic(err)
		}
	}
}
