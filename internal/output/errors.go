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

// RenderError prints err to w (stderr) — errors never land on stdout,
// which is reserved for machine-readable data.
//
// JSON mode (jsonMode=true): writes a structured {"error":{...}} envelope.
//
// Human mode: writes "✗ Error: <msg>".
// Appends "  Tip: <hint>" unless quiet=true. The error line itself is never suppressed.
func RenderError(w io.Writer, err error, jsonMode, quiet bool) {
	tip := tipFor(err)
	if jsonMode {
		env := jsonErrorEnvelope{Error: jsonErrorBody{
			Code:       errorCode(err),
			Message:    err.Error(),
			Suggestion: tip,
		}}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(env)
		return
	}
	_, _ = fmt.Fprintf(w, "%s Error: %s\n", color.New(color.FgRed).Sprint("✗"), err)
	if tip != "" && !quiet {
		_, _ = fmt.Fprintf(w, "  Tip: %s\n", tip)
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
	var unexpectedErr *api.UnexpectedResponseError
	if errors.As(err, &unexpectedErr) {
		return "Run 'youtrack auth status' or retry with '--verbose' to confirm which configured instance is being used"
	}
	msg := err.Error()
	if strings.Contains(msg, "YOUTRACK_BASE_URL") {
		return "Set it via env var or --base-url flag"
	}
	if strings.Contains(msg, "YOUTRACK_TOKEN") {
		return "Run 'youtrack auth get-token' to authenticate"
	}
	if strings.Contains(msg, "alias is required") {
		return "Run 'youtrack instance list' to see available instances"
	}
	if strings.Contains(msg, "no active instance") {
		return "Run 'youtrack instance use <alias>' to set one, or pass the alias directly"
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
	var unexpectedErr *api.UnexpectedResponseError
	if errors.As(err, &unexpectedErr) {
		return "unexpected_response"
	}
	return "internal_error"
}
