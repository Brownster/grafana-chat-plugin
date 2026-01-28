package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sabio/grafana-sm3-chat-plugin/pkg/agent"
	"github.com/sabio/grafana-sm3-chat-plugin/pkg/mcp"
)

// Make sure Plugin implements required interfaces
var (
	_ backend.CallResourceHandler = (*Plugin)(nil)
)

// Plugin is the main plugin struct that manages instances
type Plugin struct {
	mu        sync.RWMutex
	instances map[int64]*Instance
}

// Instance represents a plugin instance for a specific data source
type Instance struct {
	agentManager *agent.Manager
	mcpClients   map[string]*mcp.Client
	settings     *PluginSettings
}

// NewPlugin creates a new Plugin
func NewPlugin() *Plugin {
	return &Plugin{
		instances: make(map[int64]*Instance),
	}
}

// CallResource handles HTTP requests to plugin resources
func (p *Plugin) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	log.DefaultLogger.Info("CallResource", "path", req.Path, "method", req.Method)

	// Get or create instance
	instance, err := p.getInstance(ctx, req.PluginContext)
	if err != nil {
		return p.sendError(sender, 500, fmt.Sprintf("Failed to get plugin instance: %v", err))
	}

	// Route to appropriate handler
	switch req.Path {
	case "chat":
		return instance.handleChat(ctx, req, sender)
	case "chat-stream":
		return instance.handleChatStream(ctx, req, sender)
	case "health":
		return instance.handleHealth(ctx, req, sender)
	default:
		return p.sendError(sender, 404, "Not found")
	}
}

// getInstance gets or creates an instance for the given plugin context
func (p *Plugin) getInstance(ctx context.Context, pluginCtx backend.PluginContext) (*Instance, error) {
	// Use OrgID as the instance key since this is a panel plugin
	instanceID := pluginCtx.OrgID

	// Check if instance already exists
	p.mu.RLock()
	instance, exists := p.instances[instanceID]
	p.mu.RUnlock()

	if exists {
		return instance, nil
	}

	// Create new instance
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if instance, exists = p.instances[instanceID]; exists {
		return instance, nil
	}

	// Create new instance
	instance, err := p.createInstance(ctx, pluginCtx)
	if err != nil {
		return nil, err
	}

	p.instances[instanceID] = instance
	return instance, nil
}

// createInstance creates a new plugin instance
func (p *Plugin) createInstance(ctx context.Context, pluginCtx backend.PluginContext) (*Instance, error) {
	log.DefaultLogger.Info("Creating new plugin instance", "org_id", pluginCtx.OrgID)

	// Get settings from AppInstanceSettings if available, otherwise use DataSourceInstanceSettings
	var jsonData []byte
	var decryptedSecrets map[string]string

	if pluginCtx.AppInstanceSettings != nil {
		jsonData = pluginCtx.AppInstanceSettings.JSONData
		decryptedSecrets = pluginCtx.AppInstanceSettings.DecryptedSecureJSONData
	} else if pluginCtx.DataSourceInstanceSettings != nil {
		jsonData = pluginCtx.DataSourceInstanceSettings.JSONData
		decryptedSecrets = pluginCtx.DataSourceInstanceSettings.DecryptedSecureJSONData
	}

	// Parse plugin settings
	pluginSettings, err := LoadSettings(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	// Validate settings
	if err := pluginSettings.Validate(); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	// Get decrypted secrets (API keys)
	apiKey := pluginSettings.OpenAIAPIKey
	if decryptedSecrets != nil {
		if decrypted := decryptedSecrets["openai_api_key"]; decrypted != "" {
			apiKey = decrypted
		}
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

// handleHealth returns health status
func (i *Instance) handleHealth(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	const healthTimeout = 3 * time.Second

	overallStatus := "healthy"
	response := map[string]interface{}{
		"status":      "healthy",
		"mcp_servers": map[string]map[string]interface{}{},
	}

	servers := response["mcp_servers"].(map[string]map[string]interface{})
	for serverType, client := range i.mcpClients {
		healthCtx, cancel := context.WithTimeout(ctx, healthTimeout)
		err := client.Health(healthCtx)
		cancel()

		if err != nil {
			overallStatus = "unhealthy"
			servers[serverType] = map[string]interface{}{
				"ok":    false,
				"error": err.Error(),
			}
			continue
		}

		servers[serverType] = map[string]interface{}{
			"ok": true,
		}
	}

	response["status"] = overallStatus
	statusCode := 200
	if overallStatus != "healthy" {
		statusCode = 503
	}

	return i.sendJSON(sender, statusCode, response)
}

// sendJSON sends a JSON response
func (i *Instance) sendJSON(sender backend.CallResourceResponseSender, status int, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return i.sendError(sender, 500, fmt.Sprintf("Failed to marshal JSON: %v", err))
	}

	return sender.Send(&backend.CallResourceResponse{
		Status:  status,
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    body,
	})
}

// sendError sends an error response
func (i *Instance) sendError(sender backend.CallResourceResponseSender, status int, message string) error {
	return i.sendJSON(sender, status, map[string]string{"error": message})
}

// sendError on Plugin for use before instance is available
func (p *Plugin) sendError(sender backend.CallResourceResponseSender, status int, message string) error {
	body, _ := json.Marshal(map[string]string{"error": message})
	return sender.Send(&backend.CallResourceResponse{
		Status:  status,
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    body,
	})
}
