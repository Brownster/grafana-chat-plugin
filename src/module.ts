import { PanelPlugin } from '@grafana/data';
import { ChatPanel } from './components/ChatPanel';
import { PanelOptions } from './types';

/**
 * SM3 Monitoring Agent Panel Plugin
 *
 * AI-powered monitoring assistant with Grafana, AlertManager, and Genesys Cloud integration
 */
export const plugin = new PanelPlugin<PanelOptions>(ChatPanel).setPanelOptions((builder) => {
  return builder
    .addBooleanSwitch({
      path: 'showToolCalls',
      name: 'Show Tool Calls',
      description: 'Display tool execution details in the chat interface',
      defaultValue: true,
    });
});
