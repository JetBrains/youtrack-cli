//go:build concurrency

package config

import (
	"sync"
	"testing"
)

// Concurrent read-modify-write must neither fail nor lose an update.
func TestUpdateConcurrent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("YOUTRACK_DISABLE_KEYRING", "1")
	const n = 50
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			if _, err := Update(func(c *Config) error { c.LastUpdateCheck++; return nil }); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.LastUpdateCheck != n {
		t.Errorf("LastUpdateCheck = %d, want %d", c.LastUpdateCheck, n)
	}
}
