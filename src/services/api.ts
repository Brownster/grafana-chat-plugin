import { ChatRequest, StreamChunk } from '../types';

const PLUGIN_ID = 'sabio-sm3-chat-plugin';

/**
 * API client for SM3 chat plugin backend
 */
export const chatApi = {
  /**
   * Stream chat responses from the backend using Server-Sent Events
   */
  stream: async function* (request: ChatRequest): AsyncGenerator<StreamChunk> {
    const url = `/api/plugins/${PLUGIN_ID}/resources/chat-stream`;

    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const reader = response.body?.getReader();
    if (!reader) {
      throw new Error('Response body is not readable');
    }

    const decoder = new TextDecoder();
    let buffer = '';

    try {
      while (true) {
        const { done, value } = await reader.read();

        if (done) {
          break;
        }

        // Decode the chunk and add to buffer
        buffer += decoder.decode(value, { stream: true });

        // Process complete lines
        const lines = buffer.split('\n');
        buffer = lines.pop() || ''; // Keep incomplete line in buffer

        for (const line of lines) {
          // SSE format: "data: {json}\n"
          if (line.startsWith('data: ')) {
            const data = line.slice(6); // Remove "data: " prefix

            if (data.trim()) {
              try {
                const chunk: StreamChunk = JSON.parse(data);
                yield chunk;
              } catch (e) {
                console.error('Failed to parse SSE chunk:', data, e);
              }
            }
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  },

  /**
   * Send a non-streaming chat message (fallback)
   */
  chat: async (request: ChatRequest): Promise<{ response: string; session_id: string }> => {
    const url = `/api/plugins/${PLUGIN_ID}/resources/chat`;

    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    return response.json();
  },

  /**
   * Check backend health
   */
  health: async (): Promise<{ status: string; mcp_servers: Record<string, boolean> }> => {
    const url = `/api/plugins/${PLUGIN_ID}/resources/health`;

    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    return response.json();
  },
};
