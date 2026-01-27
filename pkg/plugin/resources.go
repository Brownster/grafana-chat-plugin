package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// handleChat handles non-streaming chat requests
func (p *SM3Plugin) handleChat(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender, instance *Instance) error {
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

	log.DefaultLogger.Info("Chat request", "session", chatReq.SessionID, "message_length", len(chatReq.Message))

	// Build contextual message
	message := buildContextualMessage(chatReq.Message, chatReq.DashboardContext)

	// Execute chat
	response, err := instance.agentManager.RunChat(ctx, message, chatReq.SessionID)
	if err != nil {
		log.DefaultLogger.Error("Chat failed", "error", err)
		return p.sendError(sender, 500, fmt.Sprintf("Chat failed: %v", err))
	}

	// Send response
	return p.sendJSON(sender, 200, ChatResponse{
		Response:  response,
		SessionID: chatReq.SessionID,
	})
}

// buildContextualMessage injects dashboard context into the user message
func buildContextualMessage(userMessage string, ctx *DashboardContext) string {
	if ctx == nil {
		return userMessage
	}

	var contextParts []string

	contextParts = append(contextParts, "[Dashboard Context]")

	if ctx.Name != "" {
		contextParts = append(contextParts, fmt.Sprintf("Name: %s", ctx.Name))
	}

	if ctx.UID != "" {
		contextParts = append(contextParts, fmt.Sprintf("UID: %s", ctx.UID))
	}

	if ctx.Folder != "" {
		contextParts = append(contextParts, fmt.Sprintf("Folder: %s", ctx.Folder))
	}

	if len(ctx.Tags) > 0 {
		contextParts = append(contextParts, fmt.Sprintf("Tags: %v", ctx.Tags))
	}

	if ctx.TimeRange != nil && len(ctx.TimeRange) > 0 {
		from := ctx.TimeRange["from"]
		to := ctx.TimeRange["to"]
		if from != "" && to != "" {
			contextParts = append(contextParts, fmt.Sprintf("Time Range: %s to %s", from, to))
		}
	}

	if len(contextParts) > 1 {
		contextStr := strings.Join(contextParts, "\n")
		return fmt.Sprintf("%s\n\n%s", contextStr, userMessage)
	}

	return userMessage
}
