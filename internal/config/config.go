package config

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Config is the top-level structure of the config file at ConfigPath.
type Config struct {
	CurrentInstance string              `json:"currentInstance"`
	Instances       map[string]Instance `json:"instances"`
	LastUpdateCheck int64               `json:"lastUpdateCheck,omitempty"`

	// staleOAuth lists the keyring accounts of OAuth session versions that the next successful Save no longer references; it deletes them after the commit.
	staleOAuth []string
	// LatestVersion is the newest release seen by the last update check; the startup prompt reads it.
	LatestVersion string `json:"latestVersion,omitempty"`
	// DismissedVersion is the release the user chose to skip in the startup prompt.
	DismissedVersion string `json:"dismissedVersion,omitempty"`
}

// Instance holds the credentials for a single YouTrack instance; Token is empty in the file when it lives in the OS keyring.
type Instance struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`

	// tokenInKeyring marks the token as keyring-stored so Save omits it from the file.
	tokenInKeyring bool
	// tokenURLMissing marks a keyring token that has no URL entry, because an older version saved it, so Save binds it to the current URL.
	tokenURLMissing bool
	// tokenURLUnreadable marks a keyring token whose URL entry could not be read (not a confirmed missing one), so the token is not sent anywhere.
	tokenURLUnreadable bool
	// tokenURL is the instance URL stored for the token or the OAuth session in the keyring; empty for file tokens and for keyring tokens without a readable URL entry.
	tokenURL string
	// AuthMethod records how the token was obtained ("oauth" for browser SSO; empty for permanent tokens).
	AuthMethod string `json:"authMethod,omitempty"`
	// TokenExpiresAt is the unix timestamp after which the token expires; zero means never (permanent token).
	TokenExpiresAt int64 `json:"tokenExpiresAt,omitempty"`

	// RefreshToken renews an OAuth access token; like Token, it is empty in the file when it lives in the OS keyring.
	RefreshToken string `json:"refreshToken,omitempty"`
	// RefreshIssuedAt is the unix time when Hub issued the current refresh token; zero when unknown (a session saved by an older version).
	RefreshIssuedAt int64 `json:"refreshIssuedAt,omitempty"`
	// OAuthVersion names the keyring session entry version (see keyringOAuthService) that holds this session's tokens. It is the commit
	// point: Load reads only this version, so a version written by a Save whose file write failed is never read.
	OAuthVersion string `json:"oauthVersion,omitempty"`
	// oauthInKeyring marks the current tokens as stored in the keyring session entry, so Save omits the refresh token from the file.
	oauthInKeyring bool
	// oauthUnread marks a session entry that the file names but Load could not read (keyring outage): Save must not replace it, since
	// it would write a version without the refresh token.
	oauthUnread bool
	// oauthKeyringDown marks an oauthUnread entry whose read failed for another reason than a missing entry: the session may still be there.
	oauthKeyringDown bool
	// oauthBound marks TokenEndpoint and tokenURL as read from the keyring session entry, independent of where the refresh token lives.
	oauthBound bool
	// SessionID is new for each sign-in and kept across refreshes. Load ignores a keyring session entry with another ID,
	// and a refresh refuses a profile whose ID changed, so the tokens of two sign-ins are never mixed.
	SessionID string `json:"sessionId,omitempty"`
	// OAuth client and token endpoint recorded at sign-in so the access token can be refreshed without re-discovery.
	ClientID      string `json:"clientId,omitempty"`
	TokenEndpoint string `json:"tokenEndpoint,omitempty"`
}

// AuthMethodOAuth marks an instance whose token came from the browser OAuth flow.
const AuthMethodOAuth = "oauth"

// RefreshTokenLifetime is how long Hub accepts a refresh token: its default "remember me" interval.
// ponytail: Hub does not report the real lifetime, so a changed interval makes SessionExpiresAt an estimate.
const RefreshTokenLifetime = 90 * 24 * time.Hour

// SessionExpiresAt estimates when the refresh token of an OAuth session expires and a new sign-in is needed; zero when unknown.
func (i *Instance) SessionExpiresAt() int64 {
	if i.AuthMethod != AuthMethodOAuth || i.RefreshToken == "" || i.RefreshIssuedAt == 0 {
		return 0
	}
	return i.RefreshIssuedAt + int64(RefreshTokenLifetime/time.Second)
}

// OAuthSession is everything a browser sign-in yields that must be persisted for refresh and logout.
type OAuthSession struct {
	AccessToken   string
	RefreshToken  string
	ExpiresAt     int64
	IssuedAt      int64 // unix time of the token request; the refresh token's lifetime starts here
	ClientID      string
	TokenEndpoint string
}

// ConfigDir returns the youtrack-cli config directory ($XDG_CONFIG_HOME/youtrack or ~/.config/youtrack).
// A relative XDG_CONFIG_HOME is ignored, as the XDG spec requires.
func ConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(configHome) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home dir: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "youtrack"), nil
}

// ConfigPath returns the absolute path to config.json in ConfigDir.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// legacyConfigPath is where versions before the XDG move kept the config.
func legacyConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}
	return filepath.Join(home, ".youtrack-cli-config.json"), nil
}

// Load reads the config file, moving a legacy ~/.youtrack-cli-config.json to ConfigPath first.
// Returns an empty Config (not an error) if no config file exists.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	path = migrateLegacy(path)
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
		fileToken := inst.Token != ""
		if inst.Token == "" {
			if tok, err := keyringGet(keyringService, alias); err == nil && tok != "" {
				inst.Token = tok
				inst.tokenInKeyring = true
				// Only a confirmed missing URL entry is bound later; after a failed read the token is unusable, since the file URL may be edited.
				switch u, err := keyringGet(keyringURLService, alias); {
				case err == nil && u != "":
					inst.tokenURL = u
				case err == nil || errors.Is(err, errKeyringNotFound):
					inst.tokenURLMissing = true
				default:
					inst.tokenURLUnreadable = true
				}
			}
		}
		// Only OAuth sessions have a keyring session entry; skip the round trip for permanent-token instances.
		if inst.AuthMethod == AuthMethodOAuth && inst.OAuthVersion != "" {
			var s oauthSecret
			raw, err := keyringGet(keyringOAuthService, oauthAccount(alias, inst.OAuthVersion))
			if err == nil && json.Unmarshal([]byte(raw), &s) == nil && s.SessionID == inst.SessionID {
				// The keyring endpoint and URL win over the file.
				inst.TokenEndpoint, inst.oauthBound = s.TokenEndpoint, true
				if s.URL != "" {
					inst.tokenURL, inst.tokenURLMissing, inst.tokenURLUnreadable = s.URL, false, false
				}
				// A token in the file is newer: its keyring write failed at the last Save. Otherwise the version's own access token
				// (with its expiry) wins over the bare token entry, which a Save that was not committed may have overwritten.
				if !fileToken && s.AccessToken != "" {
					inst.Token, inst.TokenExpiresAt, inst.tokenInKeyring = s.AccessToken, s.ExpiresAt, true
				}
				if inst.RefreshToken == "" {
					inst.RefreshToken, inst.oauthInKeyring = s.RefreshToken, true
				}
			} else if inst.RefreshToken == "" {
				inst.oauthUnread = true
				inst.oauthKeyringDown = err != nil && !errors.Is(err, errKeyringNotFound)
			}
		}
		c.Instances[alias] = inst
	}
	return &c, nil
}

// Save atomically writes the config to ConfigPath at 0600, creating its directory at 0700.
// Uses write-to-temp + rename to avoid partial writes on crash (and so a rotated refresh token is never half-written).
// Tokens go to the OS keyring when available (migrating older file tokens), else stay in the file on any keyring failure.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	// written records the session versions this Save added, with the instance before the write, so a failed commit can undo them.
	type oauthWrite struct {
		alias, account string
		before         Instance
	}
	var written []oauthWrite
	var replaced []string // previous versions of the sessions written below
	for alias := range c.Instances {
		inst := c.Instances[alias]
		before := inst
		// Decided before the token moves to the keyring: endpoints are bound only while they are trusted.
		bindOAuth := inst.AuthMethod == AuthMethodOAuth && !inst.oauthInKeyring && !inst.oauthUnread && inst.OAuthEndpointsTrusted() && (inst.Token != "" || inst.RefreshToken != "")
		switch keyringWriteFor(&inst) {
		case keyringWriteToken:
			// The URL goes first, so the keyring never holds a new token without its URL; on any failure the token stays in the file.
			if keyringSet(keyringURLService, alias, inst.URL) == nil && keyringSet(keyringService, alias, inst.Token) == nil {
				inst.tokenInKeyring, inst.tokenURLMissing, inst.tokenURLUnreadable, inst.tokenURL = true, false, false, inst.URL
			}
		case keyringWriteURL:
			if keyringSet(keyringURLService, alias, inst.URL) == nil {
				inst.tokenURLMissing, inst.tokenURL = false, inst.URL
			}
		}
		if bindOAuth {
			// A new version, never an overwrite: until the file names it, Load keeps reading the previous one.
			version := rand.Text()
			account := oauthAccount(alias, version)
			s, _ := json.Marshal(oauthSecret{AccessToken: inst.Token, ExpiresAt: inst.TokenExpiresAt, RefreshToken: inst.RefreshToken, TokenEndpoint: inst.TokenEndpoint, URL: inst.TokenURL(), SessionID: inst.SessionID})
			if keyringSet(keyringOAuthService, account, string(s)) == nil {
				if inst.OAuthVersion != "" {
					replaced = append(replaced, oauthAccount(alias, inst.OAuthVersion))
				}
				written = append(written, oauthWrite{alias: alias, account: account, before: before})
				inst.OAuthVersion, inst.oauthInKeyring, inst.oauthBound = version, true, true
			}
		}
		c.Instances[alias] = inst
	}
	view := c.fileView()
	data, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := writeFileAtomic(path, append(data, '\n'), view.holdsSecrets()); err != nil {
		// Not committed: the file still names the previous versions, which stay intact. Drop the new ones.
		// ponytail: a crash before this point leaves the new version behind unreferenced; go-keyring cannot list entries to collect it.
		for i := range written {
			_ = keyringDelete(keyringOAuthService, written[i].account)
			c.Instances[written[i].alias] = written[i].before
		}
		return err
	}
	// Committed: the versions the file no longer names are deleted. A failed delete leaves an unreferenced entry, never a wrong one.
	for _, account := range append(c.staleOAuth, replaced...) {
		_ = keyringDelete(keyringOAuthService, account)
	}
	c.staleOAuth = nil
	return nil
}

// writeFileAtomic writes data to a fresh 0600 temp file beside path and renames it over path.
// A unique temp name keeps concurrent writers from clobbering each other's partial file, and a
// pre-existing, more permissive temp file can never be reused. When the data holds secrets, it
// refuses to proceed if the file permissions cannot be restricted to the owner.
func writeFileAtomic(path string, data []byte, secret bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	tmp := f.Name()
	if err := writeAndClose(f, data, secret, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("saving config: %w", err)
	}
	if legacy, err := legacyConfigPath(); err == nil {
		_ = os.Remove(legacy)
	}
	return nil
}

// writeAndClose restricts f to 0600, writes and syncs data, and always closes f; the caller removes f on error.
func writeAndClose(f *os.File, data []byte, secret bool, path string) (err error) {
	defer func() {
		if cerr := f.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("writing config: %w", cerr)
		}
	}()
	if err := f.Chmod(0600); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("restricting config permissions: %w", err)
	}
	if secret && runtime.GOOS != "windows" {
		// Some filesystems (e.g. network or foreign mounts) silently ignore chmod; never leave a token world-readable there.
		if fi, err := f.Stat(); err != nil || fi.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("refusing to store credentials in %s: cannot restrict its permissions to 0600 — enable the OS keyring or move HOME to a filesystem that supports permissions", path)
		}
	}
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// migrateLegacy moves a legacy config to path when path does not exist yet and returns the file Load should read.
// If the move fails and no other process has migrated it meanwhile, it returns the legacy path,
// so the next Save writes path and removes the legacy file.
func migrateLegacy(path string) string {
	legacy, err := legacyConfigPath()
	if err != nil {
		return path
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		return path
	}
	if _, err := os.Stat(legacy); err != nil {
		return path
	}
	if os.MkdirAll(filepath.Dir(path), 0700) == nil && os.Rename(legacy, path) == nil {
		return path
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return legacy
}

// keyringWrite is what Save must write to the keyring for one instance.
type keyringWrite int

const (
	keyringWriteNone  keyringWrite = iota
	keyringWriteToken              // a file or newly added token: store the token and its URL
	keyringWriteURL                // a keyring token saved by an older version: add its URL entry
)

// keyringWriteFor decides what Save writes for inst. A keyring token that already has a URL is never rewritten:
// binding it to the URL in the config file again would trust an edited file.
func keyringWriteFor(inst *Instance) keyringWrite {
	switch {
	case inst.Token == "":
		return keyringWriteNone
	case !inst.tokenInKeyring:
		return keyringWriteToken
	case inst.tokenURLMissing:
		return keyringWriteURL
	default:
		return keyringWriteNone
	}
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
		if inst.oauthInKeyring {
			inst.RefreshToken = ""
		}
		// An unread session entry keeps its OAuthVersion, so the tokens are read back once the keyring recovers.
		out.Instances[alias] = inst
	}
	return out
}

// holdsSecrets reports whether any token is kept in the file itself rather than the keyring.
func (c *Config) holdsSecrets() bool {
	for alias := range c.Instances {
		if inst := c.Instances[alias]; inst.Token != "" || inst.RefreshToken != "" {
			return true
		}
	}
	return false
}

// RefreshTokenInKeyring reports whether the refresh token for alias is stored in the OS keyring.
func (c *Config) RefreshTokenInKeyring(alias string) bool {
	inst := c.Instances[alias]
	return inst.oauthInKeyring && inst.RefreshToken != ""
}

// OAuthEndpointsTrusted reports whether TokenEndpoint may receive this instance's refresh token:
// it comes from the keyring entry bound at sign-in, or the refresh token itself lives in the file (so the file holds both and an edit redirects nothing new).
func (i *Instance) OAuthEndpointsTrusted() bool {
	return i.oauthBound || !i.oauthInKeyring
}

// OAuthSessionUnreadable reports whether the keyring entry with this session's tokens could not be read (locked or unavailable keyring),
// as opposed to a session that has no tokens.
func (i *Instance) OAuthSessionUnreadable() bool {
	return i.oauthKeyringDown
}

// TokenURL returns the URL the token is bound to: the URL saved for it in the keyring, else the URL from the config file.
func (i *Instance) TokenURL() string {
	if i.tokenURL != "" {
		return i.tokenURL
	}
	return i.URL
}

// TokenURLUnreadable reports whether the token lives in the keyring but the URL it is bound to could not be read from there.
func (i *Instance) TokenURLUnreadable() bool {
	return i.tokenURLUnreadable
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
		URL:             rawURL,
		Token:           sess.AccessToken,
		AuthMethod:      AuthMethodOAuth,
		TokenExpiresAt:  sess.ExpiresAt,
		RefreshToken:    sess.RefreshToken,
		RefreshIssuedAt: sess.IssuedAt,
		ClientID:        sess.ClientID,
		TokenEndpoint:   sess.TokenEndpoint,
		SessionID:       rand.Text(),
	})
}

// SetOAuthTokens replaces the access token (and the refresh token, when Hub rotated it) of an existing OAuth instance; Save persists both.
// A rotated refresh token starts a new lifetime at issuedAt.
func (c *Config) SetOAuthTokens(alias, accessToken, refreshToken string, expiresAt, issuedAt int64) {
	inst, ok := c.Instances[alias]
	if !ok {
		return
	}
	if refreshToken != inst.RefreshToken {
		inst.RefreshIssuedAt = issuedAt
	}
	inst.Token, inst.RefreshToken, inst.TokenExpiresAt = accessToken, refreshToken, expiresAt
	// Force Save to write the new values: the bare token entry, and a new session version, which also holds the access token.
	inst.tokenInKeyring, inst.oauthInKeyring, inst.oauthUnread = false, false, false
	c.Instances[alias] = inst
}

// ClearOAuthTokens drops the tokens of an OAuth instance whose session is dead, keeping its URL and alias for re-login; call Save, then PurgeKeyringToken.
func (c *Config) ClearOAuthTokens(alias string) {
	inst, ok := c.Instances[alias]
	if !ok {
		return
	}
	if inst.OAuthVersion != "" {
		c.staleOAuth = append(c.staleOAuth, oauthAccount(alias, inst.OAuthVersion))
	}
	// Expiry 1 (not 0 = never): if the keyring purge fails and Load re-imports the dead tokens, they are refreshed (and rejected) rather than used as-is.
	inst.Token, inst.RefreshToken, inst.TokenExpiresAt, inst.OAuthVersion = "", "", 1, ""
	inst.tokenInKeyring, inst.oauthInKeyring, inst.oauthUnread, inst.oauthBound, inst.tokenURLMissing, inst.tokenURLUnreadable, inst.tokenURL = false, false, false, false, false, false, ""
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
	// The replaced session's entry is deleted once the replacement is committed.
	if prev, ok := c.Instances[final]; ok && prev.OAuthVersion != "" {
		c.staleOAuth = append(c.staleOAuth, oauthAccount(final, prev.OAuthVersion))
	}
	c.Instances[final] = *inst
	return final
}

// RemoveInstance drops the alias from the in-memory config (error if not found); it does not touch the keyring, so callers must Save() then PurgeKeyringToken.
func (c *Config) RemoveInstance(alias string) error {
	if _, ok := c.Instances[alias]; !ok {
		return fmt.Errorf("instance %q not found — run: youtrack instance list", alias)
	}
	delete(c.Instances, alias)
	return nil
}

// PurgeKeyringToken deletes the keyring entries (access token, its URL, and the OAuth session version oauthVersion, if any) for alias;
// call only after a successful Save().
// A missing entry or a disabled keyring is not an error.
func PurgeKeyringToken(alias, oauthVersion string) error {
	entries := [][2]string{{keyringService, alias}, {keyringURLService, alias}}
	if oauthVersion != "" {
		entries = append(entries, [2]string{keyringOAuthService, oauthAccount(alias, oauthVersion)})
	}
	var errs []error
	for _, e := range entries {
		err := keyringDelete(e[0], e[1])
		if err == nil || errors.Is(err, errKeyringDisabled) || errors.Is(err, errKeyringNotFound) {
			continue
		}
		errs = append(errs, fmt.Errorf("could not remove the token for %q from the OS keyring: %w — remove the %q entry for service %q manually", alias, err, e[1], e[0]))
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
