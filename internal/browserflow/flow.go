// Package browserflow runs a browser-driven authorization flow over a local HTTP listener.
package browserflow

import (
	"bufio"
	"cmp"
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed templates/callback.html
var callbackHTML string
var callbackTmpl = template.Must(template.New("callback").Parse(callbackHTML))

//go:embed logo/youtrack.svg
var logoSVG string

// openBrowser is overridable in tests to avoid spawning real browser tabs.
var openBrowser = openSystemBrowser

// DefaultTimeout is the wall-clock limit on a flow.
const DefaultTimeout = 5 * time.Minute

// Options configure a single browser-driven flow run.
type Options struct {
	Listener net.Listener  // ownership transfers to Run; closed on return. Nil means paste-only (loopback unavailable).
	State    string        // expected state value; required
	OpenURL  string        // URL the browser is sent to; required
	NoOpen   bool          // do not open a browser — the caller has printed the URL
	Timeout  time.Duration // 0 → DefaultTimeout
	Warn     func(format string, args ...any)

	// Paste, when set, lets the user paste the browser's final redirect URL instead of relying on the loopback
	// callback (SSH sessions, WSL, no display). It is read when NoOpen is set, the browser cannot be opened, or Listener is nil.
	Paste *bufio.Reader
	// RedirectURI is the redirect_uri sent in the authorization request; pasted URLs must target it. Required with Paste.
	RedirectURI string
	// PastePrompt is shown (via Warn) once paste input is accepted.
	PastePrompt string
}

// Result holds the validated callback code; state is checked in the callback handler and not exposed.
type Result struct {
	Code string
}

// ListenLoopback binds to 127.0.0.1 only — never a wildcard address — so no other host can reach the callback. Port 0 picks a free ephemeral port.
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

// errStateMismatch means a callback did not carry this flow's state (CSRF / stale login attempt).
var errStateMismatch = errors.New("state mismatch")

// errPastedStateMismatch is errStateMismatch for a pasted URL; it aborts the flow since the user pasted the wrong attempt.
var errPastedStateMismatch = errors.New("the pasted URL belongs to a different sign-in attempt (state mismatch) — re-run the command and paste the URL from this attempt")

// AuthorizationError is an error=... callback from the authorization server.
type AuthorizationError struct {
	Code string
}

func (e *AuthorizationError) Error() string {
	switch e.Code {
	case "access_denied":
		return "authorization denied: access_denied — the sign-in was declined in the browser"
	case "unauthorized_client", "invalid_request", "invalid_scope", "unsupported_response_type":
		return fmt.Sprintf("authorization failed: %s — check that the OAuth client is registered in YouTrack/Hub with Authorization Code and PKCE enabled and a loopback redirect URI (http://127.0.0.1)", e.Code)
	default:
		return "authorization failed: " + e.Code
	}
}

// validateCallback checks the query of a callback (loopback request or pasted redirect URL) and returns the authorization code.
// The state comparison is constant-time and happens before anything else is trusted; duplicated parameters are rejected as malformed.
func validateCallback(q url.Values, state string) (string, error) {
	for _, name := range []string{"state", "code", "error"} {
		if len(q[name]) > 1 {
			return "", fmt.Errorf("callback repeats the %q parameter", name)
		}
	}
	if subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1 {
		return "", errStateMismatch
	}
	if e := q.Get("error"); e != "" {
		return "", &AuthorizationError{Code: sanitizeErrorCode(e)}
	}
	code := q.Get("code")
	if code == "" {
		return "", errors.New("callback received without code")
	}
	return code, nil
}

// sanitizeErrorCode keeps only the RFC 6749 error-code alphabet, so a crafted callback cannot inject terminal escapes into the error message.
func sanitizeErrorCode(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0x20 && r <= 0x7e && r != '"' && r != '\\' {
			b.WriteRune(r)
		}
		if b.Len() >= 64 {
			break
		}
	}
	return b.String()
}

// ParseRedirectURL validates a redirect URL pasted by the user: it must target redirectURI (same scheme, host, port, and path) and carry this flow's state.
// A bare code is refused because without the state there is no proof that it belongs to this login attempt.
func ParseRedirectURL(raw, redirectURI, state string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() {
		return "", errors.New("paste the full URL from the browser address bar (it starts with " + redirectURI + ")")
	}
	want, err := url.Parse(redirectURI)
	if err != nil {
		return "", fmt.Errorf("invalid redirect uri %q", redirectURI)
	}
	if !strings.EqualFold(u.Scheme, want.Scheme) || !strings.EqualFold(u.Host, want.Host) || strings.TrimSuffix(u.Path, "/") != strings.TrimSuffix(want.Path, "/") {
		return "", errors.New("the pasted URL does not point to " + redirectURI)
	}
	code, err := validateCallback(u.Query(), state)
	if errors.Is(err, errStateMismatch) {
		return "", errPastedStateMismatch
	}
	return code, err
}

