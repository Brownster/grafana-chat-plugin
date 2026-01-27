package agent

import (
	"sync"
)

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ConversationMemory stores conversation history for a session
type ConversationMemory struct {
	messages []Message
	mu       sync.RWMutex
}

// NewConversationMemory creates a new conversation memory
func NewConversationMemory() *ConversationMemory {
	return &ConversationMemory{
		messages: make([]Message, 0),
	}
}

// AddMessage adds a message to the conversation history
func (m *ConversationMemory) AddMessage(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, Message{
		Role:    role,
		Content: content,
	})
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

// Clear removes all messages from the conversation
func (m *ConversationMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]Message, 0)
}
