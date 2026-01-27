import React, { useState, useRef, useEffect, useCallback } from 'react';
import { PanelProps } from '@grafana/data';
import { getBackendSrv, getTemplateSrv } from '@grafana/runtime';
import { Send, Loader2, Wrench } from 'lucide-react';
import { chatApi } from '../utils/api';
import { MarkdownContent } from './MarkdownContent';
import { Artifact, parseArtifacts } from './Artifact';
import type { PanelOptions, Message, ToolCall, DashboardContext } from '../types';

interface ChatPanelProps extends PanelProps<PanelOptions> {}

export function ChatPanel(props: ChatPanelProps) {
  const { options, data, width, height, timeRange } = props;
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId] = useState(() => `session-${Date.now()}`);
  const [dashboardContext, setDashboardContext] = useState<DashboardContext | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Add CSS animations to the document
  useEffect(() => {
    const styleId = 'chat-panel-animations';
    if (!document.getElementById(styleId)) {
      const style = document.createElement('style');
      style.id = styleId;
      style.textContent = `
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.5; }
        }
      `;
      document.head.appendChild(style);
    }
  }, []);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Extract dashboard context from Grafana
  useEffect(() => {
    const extractDashboardContext = async () => {
      try {
        // Extract dashboard UID from URL
        const url = window.location.href;
        const dashboardUidMatch = url.match(/\/d\/([^/]+)/);
        if (!dashboardUidMatch) {
          console.warn('Could not extract dashboard UID from URL');
          return;
        }

        const dashboardUid = dashboardUidMatch[1];

        // Fetch dashboard metadata from Grafana API
        const backendSrv = getBackendSrv();
        const dashboard = await backendSrv.get(`/api/dashboards/uid/${dashboardUid}`);

        if (dashboard && dashboard.dashboard) {
          const context: DashboardContext = {
            uid: dashboardUid,
            name: dashboard.dashboard.title || '',
            folder: dashboard.meta?.folderTitle || '',
            tags: dashboard.dashboard.tags || [],
            time_range: {
              from: timeRange.from.toISOString(),
              to: timeRange.to.toISOString(),
            },
          };
          setDashboardContext(context);
          console.log('Dashboard context extracted:', context);
        }
      } catch (error) {
        console.error('Error extracting dashboard context:', error);
      }
    };

    extractDashboardContext();
  }, [timeRange]);

  const sendMessage = useCallback(
    async (messageText: string) => {
      if (!messageText.trim() || isLoading) return;

      const userMessage: Message = {
        id: Date.now().toString(),
        role: 'user',
        content: messageText,
        timestamp: new Date(),
      };

      setMessages((prev) => [...prev, userMessage]);
      setInput('');
      setIsLoading(true);

      try {
        // Create assistant message placeholder
        const assistantMessageId = `${Date.now()}-assistant`;
        const assistantMessage: Message = {
          id: assistantMessageId,
          role: 'assistant',
          content: '',
          timestamp: new Date(),
          isStreaming: true,
          toolCalls: [],
          suggestions: [],
        };

        setMessages((prev) => [...prev, assistantMessage]);

        let accumulatedContent = '';
        let toolCalls: ToolCall[] = [];
        let suggestions: string[] = [];

        // Stream the response
        for await (const chunk of chatApi.stream({
          message: messageText,
          session_id: sessionId,
          dashboard_context: dashboardContext || undefined,
        })) {
          console.log('[DEBUG] Received chunk:', chunk.type, chunk);

          if (chunk.type === 'token' && chunk.message) {
            console.log(
              '[DEBUG] Token chunk length:',
              chunk.message.length,
              'accumulated so far:',
              accumulatedContent.length
            );
            accumulatedContent += chunk.message;
            console.log('[DEBUG] New accumulated length:', accumulatedContent.length);
            setMessages((prev) =>
              prev.map((msg) =>
                msg.id === assistantMessageId ? { ...msg, content: accumulatedContent } : msg
              )
            );
          } else if (chunk.type === 'tool') {
            const toolCall: ToolCall = {
              tool: chunk.tool || 'unknown',
              arguments: chunk.arguments || {},
              output: chunk.result || '',
            };
            toolCalls.push(toolCall);
            setMessages((prev) =>
              prev.map((msg) =>
                msg.id === assistantMessageId ? { ...msg, toolCalls: [...toolCalls] } : msg
              )
            );
          } else if (chunk.type === 'complete') {
            // Only use complete.message as fallback if no tokens were received
            if (chunk.message && accumulatedContent.trim().length === 0) {
              console.log('[DEBUG] No tokens received, using complete.message as fallback');
              accumulatedContent = chunk.message;
              setMessages((prev) =>
                prev.map((msg) =>
                  msg.id === assistantMessageId ? { ...msg, content: accumulatedContent } : msg
                )
              );
            } else {
              console.log('[DEBUG] Keeping accumulated content from tokens:', accumulatedContent.length);
            }
          } else if (chunk.type === 'error') {
            accumulatedContent = `Error: ${chunk.message || 'An error occurred'}`;
            setMessages((prev) =>
              prev.map((msg) =>
                msg.id === assistantMessageId ? { ...msg, content: accumulatedContent } : msg
              )
            );
          }
        }

        // Mark streaming as complete
        console.log('[DEBUG] Streaming complete. Final accumulated content length:', accumulatedContent.length);
        console.log('[DEBUG] Tool calls count:', toolCalls.length);

        setMessages((prev) =>
          prev.map((msg) =>
            msg.id === assistantMessageId ? { ...msg, isStreaming: false, suggestions } : msg
          )
        );
      } catch (error) {
        console.error('Error sending message:', error);
        const errorMessage: Message = {
          id: `${Date.now()}-error`,
          role: 'assistant',
          content: 'Sorry, an error occurred while processing your request. Please try again.',
          timestamp: new Date(),
        };
        setMessages((prev) => [...prev, errorMessage]);
      } finally {
        setIsLoading(false);
      }
    },
    [isLoading, sessionId, dashboardContext]
  );

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isLoading) return;
    await sendMessage(input);
  };

  const handleSuggestionClick = (suggestion: string) => {
    setInput(suggestion);
  };

  // Calculate container height for scrollable area
  const containerHeight = height - 80; // Subtract height for input area

  return (
    <div style={{ width, height, display: 'flex', flexDirection: 'column', backgroundColor: '#111827', color: '#f3f4f6', padding: '16px' }}>
      <div style={{ flex: 1, overflowY: 'auto', marginBottom: '16px', height: containerHeight }}>
        {messages.length === 0 ? (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', textAlign: 'center' }}>
            <div>
              <h2 style={{ fontSize: '24px', fontWeight: 'bold', marginBottom: '8px' }}>
                Dashboard Chat Assistant
              </h2>
              <p style={{ color: '#9ca3af', marginBottom: '24px' }}>
                Ask me anything about this dashboard, metrics, or logs.
              </p>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', justifyContent: 'center', maxWidth: '800px' }}>
                {[
                  'Summarize the key metrics on this dashboard',
                  'What are the trends over the selected time range?',
                  'Are there any anomalies or issues?',
                ].map((suggestion) => (
                  <button
                    key={suggestion}
                    onClick={() => handleSuggestionClick(suggestion)}
                    style={{
                      padding: '8px 16px',
                      backgroundColor: '#1f2937',
                      border: '1px solid #374151',
                      borderRadius: '8px',
                      fontSize: '14px',
                      color: '#f3f4f6',
                      cursor: 'pointer',
                      textAlign: 'left',
                    }}
                  >
                    {suggestion}
                  </button>
                ))}
              </div>
            </div>
          </div>
        ) : (
          messages.map((message) => (
            <div
              key={message.id}
              style={{
                display: 'flex',
                justifyContent: message.role === 'user' ? 'flex-end' : 'flex-start',
                marginBottom: '16px',
              }}
            >
              <div
                style={{
                  maxWidth: '80%',
                  borderRadius: '8px',
                  padding: '12px 16px',
                  backgroundColor: message.role === 'user' ? 'rgba(37, 99, 235, 0.2)' : '#1f2937',
                  color: message.role === 'user' ? '#ffffff' : '#e5e7eb',
                }}
              >
                {message.role === 'user' ? (
                  <p style={{ whiteSpace: 'pre-wrap' }}>{message.content}</p>
                ) : (
                  <>
                    {/* Parse artifacts and render separately */}
                    {(() => {
                      const { artifacts, remainingContent } = parseArtifacts(message.content);
                      return (
                        <>
                          {remainingContent && <MarkdownContent content={remainingContent} />}
                          {artifacts.map((artifact, idx) => (
                            <Artifact
                              key={idx}
                              content={`\`\`\`artifact\n${JSON.stringify(artifact)}\n\`\`\``}
                              className="mt-3"
                            />
                          ))}
                        </>
                      );
                    })()}
                  </>
                )}

                {/* Tool calls */}
                {options.showToolCalls && message.toolCalls && message.toolCalls.length > 0 && (
                  <details style={{ marginTop: '12px', backgroundColor: 'rgba(17, 24, 39, 0.4)', border: '1px solid #374151', borderRadius: '4px' }}>
                    <summary style={{ cursor: 'pointer', padding: '8px 12px', fontSize: '12px', color: '#d1d5db', display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <Wrench style={{ width: '12px', height: '12px', color: '#60a5fa' }} />
                      <span style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                        Tool calls ({message.toolCalls.length})
                      </span>
                    </summary>
                    <div style={{ padding: '0 12px 12px' }}>
                      {message.toolCalls.map((toolCall, idx) => (
                        <div
                          key={idx}
                          style={{
                            backgroundColor: 'rgba(17, 24, 39, 0.5)',
                            border: '1px solid #374151',
                            borderRadius: '4px',
                            padding: '8px',
                            fontSize: '14px',
                            marginTop: '8px',
                          }}
                        >
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: '#60a5fa', marginBottom: '4px' }}>
                            <Wrench style={{ width: '12px', height: '12px' }} />
                            <span style={{ fontFamily: 'monospace' }}>{toolCall.tool}</span>
                          </div>
                          {toolCall.output && (
                            <div style={{ fontSize: '12px', color: '#9ca3af', marginTop: '4px', maxHeight: '128px', overflowY: 'auto' }}>
                              <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                                {typeof toolCall.output === 'string'
                                  ? toolCall.output
                                  : JSON.stringify(toolCall.output, null, 2)}
                              </pre>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  </details>
                )}

                {/* Suggestions */}
                {message.suggestions && message.suggestions.length > 0 && (
                  <div style={{ marginTop: '12px', display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
                    {message.suggestions.map((suggestion, idx) => (
                      <button
                        key={idx}
                        onClick={() => handleSuggestionClick(suggestion)}
                        style={{
                          padding: '6px 12px',
                          backgroundColor: '#374151',
                          color: '#f3f4f6',
                          borderRadius: '4px',
                          fontSize: '12px',
                          cursor: 'pointer',
                          border: 'none',
                        }}
                      >
                        {suggestion}
                      </button>
                    ))}
                  </div>
                )}

                <div style={{ fontSize: '12px', color: '#6b7280', marginTop: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <span>{message.timestamp.toLocaleTimeString()}</span>
                  {message.isStreaming && (
                    <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                      <span style={{ animation: 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite' }}>●</span>
                      <span>streaming</span>
                    </span>
                  )}
                </div>
              </div>
            </div>
          ))
        )}
        {isLoading && messages[messages.length - 1]?.role !== 'assistant' && (
          <div style={{ display: 'flex', justifyContent: 'flex-start' }}>
            <div style={{ backgroundColor: '#1f2937', borderRadius: '8px', padding: '12px 16px' }}>
              <Loader2 style={{ width: '20px', height: '20px', animation: 'spin 1s linear infinite', color: '#60a5fa' }} />
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <form onSubmit={handleSubmit} style={{ display: 'flex', gap: '8px' }}>
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Ask about this dashboard, metrics, or logs..."
          style={{
            flex: 1,
            backgroundColor: '#1f2937',
            border: '1px solid #374151',
            borderRadius: '8px',
            padding: '12px 16px',
            color: '#f3f4f6',
            fontSize: '14px',
            outline: 'none',
          }}
          disabled={isLoading}
        />
        <button
          type="submit"
          disabled={isLoading || !input.trim()}
          style={{
            backgroundColor: isLoading || !input.trim() ? '#374151' : '#2563eb',
            color: '#ffffff',
            borderRadius: '8px',
            padding: '12px 24px',
            cursor: isLoading || !input.trim() ? 'not-allowed' : 'pointer',
            border: 'none',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            fontSize: '14px',
          }}
        >
          <Send style={{ width: '20px', height: '20px' }} />
          <span>Send</span>
        </button>
      </form>
    </div>
  );
}
