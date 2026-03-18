package api

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	ch := make(chan []byte, 1)
	client := &Client{hub: hub, send: ch}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast(Event{Type: "trace:new", Data: map[string]string{"id": "123"}})

	select {
	case msg := <-ch:
		if len(msg) == 0 {
			t.Error("expected non-empty message")
		}
		// Verify the message is valid JSON
		var event Event
		if err := json.Unmarshal(msg, &event); err != nil {
			t.Errorf("failed to unmarshal event: %v", err)
		}
		if event.Type != "trace:new" {
			t.Errorf("expected event type 'trace:new', got %q", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for broadcast")
	}

	hub.Unregister(client)
}

func TestHubEmit(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	ch := make(chan []byte, 1)
	client := &Client{hub: hub, send: ch}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	hub.Emit("test:event", map[string]string{"key": "value"})

	select {
	case msg := <-ch:
		if len(msg) == 0 {
			t.Error("expected non-empty message")
		}
		var event Event
		if err := json.Unmarshal(msg, &event); err != nil {
			t.Errorf("failed to unmarshal event: %v", err)
		}
		if event.Type != "test:event" {
			t.Errorf("expected event type 'test:event', got %q", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for emit")
	}

	hub.Unregister(client)
}

func TestHubMultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	ch1 := make(chan []byte, 1)
	ch2 := make(chan []byte, 1)
	client1 := &Client{hub: hub, send: ch1}
	client2 := &Client{hub: hub, send: ch2}

	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast(Event{Type: "broadcast", Data: "test"})

	// Both clients should receive the message
	select {
	case msg1 := <-ch1:
		if len(msg1) == 0 {
			t.Error("client1: expected non-empty message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client1: timeout waiting for broadcast")
	}

	select {
	case msg2 := <-ch2:
		if len(msg2) == 0 {
			t.Error("client2: expected non-empty message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client2: timeout waiting for broadcast")
	}

	hub.Unregister(client1)
	hub.Unregister(client2)
}

func TestClientUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	ch := make(chan []byte, 1)
	client := &Client{hub: hub, send: ch}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Verify client is registered by sending a message
	hub.Broadcast(Event{Type: "test", Data: nil})
	select {
	case <-ch:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("expected message before unregister")
	}

	// Unregister and verify no more messages are sent
	hub.Unregister(client)
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast(Event{Type: "test2", Data: nil})
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("should not receive message after unregister")
		}
		// Channel is closed, which is expected
	case <-time.After(100 * time.Millisecond):
		// No message received, which is expected
	}
}
