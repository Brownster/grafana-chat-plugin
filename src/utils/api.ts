import { getBackendSrv } from '@grafana/runtime';
import type { ChatRequest, StreamChunk } from '../types';

const API_PATH = '/api/plugins/sabio-sm3-chat-plugin/resources';

export const chatApi = {
  stream: async function* (request: ChatRequest): AsyncGenerator<StreamChunk> {
    const backendSrv = getBackendSrv();

    // Use Grafana's backend service to proxy the request to our plugin backend
    const response = await fetch(`${API_PATH}/chat/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const reader = response.body?.getReader();
    if (!reader) {
      throw new Error('No response body');
    }

    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6);
          try {
            const chunk: StreamChunk = JSON.parse(data);
            yield chunk;
          } catch (e) {
            console.error('Failed to parse SSE data:', data);
          }
        }
      }
    }
  },
};
