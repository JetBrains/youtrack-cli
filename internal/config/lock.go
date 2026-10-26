package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// lockWait bounds how long a process waits for another one to finish a token refresh. It exceeds the 30s refresh
// request timeout plus the OS keyring calls made under the lock, so a waiter does not give up on a refresh in progress.
const lockWait = 60 * time.Second

const lockPollInterval = 50 * time.Millisecond

// keyringDrainWait bounds how long unlock waits for a timed-out keyring write to finish. With the 30s refresh and one
// keyringTimeout (later mutations fail fast), a holder releases the lock within lockWait, so waiters do not time out first.
const keyringDrainWait = 20 * time.Second

// Lock takes an exclusive advisory lock on the config file's .lock sibling so that concurrent CLI
// processes serialize read-modify-write cycles of the config (notably OAuth refresh, where a lost
// write would drop a rotated refresh token). Call the returned function to release it.
func Lock(ctx context.Context) (unlock func(), err error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("creating config dir: %w", err)
	}
	f, err := os.OpenFile(path+".lock", os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening config lock: %w", err)
	}
	deadline := time.Now().Add(lockWait)
	for {
		ok, err := tryLockFile(f)
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("locking config: %w", err)
		}
		if ok {
			return func() {
				// A keyring write that timed out under this lock may still land; another process must not save credentials before it does.
				awaitKeyringMutations()
				_ = unlockFile(f)
				_ = f.Close()
			}, nil
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("timed out after %v waiting for another youtrack process to release %s.lock", lockWait, path)
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-time.After(lockPollInterval):
		}
	}
}

// awaitKeyringMutations blocks until the keyring writes and deletes started by this process have finished (see keyringMutations).
// ponytail: bounded, so a keyring that never answers cannot hang the CLI; a write that lands after keyringDrainWait can still race.
func awaitKeyringMutations() {
	done := make(chan struct{})
	go func() {
		keyringMutations.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(keyringDrainWait):
	}
}

// Update runs fn on a freshly loaded config under Lock and saves the result, so concurrent writers cannot overwrite each other's changes.
func Update(ctx context.Context, fn func(*Config) error) error {
	unlock, err := Lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	cfg, err := Load()
	if err != nil {
		return err
	}
	if err := fn(cfg); err != nil {
		return err
	}
	return cfg.Save()
}
