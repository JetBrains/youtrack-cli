package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// DryRunError carries a write that --dry-run stopped before it was sent.
// The command ends at the first write, so steps that depend on its result are not shown.
type DryRunError struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Body   any    `json:"body,omitempty"`
}

func (e *DryRunError) Error() string {
	return fmt.Sprintf("dry run: %s %s not sent", e.Method, e.URL)
}

func newDryRunError(method, url string, body any) error {
	return &DryRunError{Method: method, URL: url, Body: body}
}

// RenderPreview writes a redacted preview of the stopped write in CLI JSON or plain format.
func (req *DryRunError) RenderPreview(w io.Writer, jsonMode bool) error {
	// Copy through JSON so nested structs and maps receive identical redaction without
	// changing the original request used by the caller.
	data, err := json.Marshal(req.Body)
	if err != nil {
		return err
	}
	var body any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return err
	}
	redactDryRunSecrets(body)
	safe := *req
	safe.Body = body
	req = &safe
	if jsonMode {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(struct {
			DryRun bool `json:"dryRun"`
			*DryRunError
		}{true, req})
	}
	if _, err := fmt.Fprintf(w, "%s %s\n", req.Method, req.URL); err != nil {
		return err
	}
	if req.Body == nil {
		return nil
	}
	encodedBody, err := json.MarshalIndent(req.Body, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", encodedBody)
	return err
}

func redactDryRunSecrets(value any) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
			if strings.Contains(normalized, "password") || strings.Contains(normalized, "token") || strings.Contains(normalized, "secret") {
				v[key] = "***"
			} else {
				redactDryRunSecrets(child)
			}
		}
	case []any:
		for _, child := range v {
			redactDryRunSecrets(child)
		}
	}
}
