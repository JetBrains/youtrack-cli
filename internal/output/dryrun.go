package output

import (
	"github.com/JetBrains/youtrack-cli/api"
	"io"
)

// RenderDryRun renders a stopped write without exposing known request secrets.
func RenderDryRun(w io.Writer, req *api.DryRunError, jsonMode bool) error {
	return req.RenderPreview(w, jsonMode)
}
