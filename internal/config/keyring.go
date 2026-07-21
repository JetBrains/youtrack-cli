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

func keyringSet(alias, token string) error {
	if keyringDisabled() {
		return errKeyringDisabled
	}
	ch := make(chan error, 1)
	go func() {
		ch <- gokeyring.Set(keyringService, alias, token)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(keyringTimeout):
		return &keyringTimeoutError{op: "set"}
	}
}

func keyringGet(alias string) (string, error) {
	if keyringDisabled() {
		return "", errKeyringDisabled
	}
	type result struct {
		val string
		err error
	}
	ch := make(chan result, 1)
	go func() {
		val, err := gokeyring.Get(keyringService, alias)
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

func keyringDelete(alias string) error {
	if keyringDisabled() {
		return errKeyringDisabled
	}
	ch := make(chan error, 1)
	go func() {
		ch <- gokeyring.Delete(keyringService, alias)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(keyringTimeout):
		return &keyringTimeoutError{op: "delete"}
	}
}
