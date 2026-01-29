# SM3 Monitoring Agent - Grafana Plugin

AI-powered monitoring assistant for Grafana with MCP integration for Grafana, AlertManager, and Genesys Cloud.

## Features

- 🤖 **AI-Powered Chat Interface** - Natural language queries for monitoring data
- 📊 **Rich Visualizations** - Charts, tables, metrics cards, and reports rendered as artifacts
- 🔄 **Real-Time Streaming** - Live token streaming for responsive interactions
- 🔧 **Tool Execution** - Watch as the agent executes tools to gather data
- 📍 **Dashboard Context Awareness** - Automatically includes dashboard metadata in queries
- 🌐 **Multi-MCP Support** - Connects to Grafana, AlertManager, and Genesys Cloud MCP servers

## Architecture

### Backend (Go)
- Plugin SDK integration with Grafana
- MCP client for connecting to external MCP servers
- LLM integration via [Grafana LLM App](https://github.com/grafana/grafana-llm-app) (supports OpenAI, Azure, Anthropic, Grafana-managed)
- Session-based conversation memory
- SSE streaming for real-time responses

### Frontend (React + TypeScript)
- Grafana panel plugin interface
- Real-time message streaming with SSE
- Artifact rendering (charts, tables, reports)
- Markdown content with syntax highlighting
- Tool call visualization

## Installation

### Prerequisites

- Grafana 9.0.0 or later
- Go 1.21 or later
- Node.js 22 or later
- npm 11 or later
- [Grafana LLM App](https://github.com/grafana/grafana-llm-app) installed and configured with an LLM provider
- Grafana service account token (for plugin-to-LLM-App communication)
- Running MCP servers (Grafana, AlertManager, Genesys)

### Build from Source

1. Clone the repository:
```bash
git clone https://github.com/sabio/grafana-sm3-chat-plugin.git
cd grafana-sm3-chat-plugin
```

2. Install dependencies:
```bash
make deps
```

3. Build the plugin:
```bash
make build
```

4. Install to Grafana:
```bash
sudo make install
```

5. Restart Grafana:
```bash
sudo systemctl restart grafana-server
```

### Configuration

#### Plugin Settings

Configure the plugin in Grafana:

1. Navigate to: **Administration → Plugins → SM3 Monitoring Agent → Configuration**

2. Configure the following settings:

**Grafana URL** (Required)
- URL of the Grafana instance (e.g. `http://localhost:3000`)

**Grafana API Key** (Required)
- Service account token for authenticating with the Grafana LLM App
- Store securely as a secret variable
- The LLM provider (OpenAI, Azure, Anthropic, etc.) is configured in the Grafana LLM App itself

**MCP Server URLs** (At least one required)
- **Grafana MCP**: `http://grafana-mcp:8888`
- **AlertManager MCP**: `http://alertmanager-mcp:9300`
- **Genesys MCP**: `http://genesys-mcp:9400`

Example configuration JSON:
```json
{
  "grafana_url": "http://localhost:3000",
  "grafana_api_key": "${GRAFANA_API_KEY}",
  "grafana_mcp_url": "http://grafana-mcp:8888",
  "alertmanager_mcp_url": "http://alertmanager-mcp:9300",
  "genesys_mcp_url": "http://genesys-mcp:9400"
}
```

## Usage

### Adding to Dashboards

1. Edit or create a dashboard
2. Click **Add panel**
3. Select **SM3 Monitoring Agent** from the visualization dropdown
4. Resize panel (recommended: 400-500px width for sidebar layout)
5. Configure panel options:
   - **Show Tool Calls**: Toggle visibility of tool execution details

### Dashboard Context

The plugin automatically extracts and injects dashboard context into queries:

- Dashboard name and UID
- Current folder and tags
- Active time range

Example context injection:
```
[Dashboard Context]
Name: Node Exporter Full
UID: rYdddlPWk
Folder: Infrastructure
Tags: [linux, prometheus, node]
Time Range: 2026-01-27T08:00:00Z to 2026-01-27T10:00:00Z

Show me CPU usage for this dashboard
```

### Example Queries

**Grafana Queries:**
- "Show me all dashboards tagged with 'kubernetes'"
- "What panels are in this dashboard?"
- "Query Prometheus for node_cpu_seconds_total"

**AlertManager Queries:**
- "Show me all active critical alerts"
- "List silences that expire today"
- "Create a silence for maintenance window"

**Genesys Cloud Queries:**
- "Show me queue statistics for the last hour"
- "Which agents are currently on call?"
- "Analyze conversation volumes by queue"

### Artifacts

The agent can render rich visualizations as artifacts:

**Report Artifacts:**
```artifact
{
  "type": "report",
  "title": "System Health Report",
  "sections": [
    { "type": "summary", "title": "Overview", "content": "..." },
    { "type": "metrics", "metrics": [...] },
    { "type": "chart", "chartType": "bar", "data": [...] },
    { "type": "table", "columns": [...], "rows": [...] }
  ]
}
```

**Supported Artifact Types:**
- `report` - Multi-section reports with mixed content
- `chart` - Standalone charts (bar, line, pie, area)
- `table` - Data tables with sortable columns
- `metric-cards` - Grid of metric cards with trends

## MCP Servers

The plugin connects to external MCP servers for tool execution. Two custom servers are included in the `mcp_servers/` directory.

### AlertManager MCP Server

Provides tools for querying and managing Prometheus AlertManager: alerts, silences, receivers, and alert groups with smart pagination.

```bash
cd mcp_servers/alertmanager-mcp-go

# Configure
cp .env.sample .env
# Edit .env:
#   ALERTMANAGER_URL=http://your-alertmanager:9093
#   MCP_TRANSPORT=sse
#   MCP_PORT=9300

# Build and run
make build
make run-sse
```

**Available tools:** `get_status`, `get_alerts`, `get_alert_groups`, `get_silences`, `post_silence`, `delete_silence`, `post_alerts`, `get_receivers`

**Required permissions:** Access to AlertManager API v2.

See [mcp_servers/alertmanager-mcp-go/README.md](mcp_servers/alertmanager-mcp-go/README.md) for full documentation.

### Genesys Cloud MCP Server

Provides tools for querying Genesys Cloud Platform: queue management, conversation analytics, and OAuth client management.

```bash
cd mcp_servers/genesys-cloud-mcp-go

# Configure
cp .env.sample .env
# Edit .env:
#   GENESYSCLOUD_REGION=mypurecloud.com
#   GENESYSCLOUD_OAUTHCLIENT_ID=your-client-id
#   GENESYSCLOUD_OAUTHCLIENT_SECRET=your-secret
#   MCP_TRANSPORT=sse
#   MCP_PORT=9400

# Build and run
make build
make run-sse
```

**Available tools:** `search_queues`, `query_queue_volumes`, `sample_conversations_by_queue`, `search_voice_conversations`, `oauth_clients`

**Required Genesys Cloud permissions:**
- `routing:queue:view`
- `analytics:conversationDetail:view`
- `analytics:conversationAggregate:view`
- `oauth:client:view`

See [mcp_servers/genesys-cloud-mcp-go/README.md](mcp_servers/genesys-cloud-mcp-go/README.md) for full documentation including Docker deployment and multi-region support.

### Connecting MCP Servers to the Plugin

Once the MCP servers are running in SSE mode, configure their URLs in the plugin settings:

```json
{
  "grafana_mcp_url": "http://localhost:8888",
  "alertmanager_mcp_url": "http://localhost:9300",
  "genesys_mcp_url": "http://localhost:9400"
}
```

The plugin will connect to each server on startup, discover available tools, and make them available to the LLM for tool calling.

---

## Development

### Project Structure

```
grafana-sm3-chat-plugin/
├── pkg/
│   ├── agent/         # Agent manager, memory, prompts
│   ├── llm/           # LLM client (via Grafana LLM App)
│   ├── mcp/           # MCP client and tool execution
│   └── plugin/        # Grafana plugin core, HTTP handlers
├── src/
│   ├── components/    # React components (ChatPanel, Artifact, Markdown)
│   ├── services/      # API client
│   └── types.ts       # TypeScript type definitions
├── mcp_servers/
│   ├── alertmanager-mcp-go/   # AlertManager MCP server
│   └── genesys-cloud-mcp-go/  # Genesys Cloud MCP server
├── src/plugin.json    # Plugin metadata
├── Magefile.go        # Go build configuration
├── package.json       # Frontend dependencies
└── README.md          # This file
```

### Development Workflow

1. **Backend development:**
```bash
# Run Go tests
go test ./pkg/... -v

# Build backend
make build-backend
```

2. **Frontend development:**
```bash
# Start watch mode
make dev

# Build frontend
make build-frontend
```

3. **Test in Grafana:**
```bash
# Install and restart
make install
sudo systemctl restart grafana-server
```

### Adding New Features

**Add a new MCP server:**
1. Update `PluginSettings` in `pkg/plugin/settings.go`
2. Add connection logic in `pkg/plugin/plugin.go`
3. Update `BuildSystemPrompt` in `pkg/agent/prompts.go`

**Add new artifact types:**
1. Update `Artifact.tsx` component
2. Add rendering logic for new type
3. Document in system prompt

## Troubleshooting

### Plugin doesn't appear in Grafana

1. Check plugin is in correct directory:
```bash
ls -la /var/lib/grafana/plugins/sabio-sm3-chat-plugin/
```

2. Check Grafana logs:
```bash
tail -f /var/log/grafana/grafana.log
```

3. Verify plugin.json is valid:
```bash
cat /var/lib/grafana/plugins/sabio-sm3-chat-plugin/plugin.json | jq .
```

### MCP connection failures

1. Check MCP server health:
```bash
curl http://grafana-mcp:8888/health
curl http://alertmanager-mcp:9300/health
curl http://genesys-mcp:9400/health
```

2. Verify network connectivity from Grafana container/host

3. Check plugin backend logs in Grafana

### Streaming not working

1. Verify SSE headers in browser DevTools Network tab
2. Check Grafana LLM App is enabled and configured (`GET /api/plugins/grafana-llm-app/health`)
3. Verify service account token is valid
4. Ensure no proxy blocking SSE connections

## API Reference

### Backend Endpoints

**POST /api/plugins/sabio-sm3-chat-plugin/resources/chat**
- Non-streaming chat endpoint
- Request: `ChatRequest` JSON
- Response: `ChatResponse` JSON

**POST /api/plugins/sabio-sm3-chat-plugin/resources/chat-stream**
- Streaming chat endpoint with SSE
- Request: `ChatRequest` JSON
- Response: SSE stream of `StreamChunk` events

**GET /api/plugins/sabio-sm3-chat-plugin/resources/health**
- Health check endpoint
- Response: `{ status: string, llm_provider: { ok: boolean }, mcp_servers: Record<string, { ok: boolean }> }`

### TypeScript Types

```typescript
interface ChatRequest {
  message: string;
  session_id?: string;
  dashboard_context?: DashboardContext;
}

interface StreamChunk {
  type: 'start' | 'token' | 'tool' | 'error' | 'complete' | 'done';
  message?: string;
  tool?: string;
  arguments?: Record<string, any>;
  result?: any;
}
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

Apache-2.0

## Support

- GitHub Issues: https://github.com/sabio/grafana-sm3-chat-plugin/issues
- Documentation: https://github.com/sabio/grafana-sm3-chat-plugin/wiki
- Email: support@sabio.co.uk
