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

// Config is the top-level structure of ~/.youtrack-cli-config.json.
type Config struct {
	CurrentInstance string              `json:"currentInstance"`
	Instances       map[string]Instance `json:"instances"`
	LastUpdateCheck int64               `json:"lastUpdateCheck,omitempty"`
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
	for alias, inst := range c.Instances {
		if inst.Token != "" {
			continue
		}
		if tok, err := keyringGet(alias); err == nil && tok != "" {
			inst.Token = tok
			inst.tokenInKeyring = true
			c.Instances[alias] = inst
		}
	}
	return &c, nil
}

// Save atomically writes the config to ~/.youtrack-cli-config.json at 0600.
// Uses write-to-temp + rename to avoid partial writes on crash.
// Tokens go to the OS keyring when available (migrating older file tokens), else stay in the file on any keyring failure.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	for alias, inst := range c.Instances {
		if inst.Token == "" || inst.tokenInKeyring {
			continue
		}
		if keyringSet(alias, inst.Token) == nil {
			inst.tokenInKeyring = true
			c.Instances[alias] = inst
		}
	}
	data, err := json.MarshalIndent(c.fileView(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0600); err != nil {
		_ = os.Remove(tmp) // clean up any partial write
		return fmt.Errorf("writing config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("saving config: %w", err)
	}
	return nil
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
	return c.upsertInstance(alias, rawURL, Instance{URL: rawURL, Token: token})
}

// AddInstanceOAuth upserts an instance holding a short-lived OAuth access token that expires at the given unix timestamp.
func (c *Config) AddInstanceOAuth(alias, rawURL, token string, expiresAt int64) string {
	return c.upsertInstance(alias, rawURL, Instance{
		URL:            rawURL,
		Token:          token,
		AuthMethod:     "oauth",
		TokenExpiresAt: expiresAt,
	})
}

func (c *Config) upsertInstance(alias, rawURL string, inst Instance) string {
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
	c.Instances[final] = inst
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

// PurgeKeyringToken best-effort deletes the keyring entry for alias; call only after a successful Save().
func PurgeKeyringToken(alias string) {
	_ = keyringDelete(alias)
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