// Run hosts opts.Listener, opens opts.OpenURL, waits for the callback at "/" only (path-less, matching Hub's loopback redirect URIs), validates state in constant time, and closes the listener on return.
// When opts.Paste is set it also accepts the redirect URL pasted into the terminal. Exactly one valid callback is accepted.
func Run(ctx context.Context, opts *Options) (*Result, error) {
	if opts.Listener != nil {
		defer func() { _ = opts.Listener.Close() }() // safe even after srv.Shutdown closes it
	}
	if opts.Listener == nil && opts.Paste == nil {
		return nil, errors.New("browserflow: listener or paste input is required")
	}
	if opts.State == "" {
		return nil, errors.New("browserflow: state is required")
	}
	if opts.OpenURL == "" {
		return nil, errors.New("browserflow: open url is required")
	}
	if opts.Paste != nil && opts.RedirectURI == "" {
		return nil, errors.New("browserflow: redirect uri is required with paste input")
	}

	timeout := cmp.Or(opts.Timeout, DefaultTimeout)
	warn := opts.Warn
	if warn == nil {
		warn = func(string, ...any) {}
	}

	type outcome struct {
		code string
		err  error
	}
	resultCh := make(chan outcome, 2)
	// consumed makes the state single-use: after the first valid callback, any replay is refused.
	var consumed atomic.Bool

	serverErr := make(chan error, 1)
	if opts.Listener != nil {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Cache-Control", "no-store")
			// The URL of this page carries the authorization code; never leak it through a Referer header.
			w.Header().Set("Referrer-Policy", "no-referrer")
			// The page is static and self-contained: no remote resources, only its own inline style and close-timer script.
			w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'")

			// Validate state in the handler so a forged or stray request can't claim the result slot; real errors echo state (RFC 6749).
			// The query string is never logged.
			code, err := validateCallback(r.URL.Query(), opts.State)
			if errors.Is(err, errStateMismatch) || (err != nil && !isAuthorizationError(err)) {
				w.WriteHeader(http.StatusBadRequest)
				_ = callbackTmpl.Execute(w, callbackData{Error: "invalid or expired request", Logo: template.HTML(logoSVG)})
				return
			}
			if !consumed.CompareAndSwap(false, true) {
				w.WriteHeader(http.StatusBadRequest)
				_ = callbackTmpl.Execute(w, callbackData{Error: "this sign-in request was already used", Logo: template.HTML(logoSVG)})
				return
			}
			data := callbackData{Logo: template.HTML(logoSVG)}
			var authErr *AuthorizationError
			if errors.As(err, &authErr) {
				data.Error = authErr.Code
				w.WriteHeader(http.StatusBadRequest)
			}
			_ = callbackTmpl.Execute(w, data)
			resultCh <- outcome{code: code, err: err}
		})

		// ErrorLog is discarded so malformed requests are never echoed to the terminal.
		srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second, ErrorLog: log.New(io.Discard, "", 0)}
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
	}

	var pasteOnce sync.Once
	startPaste := func() {
		if opts.Paste == nil {
			return
		}
		pasteOnce.Do(func() {
			if opts.PastePrompt != "" {
				warn("%s", opts.PastePrompt)
			}
			go func() {
				for {
					line, err := opts.Paste.ReadString('\n')
					if strings.TrimSpace(line) != "" {
						code, perr := ParseRedirectURL(line, opts.RedirectURI, opts.State)
						switch {
						case perr == nil || isAuthorizationError(perr):
							if consumed.CompareAndSwap(false, true) {
								resultCh <- outcome{code: code, err: perr}
							}
							return
						case errors.Is(perr, errPastedStateMismatch):
							resultCh <- outcome{err: perr}
							return
						default:
							warn("%v", perr)
						}
					}
					if err != nil {
						if opts.Listener == nil {
							resultCh <- outcome{err: errors.New("no redirect URL was pasted — re-run the command and paste the URL from the browser address bar")}
						}
						return
					}
				}
			}()
		})
	}

	switch {
	case opts.NoOpen || opts.Listener == nil:
		startPaste()
	case !BrowserAvailable():
		warn("No browser detected (headless or remote session).")
		startPaste()
	default:
		if err := openBrowser(opts.OpenURL); err != nil {
			warn("Could not open browser automatically: %v", err)
			startPaste()
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}
		return &Result{Code: res.code}, nil
	case err := <-serverErr:
		return nil, fmt.Errorf("local callback server failed: %w", err)
	case <-timer.C:
		return nil, fmt.Errorf("sign-in timed out after %v without a browser callback — re-run the command (use --timeout to wait longer)", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func isAuthorizationError(err error) bool {
	var authErr *AuthorizationError
	return errors.As(err, &authErr)
}

// BrowserAvailable reports whether a local graphical browser can plausibly be opened: false over SSH and on Linux/BSD without a display (WSL is allowed — wslview/xdg-open reach the Windows browser).
func BrowserAvailable() bool {
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return false
	}
	switch runtime.GOOS {
	case "darwin", "windows":
		return true
	default:
		return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("WSL_DISTRO_NAME") != ""
	}
}

type callbackData struct {
	Error string
	Logo  template.HTML
}

// openSystemBrowser opens the given URL in the system's default browser.
// The URL is passed as a single argv element — never through a shell — so its characters cannot be interpreted as commands.
func openSystemBrowser(targetURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", targetURL).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL).Start()
	default:
		for _, opener := range []string{"xdg-open", "wslview"} {
			if path, err := exec.LookPath(opener); err == nil {
				return exec.Command(path, targetURL).Start()
			}
		}
		return errors.New("no browser opener found (install xdg-utils, or use --no-browser)")
	}
}
