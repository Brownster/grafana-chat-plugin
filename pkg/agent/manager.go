package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/sabio/grafana-sm3-chat-plugin/pkg/llm"
	"github.com/sabio/grafana-sm3-chat-plugin/pkg/mcp"
	"github.com/sashabaranov/go-openai"
)

// Manager handles agent orchestration and LLM interaction
type Manager struct {
	llmClient       *llm.LLMClient
	tools           []openai.Tool
	sessionMemories map[string]*ConversationMemory
	systemPrompt    string
	mu              sync.RWMutex
}

// NewManager creates a new agent manager
func NewManager(llmClient *llm.LLMClient, mcpClients map[string]*mcp.Client, mcpTypes []string) (*Manager, error) {
	// Build system prompt based on available MCP types
	systemPrompt := BuildSystemPrompt(mcpTypes)

	// Convert MCP tools to OpenAI format
	tools := convertMCPToolsToOpenAI(mcpClients)

	return &Manager{
		llmClient:       llmClient,
		tools:           tools,
		sessionMemories: make(map[string]*ConversationMemory),
		systemPrompt:    systemPrompt,
	}, nil
}

// RunChat executes a chat interaction (non-streaming)
func (m *Manager) RunChat(ctx context.Context, userMessage, sessionID string) (string, error) {
	memory := m.getOrCreateMemory(sessionID)

	// Add user message to memory
	memory.AddMessage("user", userMessage)

	// Build messages for API call
	messages := m.buildMessages(memory)

	// Call LLM via Grafana LLM App
	response, err := m.llmClient.Chat(ctx, messages, m.tools)
	if err != nil {
		return "", fmt.Errorf("OpenAI chat failed: %w", err)
	}

	// Add assistant response to memory
	memory.AddMessage("assistant", response)

	return response, nil
}

// RunChatStream executes a streaming chat interaction
func (m *Manager) RunChatStream(ctx context.Context, userMessage, sessionID string) (<-chan llm.StreamChunk, error) {
	memory := m.getOrCreateMemory(sessionID)

	// Add user message to memory
	memory.AddMessage("user", userMessage)

	// Build messages for API call
	messages := m.buildMessages(memory)

	// Start streaming
	return m.llmClient.StreamChat(ctx, messages, m.tools)
}

// getOrCreateMemory retrieves or creates a conversation memory for a session
func (m *Manager) getOrCreateMemory(sessionID string) *ConversationMemory {
	m.mu.Lock()
	defer m.mu.Unlock()

	if mem, ok := m.sessionMemories[sessionID]; ok {
		return mem
	}

	mem := NewConversationMemory()
	m.sessionMemories[sessionID] = mem
	return mem
}

// buildMessages constructs the message array for OpenAI API
func (m *Manager) buildMessages(memory *ConversationMemory) []openai.ChatCompletionMessage {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: m.systemPrompt,
		},
	}

	// Add conversation history
	for _, msg := range memory.GetMessages() {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages
}

// convertMCPToolsToOpenAI converts MCP tools to OpenAI function format
func convertMCPToolsToOpenAI(mcpClients map[string]*mcp.Client) []openai.Tool {
	var tools []openai.Tool

	for _, client := range mcpClients {
		// Discover tools from each client (cached internally)
		mcpTools, err := client.DiscoverTools(context.Background())
		if err != nil {
			continue
		}

		for _, mcpTool := range mcpTools {
			// Convert MCP tool to OpenAI function format
			tool := openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        mcpTool.Name,
					Description: mcpTool.Description,
					Parameters:  mcpTool.InputSchema,
				},
			}
			tools = append(tools, tool)
		}
	}

	return tools
}

// AddAssistantResponse adds an assistant response to session memory
func (m *Manager) AddAssistantResponse(sessionID, response string) {
	memory := m.getOrCreateMemory(sessionID)
	memory.AddMessage("assistant", response)
}

// ClearSession clears the conversation history for a session
func (m *Manager) ClearSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if mem, ok := m.sessionMemories[sessionID]; ok {
		mem.Clear()
	}
}
