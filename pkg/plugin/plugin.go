package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sabio/grafana-sm3-chat-plugin/pkg/agent"
	"github.com/sabio/grafana-sm3-chat-plugin/pkg/mcp"
)

// Make sure SM3Plugin implements required interfaces
var (
	_ backend.CallResourceHandler   = (*SM3Plugin)(nil)
	_ instancemgmt.InstanceDisposer = (*Instance)(nil)
)

// SM3Plugin is the main plugin struct
type SM3Plugin struct {
	backend.CallResourceHandler
	im instancemgmt.InstanceManager
}

// Instance represents a plugin instance
type Instance struct {
	agentManager *agent.Manager
	mcpClients   map[string]*mcp.Client
	settings     *PluginSettings
}

// NewPlugin creates a new plugin instance
func NewPlugin() *SM3Plugin {
	plugin := &SM3Plugin{}

	// Create instance manager
	plugin.im = instancemgmt.New(plugin.newInstance)

	return plugin
}

// newInstance creates a new plugin instance with MCP connections
func (p *SM3Plugin) newInstance(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	log.DefaultLogger.Info("Creating new plugin instance")

	// Parse plugin settings
	pluginSettings, err := LoadSettings(settings.JSONData)
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	// Validate settings
	if err := pluginSettings.Validate(); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	// Get decrypted secrets (API keys)
	apiKey := pluginSettings.OpenAIAPIKey
	if decrypted := settings.DecryptedSecureJSONData["openai_api_key"]; decrypted != "" {
		apiKey = decrypted
	}

	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not found in settings or secrets")
	}

	// Connect to MCP servers
	mcpClients := make(map[string]*mcp.Client)
	mcpTypes := []string{}

	for serverType, url := range pluginSettings.GetMCPServers() {
		log.DefaultLogger.Info("Connecting to MCP server", "type", serverType, "url", url)

		client := mcp.NewClient(url, serverType)
		if err := client.Connect(ctx); err != nil {
			log.DefaultLogger.Warn("Failed to connect to MCP server", "type", serverType, "error", err)
			continue
		}

		// Discover tools
		tools, err := client.DiscoverTools(ctx)
		if err != nil {
			log.DefaultLogger.Warn("Failed to discover tools", "type", serverType, "error", err)
			continue
		}

		log.DefaultLogger.Info("Discovered tools", "type", serverType, "count", len(tools))
		mcpClients[serverType] = client
		mcpTypes = append(mcpTypes, serverType)
	}

	if len(mcpClients) == 0 {
		return nil, fmt.Errorf("failed to connect to any MCP servers")
	}

	// Initialize agent manager
	log.DefaultLogger.Info("Initializing agent manager", "mcp_types", mcpTypes)
	agentManager, err := agent.NewManager(apiKey, mcpClients, mcpTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent manager: %w", err)
	}

	return &Instance{
		agentManager: agentManager,
		mcpClients:   mcpClients,
		settings:     pluginSettings,
	}, nil
}

// Dispose cleans up plugin instance resources
func (i *Instance) Dispose() {
	log.DefaultLogger.Info("Disposing plugin instance")
	// Clean up resources if needed
}

// CallResource handles HTTP requests to plugin resources
func (p *SM3Plugin) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	log.DefaultLogger.Info("CallResource", "path", req.Path, "method", req.Method)

	// Get plugin instance
	instance, err := p.im.Get(ctx, req.PluginContext)
	if err != nil {
		return p.sendError(sender, 500, fmt.Sprintf("Failed to get plugin instance: %v", err))
	}

	pluginInstance := instance.(*Instance)

	// Route to appropriate handler
	switch req.Path {
	case "chat":
		return p.handleChat(ctx, req, sender, pluginInstance)
	case "chat-stream":
		return p.handleChatStream(ctx, req, sender, pluginInstance)
	case "health":
		return p.handleHealth(ctx, req, sender, pluginInstance)
	default:
		return p.sendError(sender, 404, "Not found")
	}
}

// handleHealth returns health status
func (p *SM3Plugin) handleHealth(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender, instance *Instance) error {
	response := map[string]interface{}{
		"status": "healthy",
		"mcp_servers": func() map[string]bool {
			servers := make(map[string]bool)
			for serverType := range instance.mcpClients {
				servers[serverType] = true
			}
			return servers
		}(),
	}

	return p.sendJSON(sender, 200, response)
}

// sendJSON sends a JSON response
func (p *SM3Plugin) sendJSON(sender backend.CallResourceResponseSender, status int, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return p.sendError(sender, 500, fmt.Sprintf("Failed to marshal JSON: %v", err))
	}

	return sender.Send(&backend.CallResourceResponse{
		Status:  status,
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    body,
	})
}

// sendError sends an error response
func (p *SM3Plugin) sendError(sender backend.CallResourceResponseSender, status int, message string) error {
	return p.sendJSON(sender, status, map[string]string{"error": message})
}
