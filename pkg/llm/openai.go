package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"
)

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	Type      string                 `json:"type"`
	Message   string                 `json:"message,omitempty"`
	Tool      string                 `json:"tool,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Result    interface{}            `json:"result,omitempty"`
}

// OpenAIClient wraps the OpenAI API client
type OpenAIClient struct {
	client *openai.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, errors.New("OpenAI API key is required")
	}

	client := openai.NewClient(apiKey)

	return &OpenAIClient{
		client: client,
	}, nil
}

// Chat performs a non-streaming chat completion
func (c *OpenAIClient) Chat(ctx context.Context, messages []openai.ChatCompletionMessage, tools []openai.Tool) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:    openai.GPT4,
		Messages: messages,
		Tools:    tools,
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// StreamChat performs a streaming chat completion
func (c *OpenAIClient) StreamChat(ctx context.Context, messages []openai.ChatCompletionMessage, tools []openai.Tool) (<-chan StreamChunk, error) {
	chunks := make(chan StreamChunk, 100)

	req := openai.ChatCompletionRequest{
		Model:    openai.GPT4,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		close(chunks)
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Start streaming in background goroutine
	go func() {
		defer close(chunks)
		defer stream.Close()

		// Send start event
		chunks <- StreamChunk{Type: "start"}

		var fullContent string
		var toolCalls []openai.ToolCall

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				chunks <- StreamChunk{
					Type:    "error",
					Message: fmt.Sprintf("Stream error: %v", err),
				}
				return
			}

			if len(response.Choices) == 0 {
				continue
			}

			delta := response.Choices[0].Delta

			// Handle content tokens
			if delta.Content != "" {
				fullContent += delta.Content
				chunks <- StreamChunk{
					Type:    "token",
					Message: delta.Content,
				}
			}

			// Handle tool calls
			if len(delta.ToolCalls) > 0 {
				for _, tc := range delta.ToolCalls {
					// Accumulate tool calls
					if tc.Index != nil {
						idx := *tc.Index
						for len(toolCalls) <= idx {
							toolCalls = append(toolCalls, openai.ToolCall{})
						}

						if tc.ID != "" {
							toolCalls[idx].ID = tc.ID
						}
						if tc.Type != "" {
							toolCalls[idx].Type = tc.Type
						}
						if tc.Function.Name != "" {
							toolCalls[idx].Function.Name = tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							toolCalls[idx].Function.Arguments += tc.Function.Arguments
						}
					}
				}
			}
		}

		// Process tool calls if any
		if len(toolCalls) > 0 {
			for _, tc := range toolCalls {
				if tc.Function.Name == "" {
					continue
				}

				// Parse arguments
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					chunks <- StreamChunk{
						Type:    "error",
						Message: fmt.Sprintf("Failed to parse tool arguments: %v", err),
					}
					continue
				}

				// Send tool call event
				// Note: Actual tool execution happens in the plugin layer where MCP clients are available
				chunks <- StreamChunk{
					Type:      "tool",
					Tool:      tc.Function.Name,
					Arguments: args,
				}
			}
		}

		// Send completion event
		chunks <- StreamChunk{
			Type:    "complete",
			Message: fullContent,
		}

		// Send done event
		chunks <- StreamChunk{Type: "done"}
	}()

	return chunks, nil
}

// ExecuteToolCall executes a tool call (placeholder - actual execution done by MCP clients)
func ExecuteToolCall(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	// This is a placeholder. Actual tool execution happens in the plugin layer
	// where MCP clients are available
	return nil, errors.New("tool execution not implemented in LLM layer")
}
