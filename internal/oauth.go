package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal/config"
)

// refreshSkew refreshes a saved OAuth access token this long before it expires, absorbing clock skew and request latency.
const refreshSkew = 60 * time.Second

// refreshTimeout bounds a single refresh request so a hung Hub cannot stall every command.
const refreshTimeout = 30 * time.Second

// oauthSession supplies the access token of a saved OAuth profile, refreshing it transparently when it is about to expire.
// The mutex makes refresh single-flight within a process; config.Lock serializes it across concurrent CLI processes.
type oauthSession struct {
	mu      sync.Mutex
	alias   string
	inst    config.Instance
	verbose io.Writer // nil unless --verbose
	now     func() time.Time
	skew    time.Duration // refreshSkew, or half the lifetime of a shorter-lived token so it is not refreshed on every request
}

func newOAuthSession(alias string, inst *config.Instance, verbose io.Writer) *oauthSession {
	return &oauthSession{alias: alias, inst: *inst, verbose: verbose, now: time.Now, skew: refreshSkew}
}

// canRefresh reports whether inst is an OAuth session with everything needed to refresh it.
func canRefresh(inst *config.Instance) bool {
	return inst.AuthMethod == config.AuthMethodOAuth && inst.RefreshToken != "" && inst.TokenEndpoint != "" && inst.ClientID != ""
}

// Token returns a valid access token, refreshing (and persisting the result) when it is expired or within refreshSkew of expiry.
func (s *oauthSession) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !api.TokenNeedsRefresh(s.inst.TokenExpiresAt, s.now(), s.skew) {
		return s.inst.Token, nil
	}

	unlock, err := config.Lock(ctx)
	if err != nil {
		return "", fmt.Errorf("refreshing the OAuth session for %q: %w", s.alias, err)
	}
	defer unlock()

	// Another process may have refreshed while we waited for the lock; its rotated refresh token supersedes ours.
	// A new sign-in (another SessionID) is not a refresh: adopting its tokens would switch accounts in the middle of a command.
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	cur, ok := cfg.Instances[s.alias]
	if !ok || !sameOrigin(cur.URL, s.inst.URL) || !sameOrigin(cur.TokenURL(), s.inst.URL) || cur.AuthMethod != config.AuthMethodOAuth || cur.SessionID != s.inst.SessionID {
		return "", fmt.Errorf("the saved session for %q was removed or replaced by another youtrack process — re-run the command", s.alias)
	}
	if cur.Token != "" && !api.TokenNeedsRefresh(cur.TokenExpiresAt, s.now(), s.skew) {
		s.inst = cur
		return cur.Token, nil
	}
	if !canRefresh(&cur) {
		return "", sessionExpiredError(s.alias)
	}
	if !cur.OAuthEndpointsTrusted() {
		return "", fmt.Errorf("refusing to send the refresh token for %q to %s: the endpoint is not the one saved with the token in the OS keyring — sign in again: youtrack auth login --session %s", s.alias, cur.TokenEndpoint, s.alias)
	}

	requestedAt := s.now()
	if s.verbose != nil {
		// Only the endpoint is logged; the form body carries the refresh token.
		_, _ = fmt.Fprintf(s.verbose, "> POST %s (refreshing OAuth access token for %q)\n", cur.TokenEndpoint, s.alias)
	}
	refreshCtx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()
	tok, err := api.RefreshOAuthTokenContext(refreshCtx, cur.TokenEndpoint, cur.ClientID, cur.RefreshToken)
	if err != nil {
		if s.verbose != nil {
			_, _ = fmt.Fprintf(s.verbose, "< refresh failed\n")
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if api.IsInvalidGrant(err) {
			// The refresh token is dead (expired, revoked, or rotated away); drop the stale credentials so they are not retried.
			cfg.ClearOAuthTokens(s.alias)
			cleanupErr := cfg.Save()
			if cleanupErr == nil {
				cleanupErr = config.PurgeKeyringToken(s.alias)
			}
			return "", errors.Join(sessionExpiredError(s.alias), cleanupErr)
		}
		return "", fmt.Errorf("refreshing the OAuth session for %q: %w", s.alias, err)
	}
	if s.verbose != nil {
		_, _ = fmt.Fprintf(s.verbose, "< refreshed\n")
	}

	refreshToken := cur.RefreshToken
	if tok.RefreshToken != "" {
		refreshToken = tok.RefreshToken // rotation: Hub may revoke the old refresh token, so it must be replaced
	}
	cfg.SetOAuthTokens(s.alias, tok.AccessToken, refreshToken, api.TokenExpiry(requestedAt, tok.ExpiresIn))
	// Save writes a temp file and renames it, so the rotated refresh token is replaced atomically.
	if err := cfg.Save(); err != nil {
		return "", fmt.Errorf("saving the refreshed OAuth session for %q: %w", s.alias, err)
	}
	s.inst = cfg.Instances[s.alias]
	s.skew = min(refreshSkew, time.Duration(tok.ExpiresIn)*time.Second/2)
	return tok.AccessToken, nil
}

func sessionExpiredError(alias string) error {
	return fmt.Errorf("the OAuth session for %q has expired or was revoked — sign in again: youtrack auth login --session %s", alias, alias)
}

// NewSavedInstanceClient builds a client for a saved instance, refreshing its token on demand when it is an OAuth session.
// It refuses an instance whose config-file URL has another origin than the URL its keyring credentials are bound to.
func NewSavedInstanceClient(cmd *cobra.Command, alias string, inst *config.Instance) (*api.Client, error) {
	if !sameOrigin(inst.URL, inst.TokenURL()) {
		return nil, fmt.Errorf("refusing to send the saved token for %q (%s) to %s — sign in again: youtrack auth login --session %s", alias, inst.TokenURL(), inst.URL, alias)
	}
	c := api.NewClient(inst.URL, inst.Token).WithContext(cmd.Context())
	if canRefresh(inst) {
		c.TokenSource = newOAuthSession(alias, inst, verboseWriter(cmd)).Token
	}
	return c, nil
}

func verboseWriter(cmd *cobra.Command) io.Writer {
	if verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose"); verbose {
		return cmd.ErrOrStderr()
	}
	return nil
}
