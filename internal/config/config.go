package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Config is the top-level structure of the config file at ConfigPath.
type Config struct {
	CurrentInstance string              `json:"currentInstance"`
	Instances       map[string]Instance `json:"instances"`
	LastUpdateCheck int64               `json:"lastUpdateCheck,omitempty"`
	// LatestVersion is the newest release seen by the last update check; the startup prompt reads it.
	LatestVersion string `json:"latestVersion,omitempty"`
	// DismissedVersion is the release the user chose to skip in the startup prompt.
	DismissedVersion string `json:"dismissedVersion,omitempty"`
}

// Instance holds the credentials for a single YouTrack instance; Token is empty in the file when it lives in the OS keyring.
type Instance struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`

	// tokenInKeyring marks the token as keyring-stored so save omits it from the file.
	tokenInKeyring bool
	// tokenURLMissing marks a keyring token that has no URL entry, because an older version saved it, so save binds it to the current URL.
	tokenURLMissing bool
	// tokenURL is the instance URL stored for the token in the keyring; empty for file tokens and for keyring tokens without a readable URL entry.
	tokenURL string
	// AuthMethod records how the token was obtained ("oauth" for browser SSO; empty for permanent tokens).
	AuthMethod string `json:"authMethod,omitempty"`
	// TokenExpiresAt is the unix timestamp after which the token expires; zero means never (permanent token).
	TokenExpiresAt int64 `json:"tokenExpiresAt,omitempty"`
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
	for alias, inst := range c.Instances {
		if inst.Token != "" {
			continue
		}
		if tok, err := keyringGet(keyringService, alias); err == nil && tok != "" {
			inst.Token = tok
			inst.tokenInKeyring = true
			// Only a confirmed missing URL entry is bound later; a failed read leaves the token bound to the file URL without rebinding it.
			switch u, err := keyringGet(keyringURLService, alias); {
			case err == nil && u != "":
				inst.tokenURL = u
			case err == nil || errors.Is(err, errKeyringNotFound):
				inst.tokenURLMissing = true
			}
			c.Instances[alias] = inst
		}
	}
	return &c, nil
}

// Update loads the config, applies fn, and saves the result while holding a cross-process lock,
// so concurrent CLI processes never lose each other's changes. If fn returns an error, nothing is saved.
// Keep fn short: other processes that write the config wait until it returns.
func Update(fn func(*Config) error) (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("creating config dir: %w", err)
	}
	lock, err := os.OpenFile(path+".lock", os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening config lock: %w", err)
	}
	defer func() { _ = lock.Close() }()
	if err := lockFile(lock); err != nil {
		return nil, fmt.Errorf("locking config: %w", err)
	}
	c, err := Load()
	if err != nil {
		return nil, err
	}
	if err := fn(c); err != nil {
		return nil, err
	}
	if err := c.save(path); err != nil {
		return nil, err
	}
	return c, nil
}

// save atomically writes the config to path at 0600; Update creates its directory.
// Uses write-to-temp + rename to avoid partial writes on crash.
// Tokens go to the OS keyring when available (migrating older file tokens), else stay in the file on any keyring failure.
func (c *Config) save(path string) error {
	for alias, inst := range c.Instances {
		switch keyringWriteFor(&inst) {
		case keyringWriteToken:
			// The URL goes first, so the keyring never holds a new token without its URL; on any failure the token stays in the file.
			if keyringSet(keyringURLService, alias, inst.URL) != nil || keyringSet(keyringService, alias, inst.Token) != nil {
				continue
			}
			inst.tokenInKeyring, inst.tokenURLMissing, inst.tokenURL = true, false, inst.URL
		case keyringWriteURL:
			if keyringSet(keyringURLService, alias, inst.URL) != nil {
				continue
			}
			inst.tokenURLMissing, inst.tokenURL = false, inst.URL
		default:
			continue
		}
		c.Instances[alias] = inst
	}
	data, err := json.MarshalIndent(c.fileView(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp") // created at 0600
	if err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	tmp := f.Name()
	_, err = f.Write(append(data, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp) // clean up any partial write
		return fmt.Errorf("writing config: %w", err)
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

// migrateLegacy moves a legacy config to path when path does not exist yet and returns the file Load should read.
// If the move fails and no other process has migrated it meanwhile, it returns the legacy path,
// so the next Update writes path and removes the legacy file.
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

// keyringWrite is what save must write to the keyring for one instance.
type keyringWrite int

const (
	keyringWriteNone  keyringWrite = iota
	keyringWriteToken              // a file or newly added token: store the token and its URL
	keyringWriteURL                // a keyring token saved by an older version: add its URL entry
)

// keyringWriteFor decides what save writes for inst. A keyring token that already has a URL is never rewritten:
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
	for alias, inst := range c.Instances {
		if inst.tokenInKeyring {
			inst.Token = ""
		}
		out.Instances[alias] = inst
	}
	return out
}

// TokenURL returns the URL the token is bound to: the URL saved for it in the keyring, else the URL from the config file.
func (i *Instance) TokenURL() string {
	if i.tokenURL != "" {
		return i.tokenURL
	}
	return i.URL
}

// TokenInKeyring reports whether the token for alias is stored in the OS keyring.
func (c *Config) TokenInKeyring(alias string) bool {
	return c.Instances[alias].tokenInKeyring
}

// ActiveInstance returns a pointer to the active instance and its alias.
// The returned pointer points to a detached copy — mutating it has no effect on Config.
// Modify instances through Config methods inside Update.
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

// AddInstanceOAuth upserts an instance holding a short-lived OAuth access token that expires at the given unix timestamp.
func (c *Config) AddInstanceOAuth(alias, rawURL, token string, expiresAt int64) string {
	return c.upsertInstance(alias, rawURL, &Instance{
		URL:            rawURL,
		Token:          token,
		AuthMethod:     "oauth",
		TokenExpiresAt: expiresAt,
	})
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
	c.Instances[final] = *inst
	return final
}

// RemoveInstance drops the alias from the in-memory config (error if not found); it does not touch the keyring, so callers must call it inside Update, then PurgeKeyringToken(alias).
func (c *Config) RemoveInstance(alias string) error {
	if _, ok := c.Instances[alias]; !ok {
		return fmt.Errorf("instance %q not found — run: youtrack instance list", alias)
	}
	delete(c.Instances, alias)
	return nil
}

// PurgeKeyringToken deletes the keyring token and URL entries for alias; call only after a successful Update.
// A missing entry or a disabled keyring is not an error.
func PurgeKeyringToken(alias string) error {
	for _, service := range []string{keyringService, keyringURLService} {
		err := keyringDelete(service, alias)
		if err == nil || errors.Is(err, errKeyringDisabled) || errors.Is(err, errKeyringNotFound) {
			continue
		}
		return fmt.Errorf("could not remove the token for %q from the OS keyring: %w — remove the %q entry for service %q manually", alias, err, alias, service)
	}
	return nil
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
