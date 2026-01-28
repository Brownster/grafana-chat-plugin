# Frontend Testing Roadmap

This document outlines the plan for adding comprehensive frontend tests to the SM3 Chat Plugin.

## Current State

- **Test Framework**: Jest + React Testing Library (configured via `@grafana/create-plugin`)
- **Existing Tests**: None
- **Components to Test**: 3 main components + 1 utility module

---

## Phase 1: Unit Tests for Pure Functions

**Priority: High | Complexity: Low**

Start with pure functions that have no React dependencies. These are easy to test and provide immediate value.

### 1.1 Artifact Parsing (`src/components/Artifact.tsx`)

| Function | Test Cases |
|----------|------------|
| `parseArtifact()` | Valid artifact block, invalid JSON, no artifact block, malformed block |
| `parseArtifacts()` | Multiple artifacts, mixed content, empty content |

```typescript
// Example test file: src/components/Artifact.test.ts
describe('parseArtifact', () => {
  it('extracts valid artifact from content');
  it('returns null for content without artifact');
  it('handles malformed JSON gracefully');
  it('returns remaining content after extraction');
});

describe('parseArtifacts', () => {
  it('extracts multiple artifacts');
  it('preserves non-artifact content');
});
```

---

## Phase 2: Component Unit Tests

**Priority: High | Complexity: Medium**

Test individual components in isolation using mocks for Grafana dependencies.

### 2.1 MarkdownContent Component

```typescript
// src/components/MarkdownContent.test.tsx
describe('MarkdownContent', () => {
  describe('headings', () => {
    it('renders h1-h6 with correct styling');
  });

  describe('lists', () => {
    it('renders numbered lists');
    it('renders bullet lists');
    it('renders nested lists');
  });

  describe('code blocks', () => {
    it('renders fenced code blocks');
    it('applies language class');
  });

  describe('inline formatting', () => {
    it('renders bold text');
    it('renders italic text');
    it('renders inline code');
    it('renders links with target=_blank');
  });
});
```

### 2.1a Markdown Parsing (Component-Level)

Test the `normalizeContent` and inline parsing logic via component behavior:

| Function | Test Cases |
|----------|------------|
| `normalizeContent()` | CRLF to LF, inline lists split to newlines |
| `parseInlineMarkdown()` | Bold, italic, code, links, nested formatting |

### 2.2 Artifact Component

```typescript
// src/components/Artifact.test.tsx
describe('Artifact', () => {
  describe('rendering', () => {
    it('renders nothing for invalid artifact');
    it('renders report type with sections');
    it('renders chart type with correct chart component');
    it('renders table type with columns and rows');
    it('renders metric-cards type');
  });

  describe('toolbar actions', () => {
    it('copies JSON to clipboard');
    it('downloads artifact as JSON file');
    it('expands to modal view');
  });

  describe('MetricCardComponent', () => {
    it('displays label, value, and change indicator');
    it('shows correct trend icon for positive/negative change');
    it('applies color classes correctly');
  });

  describe('ChartComponent', () => {
    it('renders bar chart by default');
    it('renders line chart when specified');
    it('renders pie chart when specified');
    it('renders area chart when specified');
    it('handles empty data gracefully');
  });

  describe('TableComponent', () => {
    it('renders table headers and rows');
    it('applies column alignment');
    it('handles missing data');
  });
});
```

### 2.3 ChatPanel Component

This is the most complex component. Test in layers:

```typescript
// src/components/ChatPanel.test.tsx
describe('ChatPanel', () => {
  describe('initial state', () => {
    it('renders welcome message when no messages');
    it('displays suggestion buttons');
  });

  describe('message input', () => {
    it('updates input value on change');
    it('disables send button when input is empty');
    it('disables send button while loading');
    it('clears input after sending');
  });

  describe('message display', () => {
    it('renders user messages on the right');
    it('renders assistant messages on the left');
    it('shows timestamp on messages');
    it('shows streaming indicator during streaming');
  });

  describe('tool calls', () => {
    it('hides tool calls when showToolCalls is false');
    it('renders tool calls in collapsible section');
    it('displays tool name and output');
  });

  describe('suggestions', () => {
    it('renders suggestion buttons on assistant messages');
    it('populates input when suggestion clicked');
  });

  describe('auto-scroll', () => {
    it('scrolls to bottom when new message added');
  });
});
```

---

## Phase 3: API Utility Tests

**Priority: High | Complexity: Medium**

### 3.1 Chat API (`src/utils/api.ts`)

