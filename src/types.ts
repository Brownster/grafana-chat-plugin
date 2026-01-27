export interface PanelOptions {
  showToolCalls: boolean;
}

export interface ChatRequest {
  message: string;
  session_id?: string;
  dashboard_context?: DashboardContext;
}

export interface DashboardContext {
  uid: string;
  name: string;
  folder: string;
  tags: string[];
  time_range: { from: string; to: string };
}

export interface StreamChunk {
  type: 'start' | 'token' | 'tool' | 'error' | 'complete' | 'done';
  message?: string;
  tool?: string;
  arguments?: Record<string, any>;
  result?: any;
}

export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  toolCalls?: ToolCall[];
  suggestions?: string[];
  isStreaming?: boolean;
}

export interface ToolCall {
  tool: string;
  arguments: Record<string, any>;
  output: string;
}

export interface ChatResponse {
  response: string;
  session_id: string;
}
