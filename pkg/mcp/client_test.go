package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		serverType string
		wantErr    bool
	}{
		{
			name:       "valid grafana client",
			url:        "http://localhost:8888",
			serverType: "grafana",
			wantErr:    false,
		},
		{
			name:       "valid alertmanager client",
			url:        "http://localhost:9300",
			serverType: "alertmanager",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.url, tt.serverType)
			if client == nil {
				t.Error("NewClient returned nil")
			}
			if client.url != tt.url {
				t.Errorf("NewClient url = %v, want %v", client.url, tt.url)
			}
			if client.serverType != tt.serverType {
				t.Errorf("NewClient serverType = %v, want %v", client.serverType, tt.serverType)
			}
		})
	}
}

func TestConnect(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful connection",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "server unavailable",
			statusCode: http.StatusServiceUnavailable,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					w.WriteHeader(tt.statusCode)
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "grafana")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := client.Connect(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiscoverTools(t *testing.T) {
	// Mock MCP tools/list response
	mockResponse := `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"tools": [
				{
					"name": "search_dashboards",
					"description": "Search for dashboards",
					"inputSchema": {
						"type": "object",
						"properties": {
							"query": {"type": "string"}
						}
					}
				},
				{
					"name": "query_prometheus",
					"description": "Query Prometheus",
					"inputSchema": {
						"type": "object",
						"properties": {
							"query": {"type": "string"}
						}
					}
				}
			]
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient(server.URL, "grafana")
	ctx := context.Background()

	tools, err := client.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("DiscoverTools() error = %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("DiscoverTools() returned %d tools, want 2", len(tools))
	}

	// Check first tool
	if tools[0].Name != "search_dashboards" {
		t.Errorf("First tool name = %v, want search_dashboards", tools[0].Name)
	}

	// Test caching - should return same tools without hitting server again
	tools2, err := client.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("DiscoverTools() cached error = %v", err)
	}
	if len(tools2) != len(tools) {
		t.Error("DiscoverTools() cache returned different number of tools")
	}
}

func TestToolPrefixing(t *testing.T) {
	mockResponse := `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"tools": [
				{"name": "list_alerts", "description": "List alerts", "inputSchema": {}}
			]
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	// Test with alertmanager - should prefix tools
	client := NewClient(server.URL, "alertmanager")
	ctx := context.Background()

	tools, err := client.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("DiscoverTools() error = %v", err)
	}

	if tools[0].Name != "alertmanager__list_alerts" {
		t.Errorf("Tool name = %v, want alertmanager__list_alerts", tools[0].Name)
	}

	// Test with grafana - should NOT prefix tools
	grafanaClient := NewClient(server.URL, "grafana")
	grafanaTools, err := grafanaClient.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("DiscoverTools() error = %v", err)
	}

	if grafanaTools[0].Name != "list_alerts" {
		t.Errorf("Grafana tool name = %v, want list_alerts (no prefix)", grafanaTools[0].Name)
	}
}

func TestParseRelativeTime(t *testing.T) {
	tests := []struct {
		name     string
		relative string
		wantErr  bool
	}{
		{
			name:     "now-1h",
			relative: "now-1h",
			wantErr:  false,
		},
		{
			name:     "now-24h",
			relative: "now-24h",
			wantErr:  false,
		},
		{
			name:     "now-7d",
			relative: "now-7d",
			wantErr:  false,
		},
		{
			name:     "now+1h",
			relative: "now+1h",
			wantErr:  false,
		},
		{
			name:     "invalid format",
			relative: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRelativeTime(tt.relative)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRelativeTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == "" {
				t.Error("parseRelativeTime() returned empty string for valid input")
			}

			// Verify RFC3339 format if successful
			if !tt.wantErr {
				_, err := time.Parse(time.RFC3339, result)
				if err != nil {
					t.Errorf("parseRelativeTime() returned invalid RFC3339 timestamp: %v", result)
				}
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "snake_case to camelCase",
			input: "datasource_uid",
			want:  "datasourceUid",
		},
		{
			name:  "multiple underscores",
			input: "my_long_variable_name",
			want:  "myLongVariableName",
		},
		{
			name:  "no underscores",
			input: "simple",
			want:  "simple",
		},
		{
			name:  "trailing underscore",
			input: "trailing_",
			want:  "trailing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("toCamelCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeArguments(t *testing.T) {
	client := NewClient("http://localhost", "grafana")

	tests := []struct {
		name     string
		toolName string
		args     map[string]interface{}
		want     map[string]interface{}
	}{
		{
			name:     "converts snake_case to camelCase",
			toolName: "update_dashboard",
			args: map[string]interface{}{
				"datasource_uid": "abc123",
				"folder_uid":     "folder1",
			},
			want: map[string]interface{}{
				"datasourceUid": "abc123",
				"folderUid":     "folder1",
			},
		},
		{
			name:     "adds stepSeconds for prometheus range query",
			toolName: "query_prometheus_range",
			args: map[string]interface{}{
				"query": "up",
			},
			want: map[string]interface{}{
				"query":       "up",
				"stepSeconds": 60,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.normalizeArguments(tt.toolName, tt.args)

			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("normalizeArguments() missing key %v", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("normalizeArguments() key %v = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}
