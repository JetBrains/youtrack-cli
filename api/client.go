package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	ctx        context.Context
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		HTTPClient: &http.Client{Timeout: defaultTimeout},
		ctx:        context.Background(),
	}
}

// WithContext binds requests and polling waits to the caller's lifecycle.
// A nil context is treated as context.Background.
func (c *Client) WithContext(ctx context.Context) *Client {
	if ctx == nil {
		ctx = context.Background()
	}
	c.ctx = ctx
	return c
}

func (c *Client) context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *Client) wait(delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-c.context().Done():
		return c.context().Err()
	}
}

func (c *Client) doRequest(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	if !validRequestPath(path) {
		return nil, fmt.Errorf("request path must begin with a single slash")
	}
	req, err := http.NewRequestWithContext(c.context(), method, c.BaseURL+path, body)
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

func validRequestPath(path string) bool {
	return strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//")
}

// send performs one request and is the only place that closes a response body.
func (c *Client) send(method, path string, body io.Reader, contentType string, result any) error {
	resp, err := c.doRequest(method, path, body, contentType)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return handleResponse(resp, result)
}

// handleResponse reads the response body, checks for errors,
// and optionally unmarshals JSON into result. The caller closes the body.
func handleResponse(resp *http.Response, result any) error {
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
	return c.send("GET", path, nil, "", result)
}

// mergeFieldsParam adds to a URL's fields= param rather than replacing it, so a user projection can't drop fields a call needs internally.
// Only a real fields= parameter matches, never the suffix of another name such as customFields=.
func mergeFieldsParam(path, fields string) string {
	base, query, hasQuery := strings.Cut(path, "?")
	if !hasQuery {
		return path + "?fields=" + fields
	}
	params := strings.Split(query, "&")
	for i, p := range params {
		if strings.HasPrefix(p, "fields=") {
			params[i] = p + "," + fields
			return base + "?" + strings.Join(params, "&")
		}
	}
	return path + "&fields=" + fields
}

// errReadOnly is returned by every write when YOUTRACK_RO=1.
var errReadOnly = errors.New("read-only mode is enabled (YOUTRACK_RO=1): write operations are blocked")

func readOnly() bool {
	return os.Getenv(envReadOnly) == "1"
}

func (c *Client) Post(path string, body, result any) error {
	if readOnly() {
		return errReadOnly
	}
	return c.sendJSON("POST", path, body, result)
}

// postQuery sends a POST to an endpoint that only computes a result and changes nothing
// (issue counts, command suggestions), so it stays available in read-only mode.
func (c *Client) postQuery(path string, body, result any) error {
	return c.sendJSON("POST", path, body, result)
}

func (c *Client) Put(path string, body, result any) error {
	if readOnly() {
		return errReadOnly
	}
	return c.sendJSON("PUT", path, body, result)
}

func (c *Client) Delete(path string) error {
	if readOnly() {
		return errReadOnly
	}
	return c.send("DELETE", path, nil, "", nil)
}

func (c *Client) sendJSON(method, path string, body, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.send(method, path, bytes.NewReader(data), "application/json", result)
}

// Multipart uploads a file as a multipart/form-data attachment.
func (c *Client) Multipart(path, filePath, fileName, mimeType string, result any) error {
	if readOnly() {
		return errReadOnly
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

	return c.send("POST", path, &buf, w.FormDataContentType(), result)
}

