package cmd

import (
	"encoding/json"
	"fmt"
	"io"
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
			bodyFile, _ := cmd.Flags().GetString("body-file")
			outPath, _ := cmd.Flags().GetString("output")
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

			if cmd.Flags().Changed("body") && cmd.Flags().Changed("body-file") {
				return fmt.Errorf("--body and --body-file are mutually exclusive")
			}
			if paginate && outPath != "" {
				return fmt.Errorf("--paginate and --output are mutually exclusive")
			}
			if bodyFile != "" {
				b, err := internal.ReadInput(cmd, bodyFile)
				if err != nil {
					return fmt.Errorf("read body file: %w", err)
				}
				// PowerShell 5 writes UTF-8 files with a byte order mark, which is not valid JSON.
				bodyStr = strings.TrimPrefix(string(b), "\ufeff")
			}
			var body any
			if strings.TrimSpace(bodyStr) != "" {
				if !json.Valid([]byte(bodyStr)) {
					return fmt.Errorf("invalid JSON body")
				}
				body = json.RawMessage(bodyStr)
			}

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			method = strings.ToUpper(method)
			if paginate && method != "GET" {
				return fmt.Errorf("--paginate requires --method GET")
			}
			if outPath != "" {
				n, err := internal.WriteOutput(cmd, outPath, func(w io.Writer) error {
					return client.RawStream(method, path, body, w)
				})
				if err != nil {
					return err
				}
				if outPath != "-" {
					output.PrintSuccess(cmd, fmt.Sprintf("wrote %d bytes to %s", n, outPath))
				}
				return nil
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
			if method == "GET" && !paginate && mayBeTruncated(path, result) {
				output.PrintWarn(cmd, fmt.Sprintf("got %d items, the YouTrack default page size; the list may be cut short. Pass --paginate to fetch every page", youTrackDefaultTop))
			}
			return output.Print(cmd, result)
		},
	}
	cmd.Flags().String("method", "GET", "HTTP method (GET, POST, PUT, DELETE)")
	cmd.Flags().String("body", "", "JSON request body")
	cmd.Flags().String("body-file", "", "Read the JSON request body from a file, or from stdin with -")
	cmd.Flags().StringP("output", "o", "", "Write the response body as-is to a file, or to stdout with -, without parsing it as JSON")
	cmd.Flags().String("fields", "", "Field projection string (added as fields=...)")
	cmd.Flags().Bool("paginate", false, "Fetch every page of a JSON array response")
	cmd.Flags().Int("page-size", 100, "Items per request with --paginate")
	return cmd
}

// youTrackDefaultTop is how many items YouTrack returns from a collection when the request sets no $top.
const youTrackDefaultTop = 42

func mayBeTruncated(path string, result any) bool {
	items, ok := result.([]any)
	if !ok || len(items) != youTrackDefaultTop {
		return false
	}
	u, err := url.Parse(path)
	return err == nil && !u.Query().Has("$top")
}

func rawPaginated(client *api.Client, path string, pageSize int) ([]any, error) {
	if pageSize <= 0 {
		return nil, fmt.Errorf("--page-size must be greater than zero")
	}
	all := make([]any, 0)
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
