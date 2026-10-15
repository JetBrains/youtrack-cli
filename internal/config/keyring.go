package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	gokeyring "github.com/zalando/go-keyring"
)

// keyringService is the OS keyring service name for instance tokens; the keyring user is the instance alias.
const keyringService = "youtrack-cli"

// keyringURLService holds, under the same alias, the instance URL a keyring token was saved for, so an edit of the config file cannot point the token at another host.
// It is a separate entry so the token entry keeps the bare token that older CLI versions read.
const keyringURLService = "youtrack-cli-url"

// keyringOAuthService holds, under the alias, an OAuth session's refresh token together with the Hub token endpoint it may be sent to (an oauthSecret),
// so an edit of the config file cannot redirect the keyring tokens to another host. A separate service keeps it from colliding with any alias.
const keyringOAuthService = "youtrack-cli-oauth"

// oauthSecret is the JSON value of a keyringOAuthService entry.
type oauthSecret struct {
	RefreshToken  string `json:"refreshToken,omitempty"`
	TokenEndpoint string `json:"tokenEndpoint,omitempty"`
}

// keyringTimeout guards against keyring backends that hang instead of failing (e.g. a locked keychain awaiting input).
const keyringTimeout = 3 * time.Second

var (
	errKeyringNotFound = errors.New("secret not found in keyring")
	errKeyringDisabled = errors.New("keyring disabled via YOUTRACK_DISABLE_KEYRING")
)

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
	if keyringDisabled() {
		return errKeyringDisabled
	}
	ch := make(chan error, 1)
	go func() {
		ch <- gokeyring.Set(service, alias, secret)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(keyringTimeout):
		return &keyringTimeoutError{op: "set"}
	}
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
	if keyringDisabled() {
		return errKeyringDisabled
	}
	ch := make(chan error, 1)
	go func() {
		ch <- gokeyring.Delete(service, alias)
	}()
	select {
	case err := <-ch:
		if errors.Is(err, gokeyring.ErrNotFound) {
			return errKeyringNotFound
		}
		return err
	case <-time.After(keyringTimeout):
		return &keyringTimeoutError{op: "delete"}
	}
}
