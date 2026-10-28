package api

import (
	"net/url"
	"strings"
)

// withMuteParam sets muteUpdateNotifications=true, replacing conflicting or duplicate values.
func withMuteParam(path string) string {
	base, query, _ := strings.Cut(path, "?")
	params := make([]string, 0)
	for _, param := range strings.Split(query, "&") {
		key, _, _ := strings.Cut(param, "=")
		decoded, err := url.QueryUnescape(key)
		if param != "" && (err != nil || decoded != "muteUpdateNotifications") {
			params = append(params, param)
		}
	}
	params = append(params, "muteUpdateNotifications=true")
	return base + "?" + strings.Join(params, "&")
}

// applyMute adds the mute parameter to every write when MuteNotifications is set.
func (c *Client) applyMute(method, path string) string {
	if c.MuteNotifications && (method == "POST" || method == "PUT" || method == "DELETE") {
		return withMuteParam(path)
	}
	return path
}
