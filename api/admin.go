package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetGlobalSettings() (*GlobalSettings, error) {
	var result GlobalSettings
	err := c.Get(pathAdminGlobalSettings+"?fields=id,systemSettings(id,baseUrl,maxUploadFileSize,maxExportItems,administratorEmail,allowStatisticsCollection,isApplicationReadOnly)", &result)
	return &result, err
}

func (c *Client) EnableFeatureFlag(feature string, enabled bool) error {
	path := fmt.Sprintf(pathAdminFeatureFlag, url.PathEscape(feature))
	return c.Post(path, map[string]any{"id": feature, "enabled": enabled}, nil)
}

func (c *Client) GetInstanceVersion() (*InstanceVersion, error) {
	var result InstanceVersion
	err := c.Get(pathConfig+"?fields=version,build", &result)
	return &result, err
}

func (c *Client) ListPlugins() ([]Plugin, error) {
	var result []Plugin
	err := c.Get(pathAdminPluginUsages+"?fields=id,enabled,state&$top=500", &result)
	return result, err
}

func (c *Client) DeletePlugin(id string) error {
	return c.Delete(fmt.Sprintf(pathAdminPluginUsage, url.PathEscape(id)))
}
