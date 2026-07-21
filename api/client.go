package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"time"
)

const envReadOnly = "YOUTRACK_RO"

const defaultTimeout = 30 * time.Second

// maxErrorBodySize caps error-body reads so a misbehaving server can't balloon client memory.
const maxErrorBodySize = 1 << 20 // 1 MiB

// UserAgent is sent on every request; internal.GetClient sets it to the build version.
var UserAgent = "youtrack-cli"

// Client sends authenticated requests to a YouTrack instance.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	Verbose    bool
	Fields     string // when non-empty, extends the fields= query param on GET requests
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		HTTPClient: &http.Client{Timeout: defaultTimeout},
	}
}

func (c *Client) doRequest(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.Verbose {
		_, _ = fmt.Fprintf(os.Stderr, "> %s %s%s\n", method, c.BaseURL, path)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if c.Verbose {
		_, _ = fmt.Fprintf(os.Stderr, "< %d\n", resp.StatusCode)
	}
	return resp, nil
}

// handleResponse reads the response body, closes it, checks for errors,
// and optionally unmarshals JSON into result.
func handleResponse(resp *http.Response, result any) error {
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
		if err != nil {
			return err
		}
		var apiErr struct {
			ErrorDescription string `json:"error_description"`
			ErrorDevMessage  string `json:"error_developer_message"`
			Error            string `json:"error"`
			Message          string `json:"message"`
		}
		if jsonErr := json.Unmarshal(data, &apiErr); jsonErr == nil {
			for _, detail := range []string{
				apiErr.ErrorDescription,
				apiErr.ErrorDevMessage,
				apiErr.Error,
				apiErr.Message,
			} {
				if detail != "" {
					return &APIError{StatusCode: resp.StatusCode, Message: detail}
				}
			}
		}
		return &APIError{StatusCode: resp.StatusCode, Message: string(data)}
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if result == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, result); err != nil {
		return unexpectedResponseError(resp, err)
	}
	return nil
}

func unexpectedResponseError(resp *http.Response, err error) error {
	return &UnexpectedResponseError{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Err:         err,
	}
}

func (c *Client) Get(path string, result any) error {
	if c.Fields != "" {
		path = mergeFieldsParam(path, c.Fields)
	}
	resp, err := c.doRequest("GET", path, nil, "")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return handleResponse(resp, result)
}

// mergeFieldsParam adds to a URL's fields= param rather than replacing it, so a user projection can't drop fields a call needs internally.
func mergeFieldsParam(path, fields string) string {
	idx := strings.Index(path, "fields=")
	if idx < 0 {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		return path + sep + "fields=" + fields
	}
	valStart := idx + len("fields=")
	end := strings.IndexByte(path[valStart:], '&')
	if end < 0 {
		return path + "," + fields
	}
	insertAt := valStart + end
	return path[:insertAt] + "," + fields + path[insertAt:]
}

func (c *Client) Post(path string, body, result any) error {
	if os.Getenv(envReadOnly) == "1" {
		return fmt.Errorf("read-only mode is enabled (YOUTRACK_RO=1): write operations are blocked")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.doRequest("POST", path, bytes.NewReader(data), "application/json")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return handleResponse(resp, result)
}

func (c *Client) Put(path string, body, result any) error {
	if os.Getenv(envReadOnly) == "1" {
		return fmt.Errorf("read-only mode is enabled (YOUTRACK_RO=1): write operations are blocked")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.doRequest("PUT", path, bytes.NewReader(data), "application/json")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return handleResponse(resp, result)
}

func (c *Client) Delete(path string) error {
	if os.Getenv(envReadOnly) == "1" {
		return fmt.Errorf("read-only mode is enabled (YOUTRACK_RO=1): write operations are blocked")
	}
	resp, err := c.doRequest("DELETE", path, nil, "")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return handleResponse(resp, nil)
}

// Multipart uploads a file as a multipart/form-data attachment.
func (c *Client) Multipart(path, filePath, fileName, mimeType string, result any) error {
	if os.Getenv(envReadOnly) == "1" {
		return fmt.Errorf("read-only mode is enabled (YOUTRACK_RO=1): write operations are blocked")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="upload"; filename="%s"`, fileName)) //nolint:gocritic // %q uses Go escaping, not RFC 6266 HTTP header quoting
	h.Set("Content-Type", mimeType)

	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}
	if _, err = io.Copy(part, f); err != nil {
		return err
	}
	_ = w.Close()

	resp, err := c.doRequest("POST", path, &buf, w.FormDataContentType())
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return handleResponse(resp, result)
}
