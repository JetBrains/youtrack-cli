// Package browserflow runs a browser-driven authorization flow over a local HTTP listener.
package browserflow

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

//go:embed templates/callback.html
var callbackHTML string
var callbackTmpl = template.Must(template.New("callback").Parse(callbackHTML))

// openBrowser is overridable in tests to avoid spawning real browser tabs.
var openBrowser = openSystemBrowser

// DefaultTimeout is the wall-clock limit on a flow.
const DefaultTimeout = 5 * time.Minute

// Options configure a single browser-driven flow run.
type Options struct {
	Listener net.Listener  // ownership transfers to Run; closed on return
	State    string        // expected state value; required
	OpenURL  string        // URL the browser is sent to; required
	NoOpen   bool          // do not open a browser — the caller has printed the URL
	Timeout  time.Duration // 0 → DefaultTimeout
	Warn     func(format string, args ...any)
}

// Result holds the validated callback code; state is checked in the callback handler and not exposed.
type Result struct {
	Code string
}

// ListenLoopback binds to 127.0.0.1. Port 0 picks a free ephemeral port.
func ListenLoopback(port int) (net.Listener, error) {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		if port != 0 {
			return nil, fmt.Errorf("port %d is in use — omit --port to pick one automatically", port)
		}
		return nil, fmt.Errorf("listening on loopback: %w", err)
	}
	return l, nil
}

// GenerateState returns a 16-byte random URL-safe string suitable for OAuth state / CSRF protection.
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Run hosts opts.Listener, opens opts.OpenURL, waits for the callback on the
// bare root path, validates state in constant time, and closes the listener on
// return. The callback is served at "/" only (no path) because Hub's registered
// loopback redirect URIs are path-less.
func Run(ctx context.Context, opts Options) (*Result, error) {
	if opts.Listener == nil {
		return nil, errors.New("browserflow: listener is required")
	}
	defer func() { _ = opts.Listener.Close() }() // safe even after srv.Shutdown closes it
	if opts.State == "" {
		return nil, errors.New("browserflow: state is required")
	}
	if opts.OpenURL == "" {
		return nil, errors.New("browserflow: open url is required")
	}

	timeout := cmp.Or(opts.Timeout, DefaultTimeout)
	warn := opts.Warn
	if warn == nil {
		warn = func(string, ...any) {}
	}

	type rawResult struct {
		code, state, errMsg string
	}
	resultCh := make(chan rawResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		raw := rawResult{code: q.Get("code"), state: q.Get("state"), errMsg: q.Get("error")}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'")

		// Validate state in the handler so a forged or stray request can't claim
		// the result slot; real errors echo state (RFC 6749).
		if subtle.ConstantTimeCompare([]byte(raw.state), []byte(opts.State)) != 1 {
			w.WriteHeader(http.StatusBadRequest)
			_ = callbackTmpl.Execute(w, callbackData{Error: "invalid or expired request"})
			return
		}

		data := callbackData{}
		if raw.errMsg != "" {
			data.Error = raw.errMsg
			w.WriteHeader(http.StatusBadRequest)
		}
		_ = callbackTmpl.Execute(w, data)

		select {
		case resultCh <- raw:
		default:
		}
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(opts.Listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if !opts.NoOpen {
		if err := openBrowser(opts.OpenURL); err != nil {
			warn("Could not open browser automatically: %v", err)
		}
	}

	select {
	case raw := <-resultCh:
		// state was validated by the handler before raw reached this channel.
		if raw.errMsg != "" {
			return nil, fmt.Errorf("authorization denied: %s", raw.errMsg)
		}
		if raw.code == "" {
			return nil, errors.New("callback received without code")
		}
		return &Result{Code: raw.code}, nil
	case err := <-serverErr:
		return nil, fmt.Errorf("local callback server failed: %w", err)
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for browser callback (%v) — re-run or use the paste-token flow (youtrack auth login)", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type callbackData struct {
	Error string
}

// openSystemBrowser opens the given URL in the system's default browser.
func openSystemBrowser(targetURL string) error {
	var cmdName string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmdName = "open"
		args = []string{targetURL}
	case "windows":
		cmdName = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", targetURL}
	default:
		cmdName = "xdg-open"
		args = []string{targetURL}
	}
	return exec.Command(cmdName, args...).Start()
}
