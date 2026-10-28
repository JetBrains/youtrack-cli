package issue

import (
	"fmt"
	"io"
	"strings"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
	"github.com/spf13/cobra"
)

func newIssueAttachmentDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <issueId> [attachmentId]",
		Short: "Save an issue attachment to a file",
		Long: "Save an issue attachment to a file, or to stdout with --output -. Pick the attachment by ID or with --name.\n" +
			"The download link is signed and works without a token, so it is never printed.",
		Example: "  youtrack issue attachment download FOO-42 8-15 --output screenshot.png\n" +
			"  youtrack issue attachment download FOO-42 --name trace.log --output -",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			outPath, _ := cmd.Flags().GetString("output")
			if (len(args) == 2) == (name != "") {
				return fmt.Errorf("pass either an attachment ID or --name")
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			var attachment *api.IssueAttachment
			if len(args) == 2 {
				attachment, err = client.GetIssueAttachment(args[0], args[1])
			} else {
				attachments, listErr := client.GetIssueAttachments(args[0])
				if listErr != nil {
					return listErr
				}
				var found api.IssueAttachment
				found, err = attachmentByName(attachments, args[0], name)
				if err == nil {
					attachment = &found
				}
			}
			if err != nil {
				return err
			}
			if attachment.URL == "" {
				return fmt.Errorf("attachment %s has no download link", attachment.ID)
			}
			n, err := internal.WriteOutput(cmd, outPath, func(w io.Writer) error {
				return client.DownloadFile(attachment.URL, w)
			})
			if err != nil {
				return fmt.Errorf("download attachment %s: %w", attachment.ID, err)
			}
			if outPath != "-" {
				output.PrintSuccess(cmd, fmt.Sprintf("downloaded %s (%d bytes) to %s", attachment.Name, n, outPath))
			}
			return nil
		},
	}
	cmd.Flags().String("name", "", "Pick the attachment by file name instead of ID")
	cmd.Flags().StringP("output", "o", "", "File to write, or - for stdout (required)")
	internal.MustMarkRequired(cmd, "output")
	return cmd
}

func attachmentByName(attachments []api.IssueAttachment, issueID, name string) (api.IssueAttachment, error) {
	var matches []api.IssueAttachment
	for _, a := range attachments {
		if a.Name == name {
			matches = append(matches, a)
		}
	}
	switch len(matches) {
	case 0:
		return api.IssueAttachment{}, fmt.Errorf("issue %s has no attachment named %q", issueID, name)
	case 1:
		return matches[0], nil
	}
	ids := make([]string, len(matches))
	for i, a := range matches {
		ids[i] = a.ID
	}
	return api.IssueAttachment{}, fmt.Errorf("issue %s has %d attachments named %q (%s); pass an attachment ID instead",
		issueID, len(matches), name, strings.Join(ids, ", "))
}
