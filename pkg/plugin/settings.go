package plugin

import (
	"encoding/json"
	"fmt"
)

// PluginSettings holds the plugin configuration
type PluginSettings struct {
	OpenAIAPIKey       string `json:"openai_api_key"`
	GrafanaMCPURL      string `json:"grafana_mcp_url"`
	AlertManagerMCPURL string `json:"alertmanager_mcp_url"`
	GenesysMCPURL      string `json:"genesys_mcp_url"`
}

// LoadSettings loads plugin settings from JSON
func LoadSettings(jsonData []byte) (*PluginSettings, error) {
	settings := &PluginSettings{}

	if len(jsonData) == 0 {
		return settings, nil
	}

	if err := json.Unmarshal(jsonData, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return settings, nil
}

// Validate checks if required settings are present
func (s *PluginSettings) Validate() error {
	if s.OpenAIAPIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}

	if s.GrafanaMCPURL == "" && s.AlertManagerMCPURL == "" && s.GenesysMCPURL == "" {
		return fmt.Errorf("at least one MCP server URL must be configured")
	}

	return nil
}

// GetMCPServers returns a map of configured MCP servers
func (s *PluginSettings) GetMCPServers() map[string]string {
	servers := make(map[string]string)

	if s.GrafanaMCPURL != "" {
		servers["grafana"] = s.GrafanaMCPURL
	}

	if s.AlertManagerMCPURL != "" {
		servers["alertmanager"] = s.AlertManagerMCPURL
	}

	if s.GenesysMCPURL != "" {
		servers["genesys"] = s.GenesysMCPURL
	}

	return servers
}
