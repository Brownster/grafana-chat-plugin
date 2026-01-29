package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/grafana/grafana-llm-app/llmclient"
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

// LLMClient wraps the Grafana LLM App client
type LLMClient struct {
	provider llmclient.LLMProvider
}

// NewLLMClient creates a new LLM client that routes through Grafana LLM App
func NewLLMClient(grafanaURL, grafanaAPIKey string) (*LLMClient, error) {
	if grafanaURL == "" {
		return nil, errors.New("Grafana URL is required")
	}
	if grafanaAPIKey == "" {
		return nil, errors.New("Grafana API key is required")
	}

	provider := llmclient.NewLLMProvider(grafanaURL, grafanaAPIKey)

	return &LLMClient{
		provider: provider,
	}, nil
}

// Enabled checks if the Grafana LLM App is configured and available
func (c *LLMClient) Enabled(ctx context.Context) (bool, error) {
	return c.provider.Enabled(ctx)
}

// Chat performs a non-streaming chat completion via Grafana LLM App
func (c *LLMClient) Chat(ctx context.Context, messages []openai.ChatCompletionMessage, tools []openai.Tool) (string, error) {
	req := llmclient.ChatCompletionRequest{
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: messages,
			Tools:    tools,
		},
		Model: llmclient.ModelLarge,
	}

	resp, err := c.provider.ChatCompletions(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

// StreamChat performs a streaming chat completion via Grafana LLM App
func (c *LLMClient) StreamChat(ctx context.Context, messages []openai.ChatCompletionMessage, tools []openai.Tool) (<-chan StreamChunk, error) {
	chunks := make(chan StreamChunk, 100)

	req := llmclient.ChatCompletionRequest{
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: messages,
			Tools:    tools,
			Stream:   true,
		},
		Model: llmclient.ModelLarge,
	}

	stream, err := c.provider.ChatCompletionsStream(ctx, req)
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
