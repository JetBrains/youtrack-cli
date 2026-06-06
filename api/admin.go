package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetGlobalSettings() (*GlobalSettings, error) {
	var result GlobalSettings
	err := c.Get(pathAdminGlobalSettings+"?fields=id,baseUrl,maxUploadFileSize", &result)
	return &result, err
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

func (c *Client) ListVCSServers() ([]VCSServer, error) {
	var result []VCSServer
	err := c.Get(pathAdminVCSServers+"?fields=id,type,url&$top=-1", &result)
	return result, err
}

func (c *Client) DeleteVCSServer(id string) error {
	return c.Delete(fmt.Sprintf(pathAdminVCSServer, url.PathEscape(id)))
}

func (c *Client) GetVCSConfigurations(serverID string) ([]map[string]any, error) {
	var result []map[string]any
	path := fmt.Sprintf(pathAdminVCSServerProcs+"?fields=id,path&$top=-1", url.PathEscape(serverID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) ListVCSProcessors() ([]VCSProcessor, error) {
	var result []VCSProcessor
	err := c.Get(pathAdminVCSProcessors+"?fields=id,path,enabled,project(id,shortName)&$top=-1", &result)
	return result, err
}

func (c *Client) DeleteVCSProcessor(id string) error {
	return c.Delete(fmt.Sprintf(pathAdminVCSProcessor, url.PathEscape(id)))
}

type AddMailboxServerRequest struct {
	Protocol string `json:"-"`
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (c *Client) ListMailboxServers() ([]MailboxServer, error) {
	var result []MailboxServer
	err := c.Get(pathAdminMailboxes+"?fields=id,login,host,port&$top=-1", &result)
	return result, err
}

func (c *Client) AddMailboxServer(req AddMailboxServerRequest) (*MailboxServer, error) {
	var result MailboxServer
	body := map[string]any{
		"host":         req.Host,
		"login":        req.Login,
		"password":     req.Password,
		"mailProtocol": map[string]any{"id": req.Protocol},
	}
	if req.Port != 0 {
		body["port"] = req.Port
	}
	err := c.Post(pathAdminMailboxes+"?fields=id,login,host,port", body, &result)
	return &result, err
}

func (c *Client) DeleteMailboxServer(id string) error {
	return c.Delete(fmt.Sprintf(pathAdminMailbox, url.PathEscape(id)))
}

func (c *Client) AddMailboxRule(mailboxID, projectID, name string) error {
	return c.Post(pathAdminMailboxRules, map[string]any{
		"mailbox":  map[string]any{"id": mailboxID},
		"project":  map[string]any{"id": projectID},
		"name":     name,
		"disabled": false,
	}, nil)
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
