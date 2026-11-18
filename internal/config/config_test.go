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
		AccessToken: "at-1", RefreshToken: "rt-1", ExpiresAt: 1234,
		ClientID: "cli", TokenEndpoint: "https://work.example.com/hub/api/rest/oauth2/token",
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
		inst.ClientID != "cli" {
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
	got.SetOAuthTokens(alias, "at-2", "rt-2", 5678, 0)
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
		return &OAuthSession{AccessToken: at, RefreshToken: rt, ExpiresAt: 1234,
			ClientID: "cli", TokenEndpoint: "https://work.example.com/hub/api/rest/oauth2/token"}
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

func TestSaveDuringKeyringOutageKeepsRefreshToken(t *testing.T) {
	isolate(t)
	gokeyring.MockInit()
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	cfg, _ := Load()
	cfg.AddInstanceOAuth("work", "https://work.example.com", &OAuthSession{AccessToken: "at", RefreshToken: "rt", ExpiresAt: 1234,
		ClientID: "cli", TokenEndpoint: "https://work.example.com/hub/api/rest/oauth2/token"})
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// An unrelated save (e.g. the update check) while the keyring cannot be read must not drop the session's refresh token.
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "1")
	cfg, _ = Load()
	cfg.LastUpdateCheck = 1
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	if got, _ := Load(); got.Instances["work"].RefreshToken != "rt" {
		t.Fatalf("refresh token lost after a save during a keyring outage: %+v", got.Instances["work"])
	}

	// Clearing the session during an outage must not let its refresh token come back when the keyring recovers.
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "1")
	cfg, _ = Load()
	cfg.ClearOAuthTokens("work")
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	if got, _ := Load(); got.Instances["work"].RefreshToken != "" {
		t.Fatalf("cleared refresh token read back from the keyring: %+v", got.Instances["work"])
	}
}

func TestReplacementSessionIgnoresStaleKeyringEndpoint(t *testing.T) {
	isolate(t)
	gokeyring.MockInit()
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	session := func(rt, hub string) *OAuthSession {
		return &OAuthSession{AccessToken: "at-" + rt, RefreshToken: rt, ExpiresAt: 1234, ClientID: "cli", TokenEndpoint: hub + "/api/rest/oauth2/token"}
	}

	cfg, _ := Load()
	cfg.AddInstanceOAuth("work", "https://work.example.com", session("rt-alice", "https://old-hub.example.com"))
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// A new sign-in against another Hub is saved while the keyring is down: the old keyring session entry stays behind.
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "1")
	cfg, _ = Load()
	cfg.AddInstanceOAuth("work", "https://work.example.com", session("rt-bob", "https://new-hub.example.com"))
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	bobID := cfg.Instances["work"].SessionID

	// With the keyring back, the new refresh token must go to the new Hub only.
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	got, _ := Load()
	if inst := got.Instances["work"]; inst.RefreshToken != "rt-bob" || inst.TokenEndpoint != "https://new-hub.example.com/api/rest/oauth2/token" {
		t.Fatalf("replacement session mixed with a stale keyring entry: refreshToken=%q tokenEndpoint=%q", inst.RefreshToken, inst.TokenEndpoint)
	}

	// A refresh keeps the session ID, so a running command still recognizes its own session.
	got.SetOAuthTokens("work", "at-2", "rt-bob-2", 5678, 0)
	if err := got.Save(); err != nil {
		t.Fatal(err)
	}
	if again, _ := Load(); again.Instances["work"].SessionID != bobID || again.Instances["work"].RefreshToken != "rt-bob-2" {
		t.Fatalf("refresh changed the session: %+v", again.Instances["work"])
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
	if strings.Contains(string(data), "refreshToken") || strings.Contains(string(data), "clientId") {
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

// A Save whose file write fails must leave the committed session usable: the keyring versions it wrote are not referenced.
func TestFailedSaveKeepsCommittedSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a read-only directory does not block file creation on Windows")
	}
	path := isolate(t)
	gokeyring.MockInit()
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	session := func(at, rt string) *OAuthSession {
		return &OAuthSession{AccessToken: at, RefreshToken: rt, ExpiresAt: 1234, ClientID: "cli", TokenEndpoint: "https://work.example.com/hub/api/rest/oauth2/token"}
	}
	failSave := func(cfg *Config) {
		t.Helper()
		dir := filepath.Dir(path)
		if err := os.Chmod(dir, 0500); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(dir, 0700) }()
		if err := cfg.Save(); err == nil {
			t.Fatal("Save succeeded in a read-only directory")
		}
	}
	wantAlice := func(step string) {
		t.Helper()
		got, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if inst := got.Instances["work"]; inst.Token != "at-alice" || inst.RefreshToken != "rt-alice" || inst.TokenExpiresAt != 1234 {
			t.Fatalf("%s: committed session lost: token=%q refreshToken=%q expiresAt=%d", step, inst.Token, inst.RefreshToken, inst.TokenExpiresAt)
		}
	}

	cfg, _ := Load()
	cfg.AddInstanceOAuth("work", "https://work.example.com", session("at-alice", "rt-alice"))
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	aliceVersion := cfg.Instances["work"].OAuthVersion

	// A new sign-in overwrites the bare token entry before the file write fails.
	cfg, _ = Load()
	cfg.AddInstanceOAuth("work", "https://work.example.com", session("at-bob", "rt-bob"))
	failSave(cfg)
	wantAlice("failed re-login")

	// A refresh whose file write fails keeps the previous version as well.
	cfg, _ = Load()
	cfg.SetOAuthTokens("work", "at-alice-2", "rt-alice-2", 5678, 0)
	failSave(cfg)
	wantAlice("failed refresh")

	// A committed refresh deletes the version it replaces.
	cfg, _ = Load()
	cfg.SetOAuthTokens("work", "at-alice-2", "rt-alice", 5678, 0)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if got, _ := Load(); got.Instances["work"].Token != "at-alice-2" {
		t.Fatalf("committed refresh not read back: %+v", got.Instances["work"])
	}
	if _, err := gokeyring.Get(keyringOAuthService, oauthAccount("work", aliceVersion)); err == nil {
		t.Fatal("the replaced session version is still in the keyring")
	}
}

