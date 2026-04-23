package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"

	"github.com/JetBrains/youtrack-cli/api"
)

type jsonErrorEnvelope struct {
	Error jsonErrorBody `json:"error"`
}

type jsonErrorBody struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// RenderError prints err to the appropriate stream based on output mode.
//
// JSON mode (jsonMode=true): writes {"error":{...}} to jsonOut (stdout).
// Agents receive structured errors on stdout alongside normal JSON output.
//
// Human mode: writes "✗ Error: <msg>" to w (stderr).
// Appends "  Tip: <hint>" unless quiet=true. The error line itself is never suppressed.
func RenderError(w, jsonOut io.Writer, err error, jsonMode, quiet bool) {
	tip := tipFor(err)
	if jsonMode {
		env := jsonErrorEnvelope{Error: jsonErrorBody{
			Code:       errorCode(err),
			Message:    err.Error(),
			Suggestion: tip,
		}}
		enc := json.NewEncoder(jsonOut)
		enc.SetIndent("", "  ")
		_ = enc.Encode(env)
		return
	}
	fmt.Fprintf(w, "%s Error: %s\n", color.New(color.FgRed).Sprint("✗"), err)
	if tip != "" && !quiet {
		fmt.Fprintf(w, "  Tip: %s\n", tip)
	}
}

func tipFor(err error) string {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			return "Run 'youtrack auth get-token' to re-authenticate"
		case 403:
			return "Check your permissions for this operation"
		case 404:
			return "Resource not found — verify the ID"
		case 503:
			return "YouTrack unreachable — check YOUTRACK_BASE_URL"
		}
	}
	msg := err.Error()
	if strings.Contains(msg, "YOUTRACK_BASE_URL") {
		return "Set it via env var or --base-url flag"
	}
	if strings.Contains(msg, "YOUTRACK_TOKEN") {
		return "Run 'youtrack auth get-token' to authenticate"
	}
	return ""
}

func errorCode(err error) string {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			return "unauthorized"
		case 403:
			return "forbidden"
		case 404:
			return "not_found"
		case 503:
			return "unavailable"
		default:
			return "api_error"
		}
	}
	return "internal_error"
}
