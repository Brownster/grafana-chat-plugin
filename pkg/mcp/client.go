package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Client represents an MCP HTTP client
type Client struct {
	url        string
	httpClient *resty.Client
	tools      []Tool
	serverType string
}

// NewClient creates a new MCP client
func NewClient(url string, serverType string) *Client {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)

	return &Client{
		url:        url,
		httpClient: client,
		serverType: serverType,
	}
}

// Connect initializes the MCP session
func (c *Client) Connect(ctx context.Context) error {
	return c.Health(ctx)
}

// Health checks MCP server connectivity
func (c *Client) Health(ctx context.Context) error {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		Get(c.url + "/health")

	if err != nil {
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("MCP server health check failed with status: %d", resp.StatusCode())
	}

	return nil
}

// DiscoverTools fetches available tools from the MCP server
func (c *Client) DiscoverTools(ctx context.Context) ([]Tool, error) {
	if len(c.tools) > 0 {
		return c.tools, nil
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/list",
		}).
		Post(c.url)

	if err != nil {
		return nil, fmt.Errorf("failed to discover tools: %w", err)
	}

	var result struct {
		Result struct {
			Tools []Tool `json:"tools"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools response: %w", err)
	}

	// Prefix non-Grafana tools with server type
	if c.serverType != "grafana" {
		for i := range result.Result.Tools {
			result.Result.Tools[i].Name = fmt.Sprintf("%s__%s", c.serverType, result.Result.Tools[i].Name)
		}
	}

	c.tools = result.Result.Tools
	return c.tools, nil
}

// InvokeTool calls an MCP tool with the given arguments
func (c *Client) InvokeTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	// Remove prefix if present for actual invocation
	actualName := name
	if c.serverType != "grafana" {
		actualName = strings.TrimPrefix(name, c.serverType+"__")
	}

	// Normalize arguments
	normalizedArgs := c.normalizeArguments(actualName, args)

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      time.Now().Unix(),
			"method":  "tools/call",
			"params": map[string]interface{}{
				"name":      actualName,
				"arguments": normalizedArgs,
			},
		}).
		Post(c.url)

	if err != nil {
		return nil, fmt.Errorf("failed to invoke tool %s: %w", name, err)
	}

	var result struct {
		Result struct {
			Content []struct {
				Type string      `json:"type"`
				Text string      `json:"text,omitempty"`
				Data interface{} `json:"data,omitempty"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("tool error: %s", result.Error.Message)
	}

	// Return the first content item
	if len(result.Result.Content) > 0 {
		if result.Result.Content[0].Type == "text" {
			return result.Result.Content[0].Text, nil
		}
		return result.Result.Content[0].Data, nil
	}

	return nil, fmt.Errorf("tool returned no content")
}

// normalizeArguments applies MCP-specific argument transformations
func (c *Client) normalizeArguments(toolName string, args map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{})

	for key, value := range args {
		// Handle relative time strings
		if strVal, ok := value.(string); ok {
			if strings.HasPrefix(strVal, "now-") || strings.HasPrefix(strVal, "now+") {
				if timestamp, err := parseRelativeTime(strVal); err == nil {
					normalized[key] = timestamp
					continue
				}
			}
		}

		// Convert snake_case to camelCase for dashboard fields
		if c.serverType == "grafana" && strings.Contains(key, "_") {
			normalized[toCamelCase(key)] = value
		} else {
			normalized[key] = value
		}
	}

	// Add default stepSeconds for Prometheus range queries
	if strings.Contains(toolName, "prometheus") && strings.Contains(toolName, "range") {
		if _, ok := normalized["stepSeconds"]; !ok {
			normalized["stepSeconds"] = 60
		}
	}

	return normalized
}

// parseRelativeTime converts relative time strings to RFC3339 timestamps
func parseRelativeTime(relative string) (string, error) {
	now := time.Now()

	if strings.HasPrefix(relative, "now-") {
		duration := strings.TrimPrefix(relative, "now-")
		d, err := parseDuration(duration)
		if err != nil {
			return "", err
		}
		return now.Add(-d).Format(time.RFC3339), nil
	}

	if strings.HasPrefix(relative, "now+") {
		duration := strings.TrimPrefix(relative, "now+")
		d, err := parseDuration(duration)
		if err != nil {
			return "", err
		}
		return now.Add(d).Format(time.RFC3339), nil
	}

	return "", fmt.Errorf("invalid relative time format")
}

// parseDuration parses duration strings like "1h", "24h", "7d"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	unit := s[len(s)-1]
	value := s[:len(s)-1]

	var multiplier time.Duration
	switch unit {
	case 's':
		multiplier = time.Second
	case 'm':
		multiplier = time.Minute
	case 'h':
		multiplier = time.Hour
	case 'd':
		multiplier = 24 * time.Hour
	default:
		return time.ParseDuration(s)
	}

	var count int
	_, err := fmt.Sscanf(value, "%d", &count)
	if err != nil {
		return 0, err
	}

	return time.Duration(count) * multiplier, nil
}

// toCamelCase converts snake_case to camelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		return s
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}

	return result
}
