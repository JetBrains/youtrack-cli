package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
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
			paginate, _ := cmd.Flags().GetBool("paginate")
			pageSize, _ := cmd.Flags().GetInt("page-size")

			path := args[0]
			if fields != "" {
				sep := "?"
				if strings.Contains(path, "?") {
					sep = "&"
				}
				path += sep + "fields=" + fields
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

			method = strings.ToUpper(method)
			if paginate && method != "GET" {
				return fmt.Errorf("--paginate requires --method GET")
			}
			var result any
			if paginate {
				result, err = rawPaginated(client, path, pageSize)
			} else {
				result, err = client.RawRequest(method, path, body)
			}
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
	cmd.Flags().String("fields", "", "Field projection string (added as fields=...)")
	cmd.Flags().Bool("paginate", false, "Fetch every page of a JSON array response")
	cmd.Flags().Int("page-size", 100, "Items per request with --paginate")
	return cmd
}

func rawPaginated(client *api.Client, path string, pageSize int) ([]any, error) {
	if pageSize <= 0 {
		return nil, fmt.Errorf("--page-size must be greater than zero")
	}
	var all []any
	for offset := 0; ; offset += pageSize {
		pagePath, err := withRawPage(path, pageSize, offset)
		if err != nil {
			return nil, err
		}
		result, err := client.RawRequest("GET", pagePath, nil)
		if err != nil {
			return nil, err
		}
		page, ok := result.([]any)
		if !ok {
			return nil, fmt.Errorf("--paginate requires a JSON array response, got %T", result)
		}
		all = append(all, page...)
		if len(page) < pageSize {
			return all, nil
		}
	}
}

func withRawPage(path string, pageSize, offset int) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("parse raw path: %w", err)
	}
	q := u.Query()
	q.Set("$top", strconv.Itoa(pageSize))
	q.Set("$skip", strconv.Itoa(offset))
	u.RawQuery = q.Encode()
	return u.String(), nil
}