```typescript
// src/utils/api.test.ts
describe('chatApi', () => {
  describe('stream', () => {
    it('yields token chunks');
    it('yields tool chunks');
    it('yields complete chunk');
    it('handles error chunks');
    it('handles HTTP errors');
    it('handles missing response body');
    it('parses SSE data lines correctly');
    it('handles malformed JSON in SSE data');
  });
});
```

**Mock Strategy**: Use `msw` (Mock Service Worker) or manual fetch mocking to simulate SSE responses.

---

## Phase 4: Integration Tests

**Priority: Medium | Complexity: High**

Test component interactions and data flow.

### 4.1 Chat Flow Integration

```typescript
// src/integration/ChatFlow.test.tsx
describe('Chat Flow', () => {
  it('sends message and displays streamed response');
  it('displays tool calls during response');
  it('renders artifacts in assistant messages');
  it('handles error responses gracefully');
  it('maintains session across messages');
});
```

### 4.2 Dashboard Context Integration

```typescript
describe('Dashboard Context', () => {
  it('extracts dashboard UID from URL');
  it('fetches dashboard metadata from Grafana API');
  it('includes context in chat requests');
});
```

---

## Phase 5: E2E Tests (Playwright)

**Priority: Low | Complexity: High**

The project already has Playwright configured. Add E2E tests for critical user journeys.

### 5.1 Test Scenarios

```typescript
// e2e/chat.spec.ts
test.describe('Chat Panel E2E', () => {
  test('user can send a message and receive a response');
  test('user can click suggestion to populate input');
  test('user can expand artifact to full screen');
  test('user can copy artifact JSON');
  test('tool calls are collapsible');
});
```

---

## Test Infrastructure Setup

### Required Dependencies

Already installed via `@grafana/create-plugin`:
- `jest`
- `@testing-library/react`
- `@testing-library/jest-dom`
- `jest-environment-jsdom`

May need to add:
```bash
npm install -D @testing-library/user-event msw
```

### Mock Setup

Create mocks for Grafana dependencies. For Jest to auto-pick them up, put them under `<rootDir>/__mocks__` (not `src/__mocks__`) or use explicit `jest.mock(...)` factories.

```typescript
// __mocks__/@grafana/runtime.ts
export const getBackendSrv = jest.fn(() => ({
  get: jest.fn(),
  post: jest.fn(),
}));

export const getTemplateSrv = jest.fn(() => ({
  replace: jest.fn((str) => str),
}));
```

```typescript
// src/__mocks__/fetch.ts
// Mock for SSE streaming tests
export function createMockSSEResponse(chunks: string[]) {
  // Implementation for mocking ReadableStream
}
```

---

## Test File Structure

```
src/
├── components/
│   ├── Artifact.tsx
│   ├── Artifact.test.tsx        # Unit tests
│   ├── ChatPanel.tsx
│   ├── ChatPanel.test.tsx       # Unit tests
│   ├── MarkdownContent.tsx
│   └── MarkdownContent.test.tsx # Unit tests
├── utils/
│   ├── api.ts
│   └── api.test.ts              # Unit tests
├── __mocks__/
│   └── @grafana/
│       └── runtime.ts           # Grafana mocks
└── integration/
    └── ChatFlow.test.tsx        # Integration tests

e2e/
└── chat.spec.ts                 # Playwright E2E tests
```

---

## Coverage Goals

| Phase | Coverage Target | Metric |
|-------|-----------------|--------|
| Phase 1 | 100% | Pure function coverage |
| Phase 2 | 80% | Component statement coverage |
| Phase 3 | 90% | API utility coverage |
| Phase 4 | Key flows | Integration test pass rate |
| Phase 5 | Critical paths | E2E test pass rate |

---

## Implementation Order

1. **Week 1**: Phase 1 (pure functions) + test infrastructure setup
2. **Week 2**: Phase 2.1-2.2 (MarkdownContent, Artifact components)
3. **Week 3**: Phase 2.3 (ChatPanel) + Phase 3 (API utilities)
4. **Week 4**: Phase 4 (integration tests)
5. **Ongoing**: Phase 5 (E2E tests as features stabilize)

---

## Success Criteria

- [ ] All unit tests pass in CI (`npm run test:ci`)
- [ ] Coverage thresholds enforced in root `jest.config.js` (extend the scaffolded config, do not edit `.config/jest.config.js`)
- [ ] No regressions on PR merges
- [ ] E2E tests run on staging deployments

---

## Notes

- Start with the simplest tests (pure functions) to build momentum
- Use snapshot tests sparingly - prefer explicit assertions
- Mock external dependencies at the boundary (Grafana APIs, fetch)
- Keep tests focused - one behavior per test
- Name tests to describe the expected behavior, not implementation
