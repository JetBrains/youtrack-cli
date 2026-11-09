package config

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	gokeyring "github.com/zalando/go-keyring"
)

// keyringService is the OS keyring service name for instance tokens; the keyring user is the instance alias.
const keyringService = "youtrack-cli"

// keyringURLService holds, under the same alias, the instance URL a keyring token was saved for, so an edit of the config file cannot point the token at another host.
// It is a separate entry so the token entry keeps the bare token that older CLI versions read.
const keyringURLService = "youtrack-cli-url"

// keyringOAuthService holds, under the alias, an OAuth session's refresh token together with the Hub token endpoint it may be sent to
// and the YouTrack URL that may receive the refreshed access tokens (an oauthSecret), so an edit of the config file cannot redirect the tokens to another host. A separate service keeps it from colliding with any alias.
const keyringOAuthService = "youtrack-cli-oauth"

// oauthSecret is the JSON value of a keyringOAuthService entry.
type oauthSecret struct {
	RefreshToken  string `json:"refreshToken,omitempty"`
	TokenEndpoint string `json:"tokenEndpoint,omitempty"`
	URL           string `json:"url,omitempty"`
	SessionID     string `json:"sessionId,omitempty"`
}

// keyringTimeout guards against keyring backends that hang instead of failing (e.g. a locked keychain awaiting input).
const keyringTimeout = 3 * time.Second

var (
	errKeyringNotFound = errors.New("secret not found in keyring")
	errKeyringDisabled = errors.New("keyring disabled via YOUTRACK_DISABLE_KEYRING")
	errKeyringStalled  = errors.New("keyring skipped: an earlier keyring write has not finished")
)

// A timed-out write or delete cannot be canceled (on macOS it is a `security` subprocess that outlives us) and may still land.
// keyringMutations tracks those operations so the config lock is held until they finish, and keyringStalled makes later
// mutations fail fast, so an older value can never land after a newer one.
var (
	keyringMutations sync.WaitGroup
	keyringStalled   atomic.Bool
)

// keyringMutate runs a keyring write or delete with keyringTimeout, tracking it in keyringMutations until it really finishes.
func keyringMutate(op string, fn func() error) error {
	if keyringDisabled() {
		return errKeyringDisabled
	}
	if keyringStalled.Load() {
		return errKeyringStalled
	}
	ch := make(chan error, 1)
	keyringMutations.Add(1)
	go func() {
		defer keyringMutations.Done()
		ch <- fn()
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(keyringTimeout):
		keyringStalled.Store(true)
		return &keyringTimeoutError{op: op}
	}
}

// keyringDisabled reports whether YOUTRACK_DISABLE_KEYRING opts out of the keyring (used by the E2E suite and plain-file users).
func keyringDisabled() bool {
	return os.Getenv("YOUTRACK_DISABLE_KEYRING") != ""
}

type keyringTimeoutError struct {
	op string
}

func (e *keyringTimeoutError) Error() string {
	return fmt.Sprintf("keyring %s timed out after %v", e.op, keyringTimeout)
}

func keyringSet(service, alias, secret string) error {
	return keyringMutate("set", func() error { return gokeyring.Set(service, alias, secret) })
}

func keyringGet(service, alias string) (string, error) {
	if keyringDisabled() {
		return "", errKeyringDisabled
	}
	type result struct {
		val string
		err error
	}
	ch := make(chan result, 1)
	go func() {
		val, err := gokeyring.Get(service, alias)
		ch <- result{val, err}
	}()
	select {
	case r := <-ch:
		if errors.Is(r.err, gokeyring.ErrNotFound) {
			return "", errKeyringNotFound
		}
		return r.val, r.err
	case <-time.After(keyringTimeout):
		return "", &keyringTimeoutError{op: "get"}
	}
}

func keyringDelete(service, alias string) error {
	err := keyringMutate("delete", func() error { return gokeyring.Delete(service, alias) })
	if errors.Is(err, gokeyring.ErrNotFound) {
		return errKeyringNotFound
	}
	return err
}
