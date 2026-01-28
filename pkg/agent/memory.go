package agent

import (
	"sync"
)

// Default memory limits
const (
	DefaultMaxMessages   = 100    // Maximum number of messages to retain
	DefaultMaxCharacters = 100000 // Maximum total characters (~25k tokens)
)

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MemoryConfig holds configuration for conversation memory limits
type MemoryConfig struct {
	MaxMessages   int // Maximum number of messages (0 = default, <0 = unlimited)
	MaxCharacters int // Maximum total characters (0 = default, <0 = unlimited)
}

// ConversationMemory stores conversation history for a session
type ConversationMemory struct {
	messages      []Message
	totalChars    int
	maxMessages   int
	maxCharacters int
	mu            sync.RWMutex
}

// NewConversationMemory creates a new conversation memory with default limits
func NewConversationMemory() *ConversationMemory {
	return NewConversationMemoryWithConfig(MemoryConfig{
		MaxMessages:   DefaultMaxMessages,
		MaxCharacters: DefaultMaxCharacters,
	})
}

// NewConversationMemoryWithConfig creates a new conversation memory with custom limits
func NewConversationMemoryWithConfig(config MemoryConfig) *ConversationMemory {
	if config.MaxMessages == 0 {
		config.MaxMessages = DefaultMaxMessages
	}
	if config.MaxCharacters == 0 {
		config.MaxCharacters = DefaultMaxCharacters
	}

	return &ConversationMemory{
		messages:      make([]Message, 0),
		totalChars:    0,
		maxMessages:   config.MaxMessages,
		maxCharacters: config.MaxCharacters,
	}
}

// AddMessage adds a message to the conversation history
// If limits are exceeded, oldest messages are removed to make room
func (m *ConversationMemory) AddMessage(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newMsg := Message{
		Role:    role,
		Content: content,
	}
	msgChars := len(content)

	m.messages = append(m.messages, newMsg)
	m.totalChars += msgChars

	// Trim if over message limit
	m.trimToMessageLimit()

	// Trim if over character limit
	m.trimToCharacterLimit()
}

// trimToMessageLimit removes oldest messages if over the message count limit
// Must be called with lock held
func (m *ConversationMemory) trimToMessageLimit() {
	if m.maxMessages <= 0 {
		return // No limit
	}

	for len(m.messages) > m.maxMessages {
		// Remove oldest message
		removed := m.messages[0]
		m.messages = m.messages[1:]
		m.totalChars -= len(removed.Content)
	}
}

// trimToCharacterLimit removes oldest messages if over the character limit
// Must be called with lock held
func (m *ConversationMemory) trimToCharacterLimit() {
	if m.maxCharacters <= 0 {
		return // No limit
	}

	// Keep at least one message even if it exceeds the limit
	for m.totalChars > m.maxCharacters && len(m.messages) > 1 {
		// Remove oldest message
		removed := m.messages[0]
		m.messages = m.messages[1:]
		m.totalChars -= len(removed.Content)
	}
}

// GetMessages returns all messages in the conversation
func (m *ConversationMemory) GetMessages() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// GetStats returns memory usage statistics
func (m *ConversationMemory) GetStats() MemoryStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MemoryStats{
		MessageCount:  len(m.messages),
		TotalChars:    m.totalChars,
		MaxMessages:   m.maxMessages,
		MaxCharacters: m.maxCharacters,
	}
}

// MemoryStats holds statistics about memory usage
type MemoryStats struct {
	MessageCount  int `json:"message_count"`
	TotalChars    int `json:"total_chars"`
	MaxMessages   int `json:"max_messages"`
	MaxCharacters int `json:"max_characters"`
}

// Clear removes all messages from the conversation
func (m *ConversationMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]Message, 0)
	m.totalChars = 0
}
