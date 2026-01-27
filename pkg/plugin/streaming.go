package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sabio/grafana-sm3-chat-plugin/pkg/llm"
	"github.com/sabio/grafana-sm3-chat-plugin/pkg/mcp"
)

// handleChatStream handles streaming chat requests with SSE
func (p *SM3Plugin) handleChatStream(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender, instance *Instance) error {
	// Parse request body
	var chatReq ChatRequest
	if err := json.Unmarshal(req.Body, &chatReq); err != nil {
		return p.sendError(sender, 400, fmt.Sprintf("Invalid request body: %v", err))
	}

	// Validate request
	if chatReq.Message == "" {
		return p.sendError(sender, 400, "Message is required")
	}

	// Generate session ID if not provided
	if chatReq.SessionID == "" {
		chatReq.SessionID = fmt.Sprintf("session-%d", req.PluginContext.DataSourceInstanceSettings.ID)
	}

	log.DefaultLogger.Info("Chat stream request", "session", chatReq.SessionID, "message_length", len(chatReq.Message))

	// Build contextual message
	message := buildContextualMessage(chatReq.Message, chatReq.DashboardContext)

	// Start streaming
	chunks, err := instance.agentManager.RunChatStream(ctx, message, chatReq.SessionID)
	if err != nil {
		log.DefaultLogger.Error("Stream failed to start", "error", err)
		return p.sendError(sender, 500, fmt.Sprintf("Failed to start stream: %v", err))
	}

	// Set SSE headers
	if err := sender.Send(&backend.CallResourceResponse{
		Status:  200,
		Headers: map[string][]string{"Content-Type": {"text/event-stream"}},
	}); err != nil {
		return err
	}

	// Stream chunks
	var fullResponse string
	for chunk := range chunks {
		// Handle tool execution
		if chunk.Type == "tool" {
			log.DefaultLogger.Info("Tool call", "tool", chunk.Tool)

			// Execute tool via MCP
			result, err := p.executeTool(ctx, instance, chunk.Tool, chunk.Arguments)
			if err != nil {
				log.DefaultLogger.Error("Tool execution failed", "tool", chunk.Tool, "error", err)
				chunk.Result = fmt.Sprintf("Error: %v", err)
			} else {
				chunk.Result = result
			}
		}

		// Accumulate response content
		if chunk.Type == "token" {
			fullResponse += chunk.Message
		}

		// Send chunk as SSE
		if err := p.sendSSE(sender, chunk); err != nil {
			log.DefaultLogger.Error("Failed to send SSE", "error", err)
			return err
		}
	}

	// Add final response to memory
	instance.agentManager.AddAssistantResponse(chatReq.SessionID, fullResponse)

	return nil
}

// executeTool executes a tool call via MCP client
func (p *SM3Plugin) executeTool(ctx context.Context, instance *Instance, toolName string, args map[string]interface{}) (interface{}, error) {
	// Determine which MCP client to use based on tool prefix
	var client *mcp.Client
	var found bool

	// Check for prefixed tools (e.g., alertmanager__list_alerts)
	for serverType, mcpClient := range instance.mcpClients {
		if serverType != "grafana" {
			prefix := serverType + "__"
			if len(toolName) > len(prefix) && toolName[:len(prefix)] == prefix {
				client = mcpClient
				found = true
				break
			}
		}
	}

	// If no prefix found, use Grafana client (default)
	if !found {
		client = instance.mcpClients["grafana"]
		if client == nil {
			return nil, fmt.Errorf("Grafana MCP client not available")
		}
	}

	// Execute tool
	result, err := client.InvokeTool(ctx, toolName, args)
	if err != nil {
		return nil, err
	}

	// Format result for LLM
	return mcp.FormatToolResult(result), nil
}

// sendSSE sends a chunk as a Server-Sent Event
func (p *SM3Plugin) sendSSE(sender backend.CallResourceResponseSender, chunk llm.StreamChunk) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk: %w", err)
	}

	sseData := fmt.Sprintf("data: %s\n\n", string(data))

	return sender.Send(&backend.CallResourceResponse{
		Body: []byte(sseData),
	})
}
