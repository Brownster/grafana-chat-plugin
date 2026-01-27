package mcp

import (
	"encoding/json"
	"fmt"
)

// FormatToolResult formats MCP tool results for LLM consumption
func FormatToolResult(result interface{}) string {
	if result == nil {
		return "No result returned"
	}

	switch v := result.(type) {
	case string:
		return v
	case map[string]interface{}, []interface{}:
		// Pretty print JSON
		jsonBytes, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	default:
		return fmt.Sprintf("%v", v)
	}
}
