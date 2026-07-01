package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/JetBrains/youtrack-cli/internal/config"
)

const (
	updateCheckInterval = 24 * time.Hour
	updateCheckTimeout  = 3 * time.Second
	updateCheckURL      = "https://api.github.com/repos/JetBrains/youtrack-cli/releases/latest"
)

// CheckForUpdateAsync starts a background goroutine that checks for a newer
// GitHub release. Returns a channel that receives at most one notice string
// then closes. The caller should drain the channel after the command finishes.
func CheckForUpdateAsync(currentVersion string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		if notice := checkForUpdate(currentVersion); notice != "" {
			ch <- notice
		}
	}()
	return ch
}

func checkForUpdate(currentVersion string) string {
	if currentVersion == "dev" || currentVersion == "unknown" {
		return ""
	}

	cfg, err := config.Load()
	if err != nil {
		return ""
	}

	now := time.Now().Unix()
	if now-cfg.LastUpdateCheck < int64(updateCheckInterval.Seconds()) {
		return ""
	}

	client := &http.Client{Timeout: updateCheckTimeout}
	resp, err := client.Get(updateCheckURL)
	if err != nil {
		cfg.LastUpdateCheck = now
		_ = cfg.Save()
		return ""
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil || release.TagName == "" {
		return ""
	}

	cfg.LastUpdateCheck = now
	_ = cfg.Save()

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")
	if latest == current {
		return ""
	}
	return fmt.Sprintf(
		"ℹ  youtrack %s is available (you have %s)  →  youtrack update",
		release.TagName, currentVersion,
	)
}
