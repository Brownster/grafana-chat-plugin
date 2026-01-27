package agent

import (
	"sync"
	"testing"
)

func TestNewConversationMemory(t *testing.T) {
	memory := NewConversationMemory()
	if memory == nil {
		t.Fatal("NewConversationMemory() returned nil")
	}

	messages := memory.GetMessages()
	if len(messages) != 0 {
		t.Errorf("NewConversationMemory() should start with 0 messages, got %d", len(messages))
	}
}

func TestAddMessage(t *testing.T) {
	memory := NewConversationMemory()

	// Add user message
	memory.AddMessage("user", "Hello")
	messages := memory.GetMessages()
	if len(messages) != 1 {
		t.Errorf("After AddMessage, got %d messages, want 1", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("Message role = %v, want user", messages[0].Role)
	}
	if messages[0].Content != "Hello" {
		t.Errorf("Message content = %v, want Hello", messages[0].Content)
	}

	// Add assistant message
	memory.AddMessage("assistant", "Hi there!")
	messages = memory.GetMessages()
	if len(messages) != 2 {
		t.Errorf("After second AddMessage, got %d messages, want 2", len(messages))
	}
}

func TestGetMessages(t *testing.T) {
	memory := NewConversationMemory()

	memory.AddMessage("user", "Message 1")
	memory.AddMessage("assistant", "Response 1")
	memory.AddMessage("user", "Message 2")

	messages := memory.GetMessages()
	if len(messages) != 3 {
		t.Errorf("GetMessages() returned %d messages, want 3", len(messages))
	}

	// Verify messages are in order
	if messages[0].Content != "Message 1" {
		t.Errorf("First message content = %v, want 'Message 1'", messages[0].Content)
	}
	if messages[1].Content != "Response 1" {
		t.Errorf("Second message content = %v, want 'Response 1'", messages[1].Content)
	}
	if messages[2].Content != "Message 2" {
		t.Errorf("Third message content = %v, want 'Message 2'", messages[2].Content)
	}
}

func TestClear(t *testing.T) {
	memory := NewConversationMemory()

	memory.AddMessage("user", "Message 1")
	memory.AddMessage("assistant", "Response 1")

	if len(memory.GetMessages()) != 2 {
		t.Error("Setup failed: should have 2 messages")
	}

	memory.Clear()

	messages := memory.GetMessages()
	if len(messages) != 0 {
		t.Errorf("After Clear(), got %d messages, want 0", len(messages))
	}
}

func TestConcurrentAccess(t *testing.T) {
	memory := NewConversationMemory()
	var wg sync.WaitGroup

	// Concurrent writes
	numGoroutines := 100
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			memory.AddMessage("user", "Message from goroutine")
		}(i)
	}

	wg.Wait()

	messages := memory.GetMessages()
	if len(messages) != numGoroutines {
		t.Errorf("After concurrent writes, got %d messages, want %d", len(messages), numGoroutines)
	}

	// Concurrent reads
	wg.Add(numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			msgs := memory.GetMessages()
			if len(msgs) != numGoroutines {
				errors <- nil // Signal error without blocking
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check no errors occurred
	for range errors {
		t.Error("Concurrent read returned incorrect message count")
		break
	}
}

func TestGetMessagesReturnsCopy(t *testing.T) {
	memory := NewConversationMemory()
	memory.AddMessage("user", "Original message")

	// Get messages
	messages1 := memory.GetMessages()

	// Modify the returned slice
	if len(messages1) > 0 {
		messages1[0].Content = "Modified"
	}

	// Get messages again - should not be affected by modification
	messages2 := memory.GetMessages()

	if len(messages2) > 0 && messages2[0].Content != "Original message" {
		t.Error("GetMessages() did not return a copy; modifications affected internal state")
	}
}

func BenchmarkAddMessage(b *testing.B) {
	memory := NewConversationMemory()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memory.AddMessage("user", "Test message")
	}
}

func BenchmarkGetMessages(b *testing.B) {
	memory := NewConversationMemory()

	// Add some messages
	for i := 0; i < 100; i++ {
		memory.AddMessage("user", "Test message")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = memory.GetMessages()
	}
}

func BenchmarkConcurrentAddMessage(b *testing.B) {
	memory := NewConversationMemory()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			memory.AddMessage("user", "Test message")
		}
	})
}
