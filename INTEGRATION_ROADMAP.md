# Integration Roadmap: Grafana LLM App Backend + Custom Chat UI

## Overview

Replace the direct OpenAI integration with `llmclient` from Grafana LLM App. The `llmclient` package uses the **same `go-openai` types** the plugin already uses, so the change is mostly swapping the LLM client. The Go backend stays as the orchestrator (tool execution, memory, streaming). The frontend is unchanged.

```
Frontend (ChatPanel - unchanged)
  └── Go Backend (thin orchestrator)
        ├── llmclient → Grafana LLM App → OpenAI/Azure/Anthropic
        ├── MCP Client → AlertManager MCP Server
        └── MCP Client → Genesys MCP Server
```

**Prerequisite**: Grafana LLM App must be installed and configured in Grafana.

---

## Changes

### 1. Add `llmclient` dependency

**File**: `go.mod`

```
go get github.com/grafana/grafana-llm-app/llmclient
```

Remove `github.com/sashabaranov/go-openai` direct dependency (it becomes transitive via llmclient).

### 2. Replace LLM client (`pkg/llm/openai.go`)

**Current**: Creates `go-openai` client directly with OpenAI API key.
**New**: Create `llmclient.LLMProvider` pointing at Grafana LLM App.

Replace the `OpenAIClient` struct to use `llmclient.LLMProvider` internally:

```go
import "github.com/grafana/grafana-llm-app/llmclient"

type LLMClient struct {
    provider llmclient.LLMProvider
}

func NewLLMClient(grafanaURL, grafanaAPIKey string) *LLMClient {
    return &LLMClient{
        provider: llmclient.NewLLMProvider(grafanaURL, grafanaAPIKey),
    }
}
```

- `Chat()` → calls `provider.ChatCompletions()` with `llmclient.ChatCompletionRequest`
- `StreamChat()` → calls `provider.ChatCompletionsStream()`, returns same `go-openai` stream types
- Model: use `llmclient.ModelLarge` instead of hardcoded `openai.GPT4`
- Tool calls: work identically since `llmclient.ChatCompletionRequest` embeds `openai.ChatCompletionRequest`

### 3. Update settings (`pkg/plugin/settings.go`)

**Remove**: `OpenAIAPIKey` field
**Add**: `GrafanaURL` and `GrafanaAPIKey` fields (for llmclient auth)

```go
type PluginSettings struct {
    GrafanaURL         string  // e.g. "http://localhost:3000"
    GrafanaAPIKey      string  // Grafana service account token
    GrafanaMCPURL      string  // existing
    AlertManagerMCPURL string  // existing
    GenesysMCPURL      string  // existing
}
```

Note: `GrafanaAPIKey` is a Grafana service account token, NOT an LLM provider key. The LLM provider keys are managed by Grafana LLM App's own config.

### 4. Update agent manager (`pkg/agent/manager.go`)

- Change `openaiClient` field type from `*llm.OpenAIClient` to `*llm.LLMClient`
- Update `NewManager()` to accept `*llm.LLMClient`
- Model references: `llmclient.ModelLarge` instead of `openai.GPT4`
- Everything else stays the same (memory, tools, streaming, prompts)

### 5. Update plugin initialization (`pkg/plugin/plugin.go`)

In `getInstance()`:
- Create `llm.NewLLMClient(settings.GrafanaURL, settings.GrafanaAPIKey)` instead of `llm.NewOpenAIClient(settings.OpenAIAPIKey)`
- Pass to agent manager as before

### 6. Update health check (`pkg/plugin/plugin.go`)

Add LLM provider health check using `provider.Enabled()`:
```go
enabled, err := instance.llmClient.Enabled(ctx)
```

### 7. Frontend changes

**None required.** The frontend calls the same Go backend endpoints (`/chat-stream`, `/chat`, `/health`). The backend change is transparent.

### 8. Update provisioning / configuration

Update any provisioning YAML or env vars:
- Remove: `OPENAI_API_KEY`
- Add: `GRAFANA_URL`, `GRAFANA_API_KEY` (service account token)

---

## Files Modified

| File | Change |
|------|--------|
| `go.mod` / `go.sum` | Add llmclient, remove direct go-openai |
| `pkg/llm/openai.go` | Replace OpenAI client with llmclient wrapper |
| `pkg/plugin/settings.go` | Swap OpenAI key for Grafana URL + API key |
| `pkg/plugin/plugin.go` | Update initialization + health check |
| `pkg/agent/manager.go` | Update LLM client type + model references |

## Files Unchanged

| File | Reason |
|------|--------|
| `pkg/mcp/*` | MCP clients stay the same |
| `pkg/agent/memory.go` | Memory management stays the same |
| `pkg/agent/prompts.go` | System prompts stay the same |
| `pkg/plugin/streaming.go` | SSE streaming stays the same (go-openai stream types unchanged) |
| `pkg/plugin/resources.go` | HTTP handlers stay the same |
| `src/**` | Frontend completely unchanged |

---

## Verification

1. `go build ./pkg/...` — compiles without errors
2. `go test ./pkg/...` — existing Go tests pass
3. Manual test: start Grafana with LLM App configured, send a chat message, verify streaming response
4. Verify tool calls work (AlertManager/Genesys MCP servers running)
5. Verify health endpoint returns LLM provider status

---

## Future: Sidebar Option

When ready to add sidebar access, convert from panel plugin to app plugin:
- Change `plugin.json` type from `"panel"` to `"app"`
- Add `includes` array with panel component
- Add app pages (sidebar routes)
- Reuse the same ChatPanel React component in both contexts
- Same Go backend serves both
