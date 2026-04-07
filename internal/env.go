package internal

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnvFile reads ~/.youtrack-cli-env and sets unset env vars from it.
// Format: KEY=VALUE one per line. Lines starting with # are ignored.
func LoadEnvFile() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	f, err := os.Open(filepath.Join(home, ".youtrack-cli-env"))
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}
