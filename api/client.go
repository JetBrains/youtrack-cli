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
)

// Client sends authenticated requests to a YouTrack instance.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	Verbose    bool
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

func (c *Client) doRequest(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
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
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
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
	resp, err := c.doRequest("GET", path, nil, "")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return handleResponse(resp, result)
}

func (c *Client) Post(path string, body, result any) error {
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
	resp, err := c.doRequest("DELETE", path, nil, "")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return handleResponse(resp, nil)
}

// Multipart uploads a file as a multipart/form-data attachment.
func (c *Client) Multipart(path, filePath, fileName, mimeType string, result any) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="attachment"; filename="%s"`, fileName)) //nolint:gocritic // %q uses Go escaping, not RFC 6266 HTTP header quoting
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
