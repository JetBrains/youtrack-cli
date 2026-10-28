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
	"net/url"
	"os"
	"strings"
	"time"
)

const envReadOnly = "YOUTRACK_RO"

const defaultTimeout = 30 * time.Second

// maxErrorBodySize caps error-body reads so a misbehaving server can't balloon client memory.
const maxErrorBodySize = 1 << 20 // 1 MiB

// UserAgent is the default copied into each new Client.
//
// Deprecated: set Client.UserAgent to customize individual clients.
var UserAgent = "youtrack-cli"

// Client sends authenticated requests to a YouTrack instance.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	UserAgent  string
	Verbose    bool
	Fields     string // when non-empty, extends the fields= query param on GET requests
	// MuteNotifications suppresses notifications on API writes.
	MuteNotifications bool
	// DryRun describes writes instead of sending them.
	DryRun bool
	Warn   io.Writer
	ctx    context.Context
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		HTTPClient: &http.Client{Timeout: defaultTimeout, CheckRedirect: sameOriginRedirect},
		UserAgent:  UserAgent,
		ctx:        context.Background(),
	}
}

// sameOriginRedirect drops the Authorization header when a redirect leaves the origin (scheme, host, default-normalized port) of the first request.
// net/http keeps it on a same-host http downgrade and on a redirect to a subdomain.
// An http → https upgrade on the same hostname keeps it, as net/http does, so an http:// instance URL behind a TLS-redirecting proxy still works.
func sameOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if !redirectKeepsAuth(via[0].URL, req.URL) {
		req.Header.Del("Authorization")
	}
	return nil
}

// redirectKeepsAuth reports whether a redirect from first to next may carry the Authorization header.
func redirectKeepsAuth(first, next *url.URL) bool {
	if !strings.EqualFold(first.Hostname(), next.Hostname()) {
		return false
	}
	from, to := strings.ToLower(first.Scheme), strings.ToLower(next.Scheme)
	if from == "http" && to == "https" {
		return true
	}
	return from == to && defaultPort(first) == defaultPort(next)
}

// defaultPort returns the explicit port of u, or the default port of its scheme, so "https://x" and "https://x:443" compare equal.
func defaultPort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	}
	return ""
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

// acceptJSON is sent by requests that decode the response; acceptAny by requests that pass it through unchanged.
const (
	acceptJSON = "application/json"
	acceptAny  = "application/json, */*;q=0.8"
)

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, contentType, accept string) (*http.Response, error) {
	if !validRequestPath(path) {
		return nil, fmt.Errorf("request path must begin with a single slash")
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", c.UserAgent)
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
	resp, err := c.doRequest(c.context(), method, path, body, contentType, acceptJSON)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return handleResponse(resp, result)
}

// stream performs one request and copies a successful response body to w without decoding it.
// The overall client timeout is lifted so large bodies can finish; instead the request is
// canceled once the server sends nothing for defaultTimeout.
func (c *Client) stream(method, path string, body io.Reader, contentType string, w io.Writer) error {
	ctx, cancel := context.WithCancel(c.context())
	defer cancel()
	idle := time.AfterFunc(defaultTimeout, cancel)
	defer idle.Stop()

	httpClient := *c.HTTPClient
	httpClient.Timeout = 0
	streaming := *c
	streaming.HTTPClient = &httpClient
	resp, err := streaming.doRequest(ctx, method, path, body, contentType, acceptAny)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return handleResponse(resp, nil)
	}
	_, err = io.Copy(w, &idleReader{r: resp.Body, idle: idle})
	return err
}

// idleReader pushes back the idle deadline every time the server delivers data.
type idleReader struct {
	r    io.Reader
	idle *time.Timer
}

func (r *idleReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.idle.Reset(defaultTimeout)
	}
	return n, err
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

// prepareWrite applies the same write policy to JSON, streaming, and multipart requests.
func (c *Client) prepareWrite(method, path string, body any) (string, error) {
	path = c.applyMute(method, path)
	if c.DryRun {
		return path, newDryRunError(method, c.BaseURL+path, body)
	}
	if readOnly() {
		return path, errReadOnly
	}
	return path, nil
}

func (c *Client) Post(path string, body, result any) error {
	path, err := c.prepareWrite("POST", path, body)
	if err != nil {
		return err
	}
	return c.sendJSON("POST", path, body, result)
}

// postQuery sends a POST to an endpoint that only computes a result and changes nothing
// (issue counts, command suggestions), so it stays available in read-only mode.
func (c *Client) postQuery(path string, body, result any) error {
	return c.sendJSON("POST", path, body, result)
}

func (c *Client) Put(path string, body, result any) error {
	path, err := c.prepareWrite("PUT", path, body)
	if err != nil {
		return err
	}
	return c.sendJSON("PUT", path, body, result)
}

func (c *Client) Delete(path string) error {
	path, err := c.prepareWrite("DELETE", path, nil)
	if err != nil {
		return err
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
	if c.DryRun {
		if err := checkReadableFile(filePath); err != nil {
			return err
		}
	}
	path, err := c.prepareWrite("POST", path, map[string]string{"file": filePath, "fileName": fileName, "contentType": mimeType})
	if err != nil {
		return err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="upload"; filename="%s"`, headerBreakEscaper.Replace(fileName))) //nolint:gocritic // %q uses Go escaping, not RFC 6266 HTTP header quoting
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

// headerBreakEscaper percent-encodes CR and LF in a file name, as browsers do, so a crafted name cannot end the
// Content-Disposition line and inject part headers. Quotes and backslashes stay as-is: YouTrack reads raw quotes
// leniently but treats a backslash as a path separator, so escaping them would corrupt the stored name.
var headerBreakEscaper = strings.NewReplacer("\r", "%0D", "\n", "%0A")

// checkReadableFile fails the way a real upload would on a missing, unreadable, or directory path, so a dry run doesn't report a request that cannot be sent.
func checkReadableFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("read %s: is a directory", filePath)
	}
	return nil
}

// SetWriteOptions configures CLI write controls on a command-scoped client copy.
func (c *Client) SetWriteOptions(dryRun, mute bool, warn io.Writer) {
	c.DryRun, c.MuteNotifications, c.Warn = dryRun, mute, warn
}