// A sign-in whose bare token reached the keyring while the session entry did not (the entry is larger, so it can exceed the
// keyring size limit) keeps the refresh token and its endpoint in the file. The session must still refresh, and the next Save binds it.
func TestPartialKeyringWriteKeepsSessionRefreshable(t *testing.T) {
	path := isolate(t)
	gokeyring.MockInit()
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	if err := gokeyring.Set(keyringService, "work", "at-1"); err != nil {
		t.Fatal(err)
	}
	if err := gokeyring.Set(keyringURLService, "work", "https://work.example.com"); err != nil {
		t.Fatal(err)
	}
	file := `{"currentInstance":"work","instances":{"work":{"url":"https://work.example.com","authMethod":"oauth","tokenExpiresAt":1234,` +
		`"refreshToken":"rt-1","sessionId":"s1","clientId":"cli","tokenEndpoint":"https://work.example.com/hub/api/rest/oauth2/token"}}}`
	if err := os.WriteFile(path, []byte(file), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if inst := cfg.Instances["work"]; inst.Token != "at-1" || inst.RefreshToken != "rt-1" || !inst.OAuthEndpointsTrusted() {
		t.Fatalf("session is not refreshable: token=%q refreshToken=%q trusted=%v", inst.Token, inst.RefreshToken, inst.OAuthEndpointsTrusted())
	}

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	got, _ := Load()
	if inst := got.Instances["work"]; inst.OAuthVersion == "" || inst.RefreshToken != "rt-1" || !got.RefreshTokenInKeyring("work") || !inst.OAuthEndpointsTrusted() {
		t.Fatalf("session was not bound in the keyring: %+v", inst)
	}
	if data, _ := os.ReadFile(path); strings.Contains(string(data), "rt-1") {
		t.Fatalf("refresh token left in the file after it moved to the keyring: %s", data)
	}
}

// A session whose keyring entry cannot be read is reported as unreadable; one whose entry is gone is not.
func TestOAuthSessionUnreadable(t *testing.T) {
	isolate(t)
	gokeyring.MockInit()
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	cfg, _ := Load()
	cfg.AddInstanceOAuth("work", "https://work.example.com", &OAuthSession{AccessToken: "at", RefreshToken: "rt", ExpiresAt: 1234,
		ClientID: "cli", TokenEndpoint: "https://work.example.com/hub/api/rest/oauth2/token"})
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	version := cfg.Instances["work"].OAuthVersion
	unreadable := func() bool {
		t.Helper()
		got, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		inst := got.Instances["work"]
		return inst.OAuthSessionUnreadable()
	}

	if unreadable() {
		t.Fatal("a readable session is reported as unreadable")
	}
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "1")
	if !unreadable() {
		t.Fatal("a session behind an unavailable keyring is not reported as unreadable")
	}
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "")
	if err := gokeyring.Delete(keyringOAuthService, oauthAccount("work", version)); err != nil {
		t.Fatal(err)
	}
	if unreadable() {
		t.Fatal("a session whose keyring entry is gone is reported as unreadable")
	}
}
