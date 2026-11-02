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

// RenderError writes err to w (stderr) — a {"error":{...}} envelope in JSON mode, "✗ Error: <msg>" otherwise; appends a Tip unless quiet.
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
	_, _ = fmt.Fprintf(w, "%s Error: %s\n", color.New(color.FgRed).Sprint("✗"), sanitizeTerminal(err.Error()))
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
			if strings.Contains(strings.ToLower(apiErr.Error()), "apply commands silently") {
				return "--mute-notifications needs the Apply Commands Silently permission in each affected project; ask a project admin, or retry without the flag only if notifications are acceptable"
			}
			return "Check your permissions for this operation"
		case 404:
			return "Resource not found — verify the ID"
		case 503:
			return "YouTrack unreachable — check YOUTRACK_BASE_URL"
		}
	}
	var unexpectedErr *api.UnexpectedResponseError
	if errors.As(err, &unexpectedErr) {
		if isBinaryContentType(unexpectedErr.ContentType) {
			return "The endpoint returned a file, not JSON — save it with 'youtrack raw <path> --output <file>'"
		}
		return "Run 'youtrack auth status' or retry with '--verbose' to confirm which configured instance is being used"
	}
	msg := err.Error()
	// The origin-binding refusal already says how to proceed and mentions YOUTRACK_TOKEN, so it must not get the re-auth tip below.
	if strings.Contains(msg, "refusing to send the saved token") {
		return ""
	}
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

// isBinaryContentType reports whether a response is a file rather than a misrouted page: an HTML or JSON body points at the wrong server instead.
func isBinaryContentType(contentType string) bool {
	ct := strings.ToLower(contentType)
	return ct != "" && !strings.Contains(ct, "json") && !strings.HasPrefix(ct, "text/html")
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
