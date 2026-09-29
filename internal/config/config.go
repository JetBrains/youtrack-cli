package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Config is the top-level structure of ~/.youtrack-cli-config.json.
type Config struct {
	CurrentInstance string              `json:"currentInstance"`
	Instances       map[string]Instance `json:"instances"`
	LastUpdateCheck int64               `json:"lastUpdateCheck,omitempty"`

	// staleRefresh lists aliases whose OAuth session was replaced by a permanent token; Save drops their refresh-token keyring entries.
	staleRefresh []string
}

// Instance holds the credentials for a single YouTrack instance; Token is empty in the file when it lives in the OS keyring.
type Instance struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`

	// tokenInKeyring marks the token as keyring-stored so Save omits it from the file.
	tokenInKeyring bool
	// AuthMethod records how the token was obtained ("oauth" for browser SSO; empty for permanent tokens).
	AuthMethod string `json:"authMethod,omitempty"`
	// TokenExpiresAt is the unix timestamp after which the token expires; zero means never (permanent token).
	TokenExpiresAt int64 `json:"tokenExpiresAt,omitempty"`

	// RefreshToken renews an OAuth access token; like Token, it is empty in the file when it lives in the OS keyring.
	RefreshToken string `json:"refreshToken,omitempty"`
	// refreshTokenInKeyring marks the refresh token as keyring-stored so Save omits it from the file.
	refreshTokenInKeyring bool
	// OAuth session metadata recorded at sign-in so the access token can be refreshed and revoked without re-discovery.
	TokenType          string `json:"tokenType,omitempty"`
	Scope              string `json:"scope,omitempty"`
	HubURL             string `json:"hubUrl,omitempty"`
	ClientID           string `json:"clientId,omitempty"`
	TokenEndpoint      string `json:"tokenEndpoint,omitempty"`
	RevocationEndpoint string `json:"revocationEndpoint,omitempty"`
}

// AuthMethodOAuth marks an instance whose token came from the browser OAuth flow.
const AuthMethodOAuth = "oauth"

// OAuthSession is everything a browser sign-in yields that must be persisted for refresh and logout.
type OAuthSession struct {
	AccessToken        string
	RefreshToken       string
	ExpiresAt          int64
	TokenType          string
	Scope              string
	HubURL             string
	ClientID           string
	TokenEndpoint      string
	RevocationEndpoint string
}

// ConfigPath returns the absolute path to ~/.youtrack-cli-config.json.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}
	return filepath.Join(home, ".youtrack-cli-config.json"), nil
}

// Load reads ~/.youtrack-cli-config.json.
// Returns an empty Config (not an error) if the file does not exist.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{Instances: map[string]Instance{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if c.Instances == nil {
		c.Instances = map[string]Instance{}
	}
	for alias := range c.Instances {
		inst := c.Instances[alias]
		if inst.Token == "" {
			if tok, err := keyringGet(alias); err == nil && tok != "" {
				inst.Token = tok
				inst.tokenInKeyring = true
			}
		}
		// Only OAuth sessions have refresh tokens; skip the keyring round trip for permanent-token instances.
		if inst.AuthMethod == AuthMethodOAuth && inst.RefreshToken == "" {
			if tok, err := keyringGet(refreshKeyringUser(alias)); err == nil && tok != "" {
				inst.RefreshToken = tok
				inst.refreshTokenInKeyring = true
			}
		}
		c.Instances[alias] = inst
	}
	return &c, nil
}

// Save atomically writes the config to ~/.youtrack-cli-config.json at 0600.
// Uses write-to-temp + rename to avoid partial writes on crash (and so a rotated refresh token is never half-written).
// Tokens go to the OS keyring when available (migrating older file tokens), else stay in the file on any keyring failure.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	for alias := range c.Instances {
		inst := c.Instances[alias]
		if inst.Token != "" && !inst.tokenInKeyring && keyringSet(alias, inst.Token) == nil {
			inst.tokenInKeyring = true
		}
		if inst.RefreshToken != "" && !inst.refreshTokenInKeyring && keyringSet(refreshKeyringUser(alias), inst.RefreshToken) == nil {
			inst.refreshTokenInKeyring = true
		}
		c.Instances[alias] = inst
	}
	view := c.fileView()
	data, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := writeFileAtomic(path, append(data, '\n'), view.holdsSecrets()); err != nil {
		return err
	}
	for _, alias := range c.staleRefresh {
		PurgeKeyringRefreshToken(alias)
	}
	c.staleRefresh = nil
	return nil
}

// writeFileAtomic writes data to a fresh 0600 temp file beside path and renames it over path.
// A unique temp name keeps concurrent writers from clobbering each other's partial file, and a
// pre-existing, more permissive temp file can never be reused. When the data holds secrets, it
// refuses to proceed if the file permissions cannot be restricted to the owner.
func writeFileAtomic(path string, data []byte, secret bool) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	tmp := f.Name()
	cleanup := func() { _ = os.Remove(tmp) }
	if err := f.Chmod(0600); err != nil && runtime.GOOS != "windows" {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("restricting config permissions: %w", err)
	}
	if secret && runtime.GOOS != "windows" {
		// Some filesystems (e.g. network or foreign mounts) silently ignore chmod; never leave a token world-readable there.
		if fi, err := f.Stat(); err != nil || fi.Mode().Perm()&0o077 != 0 {
			_ = f.Close()
			cleanup()
			return fmt.Errorf("refusing to store credentials in %s: cannot restrict its permissions to 0600 — enable the OS keyring or move HOME to a filesystem that supports permissions", path)
		}
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("writing config: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("writing config: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("writing config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		cleanup()
		return fmt.Errorf("saving config: %w", err)
	}
	return nil
}

// fileView returns a copy of the config with keyring-stored tokens omitted.
func (c *Config) fileView() Config {
	out := *c
	out.Instances = make(map[string]Instance, len(c.Instances))
	for alias := range c.Instances {
		inst := c.Instances[alias]
		if inst.tokenInKeyring {
			inst.Token = ""
		}
		if inst.refreshTokenInKeyring {
			inst.RefreshToken = ""
		}
		out.Instances[alias] = inst
	}
	return out
}

// holdsSecrets reports whether any token is kept in the file itself rather than the keyring.
func (c Config) holdsSecrets() bool {
	for alias := range c.Instances {
		if inst := c.Instances[alias]; inst.Token != "" || inst.RefreshToken != "" {
			return true
		}
	}
	return false
}

// RefreshTokenInKeyring reports whether the refresh token for alias is stored in the OS keyring.
func (c *Config) RefreshTokenInKeyring(alias string) bool {
	return c.Instances[alias].refreshTokenInKeyring
}

// TokenInKeyring reports whether the token for alias is stored in the OS keyring.
func (c *Config) TokenInKeyring(alias string) bool {
	return c.Instances[alias].tokenInKeyring
}

// ActiveInstance returns a pointer to the active instance and its alias.
// The returned pointer points to a detached copy — mutating it has no effect on Config.
// Always call Save() after modifying instances through other methods.
// Returns (nil, "") when no instance is configured as active.
// Returns (nil, alias) when currentInstance names an alias that is missing from the map —
// callers should treat this as a stale-pointer error.
func (c *Config) ActiveInstance() (*Instance, string) {
	if c.CurrentInstance == "" {
		return nil, ""
	}
	inst, ok := c.Instances[c.CurrentInstance]
	if !ok {
		return nil, c.CurrentInstance
	}
	return &inst, c.CurrentInstance
}

// AddInstance upserts an instance. If alias is empty it is derived from the URL hostname.
// On alias collision with a different URL, a numeric suffix is appended (work → work-2).
// Returns the final alias used.
func (c *Config) AddInstance(alias, rawURL, token string) string {
	return c.upsertInstance(alias, rawURL, &Instance{URL: rawURL, Token: token})
}

// AddInstanceOAuth upserts an instance holding an OAuth session: a short-lived access token plus, when Hub issued one, a refresh token.
func (c *Config) AddInstanceOAuth(alias, rawURL string, sess *OAuthSession) string {
	return c.upsertInstance(alias, rawURL, &Instance{
		URL:                rawURL,
		Token:              sess.AccessToken,
		AuthMethod:         AuthMethodOAuth,
		TokenExpiresAt:     sess.ExpiresAt,
		RefreshToken:       sess.RefreshToken,
		TokenType:          sess.TokenType,
		Scope:              sess.Scope,
		HubURL:             sess.HubURL,
		ClientID:           sess.ClientID,
		TokenEndpoint:      sess.TokenEndpoint,
		RevocationEndpoint: sess.RevocationEndpoint,
	})
}

// SetOAuthTokens replaces the access token (and the refresh token, when Hub rotated it) of an existing OAuth instance; Save persists both.
func (c *Config) SetOAuthTokens(alias, accessToken, refreshToken string, expiresAt int64) {
	inst, ok := c.Instances[alias]
	if !ok {
		return
	}
	inst.Token = accessToken
	inst.tokenInKeyring = false // force Save to write the new value to the keyring
	inst.TokenExpiresAt = expiresAt
	if refreshToken != inst.RefreshToken {
		inst.RefreshToken = refreshToken
		inst.refreshTokenInKeyring = false
	}
	c.Instances[alias] = inst
}

// ClearOAuthTokens drops the tokens of an OAuth instance whose session is dead, keeping its URL and alias for re-login; call Save, then PurgeKeyringToken.
func (c *Config) ClearOAuthTokens(alias string) {
	inst, ok := c.Instances[alias]
	if !ok {
		return
	}
	inst.Token, inst.RefreshToken, inst.TokenExpiresAt = "", "", 0
	inst.tokenInKeyring, inst.refreshTokenInKeyring = false, false
	c.Instances[alias] = inst
}

func (c *Config) upsertInstance(alias, rawURL string, inst *Instance) string {
	if alias == "" {
		alias = DeriveAlias(rawURL)
	}
	final := alias
	for i := 2; ; i++ {
		existing, ok := c.Instances[final]
		if !ok || existing.URL == rawURL {
			break
		}
		final = fmt.Sprintf("%s-%d", alias, i)
	}
	if prev, ok := c.Instances[final]; ok && prev.AuthMethod == AuthMethodOAuth && inst.AuthMethod != AuthMethodOAuth {
		c.staleRefresh = append(c.staleRefresh, final)
	}
	c.Instances[final] = *inst
	return final
}

// RemoveInstance drops the alias from the in-memory config (error if not found); it does not touch the keyring, so callers must Save() then PurgeKeyringToken(alias).
func (c *Config) RemoveInstance(alias string) error {
	if _, ok := c.Instances[alias]; !ok {
		return fmt.Errorf("instance %q not found — run: youtrack instance list", alias)
	}
	delete(c.Instances, alias)
	return nil
}

// PurgeKeyringToken deletes the keyring entries (access and refresh token) for alias; call only after a successful Save().
// A missing entry or a disabled keyring is not an error.
func PurgeKeyringToken(alias string) error {
	var errs []error
	for _, user := range []string{alias, refreshKeyringUser(alias)} {
		err := keyringDelete(user)
		if err == nil || errors.Is(err, errKeyringDisabled) || errors.Is(err, errKeyringNotFound) {
			continue
		}
		errs = append(errs, fmt.Errorf("could not remove the token for %q from the OS keyring: %w — remove the %q entry for service %q manually", alias, err, user, keyringService))
	}
	return errors.Join(errs...)
}

// Aliases returns all configured instance aliases in sorted order.
func (c *Config) Aliases() []string {
	out := make([]string, 0, len(c.Instances))
	for k := range c.Instances {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// DeriveAlias extracts the first hostname label from a URL.
// "https://work.youtrack.cloud" → "work"
// Falls back to "default" on parse error or bare hostname with no dots.
func DeriveAlias(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return "default"
	}
	host := u.Hostname()
	if idx := strings.IndexByte(host, '.'); idx > 0 {
		return host[:idx]
	}
	return host
}

// PurgeKeyringRefreshToken deletes only the refresh-token keyring entry for alias, e.g. after a re-login that yielded no refresh token.
func PurgeKeyringRefreshToken(alias string) {
	_ = keyringDelete(refreshKeyringUser(alias))
}
