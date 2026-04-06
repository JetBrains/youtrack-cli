package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newRawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "raw <path>",
		Short: "Make an authenticated API request to any YouTrack endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			method, _ := cmd.Flags().GetString("method")
			bodyStr, _ := cmd.Flags().GetString("body")
			fields, _ := cmd.Flags().GetString("fields")

			path := args[0]
			if fields != "" {
				sep := "?"
				if strings.Contains(path, "?") {
					sep = "&"
				}
				path += sep + "$fields=" + fields
			}

			var body map[string]any
			if bodyStr != "" {
				if err := json.Unmarshal([]byte(bodyStr), &body); err != nil {
					return fmt.Errorf("invalid JSON body: %w", err)
				}
			}

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			result, err := client.RawRequest(strings.ToUpper(method), path, body)
			if err != nil {
				return err
			}
			if result == nil {
				return nil
			}
			return output.Print(cmd, result)
		},
	}
	cmd.Flags().String("method", "GET", "HTTP method (GET, POST, PUT, DELETE)")
	cmd.Flags().String("body", "", "JSON request body")
	cmd.Flags().String("fields", "", "Field projection string (added as $fields=...)")
	return cmd
}
