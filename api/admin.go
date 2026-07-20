package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetGlobalSettings() (*GlobalSettings, error) {
	var result GlobalSettings
	err := c.Get(pathAdminGlobalSettings+"?fields=id,baseUrl,maxUploadFileSize,systemSettings,notificationSettings,restSettings,appearanceSettings(timeZone,dateFieldFormat),localeSettings", &result)
	return &result, err
}

func (c *Client) GetSystemSettings() (*SystemSettings, error) {
	var result GlobalSettings
	err := c.Get(pathAdminGlobalSettings+"?fields=systemSettings(id,baseUrl,maxUploadFileSize,isAutoUpdateEnabled,statisticsEnabled)", &result)
	if err != nil {
		return nil, err
	}
	return result.SystemSettings, nil
}

func (c *Client) UpdateSystemSettings(baseURL string, maxUploadFileSize int64) (*SystemSettings, error) {
	body := map[string]any{}
	if baseURL != "" {
		body["systemSettings"] = map[string]any{"baseUrl": baseURL}
	}
	if maxUploadFileSize > 0 {
		if body["systemSettings"] == nil {
			body["systemSettings"] = map[string]any{}
		}
		body["systemSettings"].(map[string]any)["maxUploadFileSize"] = maxUploadFileSize
	}
	var result GlobalSettings
	err := c.Post(pathAdminGlobalSettings+"?fields=systemSettings(id,baseUrl,maxUploadFileSize,isAutoUpdateEnabled,statisticsEnabled)", body, &result)
	if err != nil {
		return nil, err
	}
	return result.SystemSettings, nil
}

func (c *Client) EnableFeatureFlag(feature string, enabled bool) error {
	path := fmt.Sprintf(pathAdminFeatureFlag, url.PathEscape(feature))
	return c.Post(path, map[string]any{"id": feature, "enabled": enabled}, nil)
}

func (c *Client) SetLicense(username, license string) error {
	return c.Post(pathAdminGlobalSettings, map[string]any{
		"license": map[string]any{"username": username, "license": license},
	}, nil)
}

func (c *Client) SetEmailSettings(server string, port int, from string) error {
	return c.Post(pathAdminGlobalSettingsNotifications, map[string]any{
		"emailSettings": map[string]any{
			"isEnabled":    true,
			"host":         server,
			"port":         port,
			"from":         from,
			"mailProtocol": "SMTP",
		},
	}, nil)
}

func (c *Client) DisableEmailSettings() error {
	return c.Post(pathAdminGlobalSettingsNotifications, map[string]any{
		"emailSettings": map[string]any{"isEnabled": false},
	}, nil)
}

func (c *Client) SetGlobalBanner(message string, enabled bool) error {
	return c.Post(pathAdminGlobalSettings, map[string]any{
		"appearanceSettings": map[string]any{
			"globalBanner": message, "globalBannerEnabled": enabled,
		},
	}, nil)
}

func (c *Client) SetReactionsEnabled(enabled bool) error {
	return c.Post(pathAdminGlobalSettings, map[string]any{
		"appearanceSettings": map[string]any{"reactionsEnabled": enabled},
	}, nil)
}

func (c *Client) SwitchJetBrainsAI(enabled bool) error {
	return c.Post(pathGrazieConfig, map[string]any{"enabled": enabled}, nil)
}

func (c *Client) SwitchTextCompletion(enabled bool) error {
	return c.Post(pathGrazieConfig, map[string]any{"enableTextCompletion": enabled}, nil)
}

func (c *Client) SwitchGrammarCorrection(enabled bool) error {
	return c.Post(pathGrazieConfig, map[string]any{"enableSpellChecker": enabled}, nil)
}

func (c *Client) GetFrontendConfig() (map[string]any, error) {
	var result map[string]any
	err := c.Get(pathConfig+"?fields=ring(url),mobile(serviceId),version,year,buildNumber", &result)
	return result, err
}

func (c *Client) ListPlugins() ([]Plugin, error) {
	var result []Plugin
	err := c.Get(pathAdminPluginUsages+"?fields=id,enabled,state&$top=-1", &result)
	return result, err
}

func (c *Client) DeletePlugin(id string) error {
	return c.Delete(fmt.Sprintf(pathAdminPluginUsage, url.PathEscape(id)))
}
