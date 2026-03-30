package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetGlobalSettings() (*GlobalSettings, error) {
	var result GlobalSettings
	return &result, c.Get("/api/admin/globalSettings?fields=id,baseUrl,maxUploadFileSize", &result)
}

func (c *Client) EnableFeatureFlag(feature string, enabled bool) error {
	path := fmt.Sprintf("/api/admin/featureFlags/%s", url.PathEscape(feature))
	return c.Post(path, map[string]any{"id": feature, "enabled": enabled}, nil)
}

func (c *Client) SetLicense(username, license string) error {
	return c.Post("/api/admin/globalSettings", map[string]any{
		"license": map[string]any{"username": username, "license": license},
	}, nil)
}

func (c *Client) SetEmailSettings(server string, port int, from string) error {
	return c.Post("/api/admin/globalSettings/notificationSettings", map[string]any{
		"emailSettings": map[string]any{
			"mailServerSettings": map[string]any{"host": server, "port": port},
			"from":               from,
		},
	}, nil)
}

func (c *Client) DisableEmailSettings() error {
	return c.Post("/api/admin/globalSettings/notificationSettings", map[string]any{
		"emailSettings": map[string]any{"isEnabled": false},
	}, nil)
}

func (c *Client) SetGlobalBanner(message string, enabled bool) error {
	return c.Post("/api/admin/globalSettings", map[string]any{
		"appearanceSettings": map[string]any{
			"globalBannerText": message, "showGlobalBanner": enabled,
		},
	}, nil)
}

func (c *Client) SetReactionsEnabled(enabled bool) error {
	return c.Post("/api/admin/globalSettings", map[string]any{
		"appearanceSettings": map[string]any{"enableReactions": enabled},
	}, nil)
}

func (c *Client) SwitchJetBrainsAI(enabled bool) error {
	return c.Post("/api/grazie/config", map[string]any{"enabled": enabled}, nil)
}

func (c *Client) SwitchTextCompletion(enabled bool) error {
	return c.Post("/api/grazie/config", map[string]any{"enableTextCompletion": enabled}, nil)
}

func (c *Client) SwitchGrammarCorrection(enabled bool) error {
	return c.Post("/api/grazie/config", map[string]any{"enableSpellChecker": enabled}, nil)
}

func (c *Client) ListVCSServers() ([]VCSServer, error) {
	var result []VCSServer
	return result, c.Get("/api/admin/integrations/vcsHostingServers?fields=id,type,url&$top=-1", &result)
}

func (c *Client) DeleteVCSServer(id string) error {
	return c.Delete(fmt.Sprintf("/api/admin/integrations/vcsHostingServers/%s", url.PathEscape(id)))
}

func (c *Client) GetVCSConfigurations(serverID string) ([]map[string]any, error) {
	var result []map[string]any
	path := fmt.Sprintf("/api/admin/integrations/vcsHostingServers/%s/changesProcessors?fields=id,path&$top=-1", url.PathEscape(serverID))
	return result, c.Get(path, &result)
}

func (c *Client) ListVCSProcessors() ([]VCSProcessor, error) {
	var result []VCSProcessor
	return result, c.Get("/api/admin/integrations/vcsHostingProcessors?fields=id,path,enabled,project(id,shortName)&$top=-1", &result)
}

func (c *Client) DeleteVCSProcessor(id string) error {
	return c.Delete(fmt.Sprintf("/api/admin/integrations/vcsHostingProcessors/%s", url.PathEscape(id)))
}

type AddMailboxServerRequest struct {
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (c *Client) ListMailboxServers() ([]MailboxServer, error) {
	var result []MailboxServer
	return result, c.Get("/api/admin/integrations/mailbox/mailboxes?fields=id,login,host,port&$top=-1", &result)
}

func (c *Client) AddMailboxServer(req AddMailboxServerRequest) (*MailboxServer, error) {
	var result MailboxServer
	return &result, c.Post("/api/admin/integrations/mailbox/mailboxes?fields=id,login,host,port", req, &result)
}

func (c *Client) DeleteMailboxServer(id string) error {
	return c.Delete(fmt.Sprintf("/api/admin/integrations/mailbox/mailboxes/%s", url.PathEscape(id)))
}

func (c *Client) AddMailboxRule(mailboxID, projectID, name string) error {
	return c.Post("/api/admin/integrations/mailbox/rules", map[string]any{
		"mailbox":  map[string]any{"id": mailboxID},
		"project":  map[string]any{"id": projectID},
		"name":     name,
		"disabled": false,
	}, nil)
}

func (c *Client) GetFrontendConfig() (map[string]any, error) {
	var result map[string]any
	return result, c.Get("/api/config?fields=ring(url),mobile(serviceId),version,year,buildNumber", &result)
}

func (c *Client) ListPlugins() ([]Plugin, error) {
	var result []Plugin
	return result, c.Get("/api/admin/pluginUsages?fields=id,enabled,state&$top=-1", &result)
}

func (c *Client) DeletePlugin(id string) error {
	return c.Delete(fmt.Sprintf("/api/admin/pluginUsages/%s", url.PathEscape(id)))
}
