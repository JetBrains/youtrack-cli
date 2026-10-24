package config

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gokeyring "github.com/zalando/go-keyring"
)

// isolate points the config at a temp HOME with file-only storage, so tests never touch the real keyring or config.
func isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "1")
	path := filepath.Join(home, ".config", "youtrack", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOAuthSessionRoundTrip(t *testing.T) {
	path := isolate(t)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	alias := cfg.AddInstanceOAuth("work", "https://work.example.com", &OAuthSession{
		AccessToken: "at-1", RefreshToken: "rt-1", ExpiresAt: 1234, TokenType: "Bearer", Scope: "svc",
		HubURL: "https://work.example.com/hub", ClientID: "cli", TokenEndpoint: "https://work.example.com/hub/api/rest/oauth2/token",
	})
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	inst := got.Instances[alias]
	if inst.Token != "at-1" || inst.RefreshToken != "rt-1" || inst.TokenExpiresAt != 1234 || inst.AuthMethod != AuthMethodOAuth ||
		inst.HubURL != "https://work.example.com/hub" || inst.ClientID != "cli" || inst.Scope != "svc" || inst.TokenType != "Bearer" {
		t.Fatalf("session not persisted: %+v", inst)
	}

	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Fatalf("config mode = %o, want 600", fi.Mode().Perm())
		}
	}

	// Rotation replaces the refresh token; no rotation keeps it.
	got.SetOAuthTokens(alias, "at-2", "rt-2", 5678)
	if err := got.Save(); err != nil {
		t.Fatal(err)
	}
	again, _ := Load()
	if i := again.Instances[alias]; i.Token != "at-2" || i.RefreshToken != "rt-2" || i.TokenExpiresAt != 5678 {
		t.Fatalf("rotation not persisted: %+v", i)
	}

	again.ClearOAuthTokens(alias)
	if err := again.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "at-2") || strings.Contains(string(data), "rt-2") {
		t.Fatalf("cleared tokens still in the file: %s", data)
	}
	cleared, _ := Load()
	if i := cleared.Instances[alias]; i.URL != "https://work.example.com" || i.AuthMethod != AuthMethodOAuth {
		t.Fatalf("clearing must keep the profile for re-login: %+v", i)
	}
}

func TestReplacementSessionIgnoresStaleKeyringRefreshToken(t *testing.T) {
	path := isolate(t)
	gokeyring.MockInit()
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	session := func(at, rt string) *OAuthSession {
		return &OAuthSession{AccessToken: at, RefreshToken: rt, ExpiresAt: 1234, TokenType: "Bearer",
			HubURL: "https://work.example.com/hub", ClientID: "cli", TokenEndpoint: "https://work.example.com/hub/api/rest/oauth2/token"}
	}

	// Alice's session goes to the keyring and is read back from it.
	cfg, _ := Load()
	cfg.AddInstanceOAuth("work", "https://work.example.com", session("at-alice", "rt-alice"))
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if got, _ := Load(); got.Instances["work"].RefreshToken != "rt-alice" {
		t.Fatalf("refresh token not read back from the keyring: %+v", got.Instances["work"])
	}
	if data, _ := os.ReadFile(path); strings.Contains(string(data), "rt-alice") {
		t.Fatalf("keyring-stored refresh token leaked into the file: %s", data)
	}

	// Bob replaces the session while the keyring is down, and Hub issues no refresh token: Alice's keyring entry stays behind.
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "1")
	cfg, _ = Load()
	cfg.AddInstanceOAuth("work", "https://work.example.com", session("at-bob", ""))
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// With the keyring back, Bob's session must not pick up Alice's refresh token.
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	got, _ := Load()
	if inst := got.Instances["work"]; inst.Token != "at-bob" || inst.RefreshToken != "" {
		t.Fatalf("replacement session reads a stale keyring refresh token: token=%q refreshToken=%q", inst.Token, inst.RefreshToken)
	}
}

func TestLegacyConfigStillLoads(t *testing.T) {
	path := isolate(t)
	// A config written by an older version: permanent token, and an OAuth session without refresh metadata.
	legacy := `{"currentInstance":"a","instances":{"a":{"url":"https://a.example.com","token":"perm:a"},"b":{"url":"https://b.example.com","token":"sso","authMethod":"oauth","tokenExpiresAt":99}}}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Instances["a"].Token != "perm:a" || cfg.Instances["b"].Token != "sso" || cfg.Instances["b"].RefreshToken != "" {
		t.Fatalf("legacy config misread: %+v", cfg.Instances)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "refreshToken") || strings.Contains(string(data), "hubUrl") {
		t.Fatalf("saving a legacy config added empty OAuth fields: %s", data)
	}
}

func TestSaveNeverReusesPermissiveTempFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not meaningful on Windows")
	}
	path := isolate(t)
	// A world-readable leftover from an older version's fixed temp name must not become the config.
	if err := os.WriteFile(path+".tmp", []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, _ := Load()
	cfg.AddInstance("a", "https://a.example.com", "perm:secret")
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", fi.Mode().Perm())
	}
	matches, _ := filepath.Glob(path + ".*.tmp")
	if len(matches) != 0 {
		t.Fatalf("temp files left behind: %v", matches)
	}
}

func TestLockIsExclusive(t *testing.T) {
	isolate(t)
	ctx := context.Background()

	unlock, err := Lock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var acquired atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		u2, err := Lock(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		acquired.Store(true)
		u2()
	}()
	time.Sleep(200 * time.Millisecond)
	if acquired.Load() {
		t.Fatal("second Lock acquired while the first was held")
	}
	unlock()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("second Lock never acquired after release")
	}

	// A canceled context stops waiting.
	unlock, err = Lock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := Lock(canceled); err == nil {
		t.Fatal("Lock with a canceled context succeeded while held")
	}
}
