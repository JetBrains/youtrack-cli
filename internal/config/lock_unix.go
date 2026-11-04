//go:build !windows

package config

import (
	"os"

	"golang.org/x/sys/unix"
)

// lockFile blocks until this process holds an exclusive lock on f; closing f releases it.
func lockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}
