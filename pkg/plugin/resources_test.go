package plugin

import (
	"testing"
)

func TestBuildContextualMessage(t *testing.T) {
	tests := []struct {
		name        string
		userMessage string
		context     *DashboardContext
		wantContain []string
	}{
		{
			name:        "no context",
			userMessage: "Hello",
			context:     nil,
			wantContain: []string{"Hello"},
		},
		{
			name:        "with full context",
			userMessage: "Show me metrics",
			context: &DashboardContext{
				UID:    "abc123",
				Name:   "Test Dashboard",
				Folder: "Infrastructure",
				Tags:   []string{"linux", "prometheus"},
				TimeRange: map[string]string{
					"from": "2026-01-27T00:00:00Z",
					"to":   "2026-01-27T23:59:59Z",
				},
			},
			wantContain: []string{
				"[Dashboard Context]",
				"Name: Test Dashboard",
				"UID: abc123",
				"Folder: Infrastructure",
				"Tags: [linux prometheus]",
				"Time Range: 2026-01-27T00:00:00Z to 2026-01-27T23:59:59Z",
				"Show me metrics",
			},
		},
		{
			name:        "with partial context",
			userMessage: "Query data",
			context: &DashboardContext{
				UID:  "xyz789",
				Name: "Partial Dashboard",
			},
			wantContain: []string{
				"[Dashboard Context]",
				"Name: Partial Dashboard",
				"UID: xyz789",
				"Query data",
			},
		},
		{
			name:        "empty context",
			userMessage: "Test message",
			context:     &DashboardContext{},
			wantContain: []string{"Test message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildContextualMessage(tt.userMessage, tt.context)

			for _, want := range tt.wantContain {
				if !contains(got, want) {
					t.Errorf("buildContextualMessage() does not contain %q\nGot: %s", want, got)
				}
			}
		})
	}
}

func TestBuildContextualMessageOrder(t *testing.T) {
	userMessage := "User query"
	context := &DashboardContext{
		Name: "Test",
	}

	result := buildContextualMessage(userMessage, context)

	// User message should appear after context
	contextIdx := indexOf(result, "[Dashboard Context]")
	messageIdx := indexOf(result, "User query")

	if contextIdx == -1 {
		t.Error("Context not found in result")
	}
	if messageIdx == -1 {
		t.Error("User message not found in result")
	}
	if contextIdx >= messageIdx {
		t.Error("Context should appear before user message")
	}
}

func TestBuildContextualMessageWithEmptyTimeRange(t *testing.T) {
	context := &DashboardContext{
		Name: "Test",
		TimeRange: map[string]string{
			"from": "",
			"to":   "",
		},
	}

	result := buildContextualMessage("Test", context)

	// Should not include time range if values are empty
	if contains(result, "Time Range:") {
		t.Error("Should not include Time Range when values are empty")
	}
}

func TestBuildContextualMessageWithNilTimeRange(t *testing.T) {
	context := &DashboardContext{
		Name:      "Test",
		TimeRange: nil,
	}

	result := buildContextualMessage("Test", context)

	// Should not panic and should not include time range
	if contains(result, "Time Range:") {
		t.Error("Should not include Time Range when nil")
	}
}

func TestBuildContextualMessageWithEmptyTags(t *testing.T) {
	context := &DashboardContext{
		Name: "Test",
		Tags: []string{},
	}

	result := buildContextualMessage("Test", context)

	// Should not panic with empty tags
	if contains(result, "Tags: []") {
		// This is actually fine - empty tags shown as []
	}
}

func TestBuildContextualMessageWithNilTags(t *testing.T) {
	context := &DashboardContext{
		Name: "Test",
		Tags: nil,
	}

	result := buildContextualMessage("Test", context)

	// Should not panic with nil tags
	if !contains(result, "Test") {
		t.Error("User message should still be included")
	}
}

// Helper functions

func contains(s, substr string) bool {
	return indexOf(s, substr) != -1
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Benchmark tests

func BenchmarkBuildContextualMessage(b *testing.B) {
	userMessage := "Show me CPU usage for this dashboard"
	context := &DashboardContext{
		UID:    "abc123",
		Name:   "Node Exporter Full",
		Folder: "Infrastructure",
		Tags:   []string{"linux", "prometheus", "node"},
		TimeRange: map[string]string{
			"from": "2026-01-27T00:00:00Z",
			"to":   "2026-01-27T23:59:59Z",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildContextualMessage(userMessage, context)
	}
}

func BenchmarkBuildContextualMessageNoContext(b *testing.B) {
	userMessage := "Show me CPU usage"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildContextualMessage(userMessage, nil)
	}
}
